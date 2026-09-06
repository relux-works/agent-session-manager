package secconftest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrashClassifyTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		point string
		want  Outcome
	}{
		{PointPrepareEnter, OutcomeSafeRetry},
		{PointPrepareCommit, OutcomeExplicitRollback},
		{PointCommitEnter, OutcomeExplicitRollback},
		{PointCommitApply, OutcomeRecoverableParked},
		{PointRollbackEnter, OutcomeExplicitRollback},
	}
	for _, kase := range cases {
		if got, err := Classify(kase.point); err != nil || got != kase.want {
			t.Errorf("Classify(%q) = %v, %v; want %v", kase.point, got, err, kase.want)
		}
	}
	if _, err := Classify("CR-NOPE-unknown"); err == nil {
		t.Error("Classify of an unknown point succeeded; a typo would arm nothing and fire nothing")
	}
}

func TestCrashPointFiresOnceWhenArmed(t *testing.T) {
	t.Parallel()
	injector := &Injector{}
	if err := injector.MaybeFail(PointPrepareEnter); err != nil {
		t.Fatalf("disarmed MaybeFail = %v, want nil", err)
	}
	injector.Arm(PointPrepareEnter)
	if !injector.Armed(PointPrepareEnter) {
		t.Fatal("Armed reports false after Arm")
	}
	err := injector.MaybeFail(PointPrepareEnter)
	fault, ok := err.(*Fault)
	if !ok {
		t.Fatalf("armed MaybeFail = %v, want *Fault", err)
	}
	if !errors.Is(err, ErrInjected) {
		t.Fatalf("fault %v does not wrap ErrInjected", err)
	}
	if fault.Point() != PointPrepareEnter || fault.Outcome() != OutcomeSafeRetry {
		t.Fatalf("fault = %q/%q, want %q/safe_retry", fault.Point(), fault.Outcome(), PointPrepareEnter)
	}
	if err := injector.MaybeFail(PointPrepareEnter); err != nil {
		t.Fatalf("consumed arm fired again: %v", err)
	}
	injector.Arm(PointPrepareEnter)
	injector.Disarm()
	if err := injector.MaybeFail(PointPrepareEnter); err != nil {
		t.Fatalf("disarmed point fired: %v", err)
	}
	if err := injector.MaybeFail("CR-NOPE-unknown"); err == nil {
		t.Error("unknown point name is silent; a misspelled arm would never fire")
	}
}

func TestTransactorPrepareCommitRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	transactor := NewTransactor(&Injector{}, root)
	body := []byte(`{"kind":"workspace","id":"op-1"}`)
	prepared, err := transactor.Prepare("op-1", body)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if prepared.Outcome != OutcomeSafeRetry {
		t.Fatalf("prepare outcome = %q, want safe_retry", prepared.Outcome)
	}
	committed, err := transactor.Commit("op-1")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	readBack, err := transactor.CommittedBytes("op-1")
	if err != nil {
		t.Fatalf("CommittedBytes: %v", err)
	}
	if string(readBack) != string(body) {
		t.Fatalf("committed bytes = %q, want %q", readBack, body)
	}
	if committed.BodyDigest != prepared.BodyDigest {
		t.Fatal("commit changed the operation digest")
	}
}

func TestTransactorIdempotentRetry(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	transactor := NewTransactor(&Injector{}, root)
	body := []byte(`{"kind":"provider","id":"op-2"}`)
	first, err := transactor.Prepare("op-2", body)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	second, err := transactor.Prepare("op-2", append([]byte(nil), body...))
	if err != nil {
		t.Fatalf("identical retry Prepare: %v", err)
	}
	if first.OperationID != second.OperationID || first.BodyDigest != second.BodyDigest ||
		first.Outcome != second.Outcome || string(first.Bytes) != string(second.Bytes) {
		t.Fatalf("identical retry receipt differs: %+v vs %+v", first, second)
	}
	if _, err := transactor.Prepare("op-2", []byte(`{"kind":"provider","id":"op-2-changed"}`)); !errors.Is(err, ErrIdempotencyMismatch) {
		t.Fatalf("changed-body retry = %v, want idempotency_mismatch", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("changed retry left %d files, want exactly the one staged journal", len(entries))
	}
}

func TestTransactorCrashOutcomes(t *testing.T) {
	t.Parallel()
	points := []struct {
		point        string
		operation    func(*Transactor) error
		wantOutcome  Outcome
		stagedExists bool
	}{
		{PointPrepareEnter, func(tx *Transactor) error { _, err := tx.Prepare("a", []byte("body-a")); return err }, OutcomeSafeRetry, false},
		{PointPrepareCommit, func(tx *Transactor) error { _, err := tx.Prepare("b", []byte("body-b")); return err }, OutcomeExplicitRollback, false},
		{PointCommitEnter, func(tx *Transactor) error {
			if _, err := tx.Prepare("c", []byte("body-c")); err != nil {
				return err
			}
			_, err := tx.Commit("c")
			return err
		}, OutcomeExplicitRollback, true},
		{PointCommitApply, func(tx *Transactor) error {
			if _, err := tx.Prepare("d", []byte("body-d")); err != nil {
				return err
			}
			_, err := tx.Commit("d")
			return err
		}, OutcomeRecoverableParked, false},
		{PointRollbackEnter, func(tx *Transactor) error {
			if _, err := tx.Prepare("e", []byte("body-e")); err != nil {
				return err
			}
			return tx.Rollback("e")
		}, OutcomeExplicitRollback, true},
	}
	for _, kase := range points {
		t.Run(kase.point, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			injector := &Injector{}
			transactor := NewTransactor(injector, root)
			injector.Arm(kase.point)
			err := kase.operation(transactor)
			if got := injector.RemainingArmed(); len(got) != 0 {
				t.Fatalf("armed point %q was never reached; remaining %q", kase.point, got)
			}
			fault, ok := err.(*Fault)
			if !ok {
				t.Fatalf("armed %q gave %v, want *Fault", kase.point, err)
			}
			if fault.Outcome() != kase.wantOutcome {
				t.Fatalf("outcome = %q, want %q", fault.Outcome(), kase.wantOutcome)
			}
			id := map[string]string{
				PointPrepareEnter: "a", PointPrepareCommit: "b",
				PointCommitEnter: "c", PointCommitApply: "d",
				PointRollbackEnter: "e",
			}[kase.point]
			_, stagedErr := os.Stat(filepath.Join(root, id+".staged"))
			if kase.stagedExists && stagedErr != nil {
				t.Fatalf("fault at %q removed the staged bytes; rollback would have nothing to restore", kase.point)
			}
			if !kase.stagedExists && stagedErr == nil {
				t.Fatalf("fault at %q left staged bytes behind", kase.point)
			}
			if kase.point == PointCommitApply {
				readBack, readErr := transactor.CommittedBytes(id)
				if readErr != nil || string(readBack) != "body-d" {
					t.Fatalf("parked commit lost the durable bytes: %q, %v", readBack, readErr)
				}
			}
		})
	}
}

func TestTransactorRecoversAfterFault(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	injector := &Injector{}
	transactor := NewTransactor(injector, root)
	injector.Arm(PointPrepareCommit)
	if _, err := transactor.Prepare("r", []byte("body-r")); err == nil {
		t.Fatal("armed prepare-commit passed; the fault never fired")
	}
	// The fault consumed its arm and cleaned the staging file, so the
	// identical retry must proceed and commit byte-exact bytes.
	receipt, err := transactor.Prepare("r", []byte("body-r"))
	if err != nil {
		t.Fatalf("retry after fault: %v", err)
	}
	if _, err := transactor.Commit("r"); err != nil {
		t.Fatalf("commit after recovery: %v", err)
	}
	readBack, err := transactor.CommittedBytes("r")
	if err != nil || string(readBack) != "body-r" {
		t.Fatalf("recovered bytes = %q, %v; want the exact staged body", readBack, err)
	}
	_ = receipt
}

func TestTransactorRefusesRollbackPastFinalize(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	transactor := NewTransactor(&Injector{}, root)
	if err := transactor.Rollback("ghost"); err == nil {
		t.Fatal("rollback without prepare succeeded")
	}
	if _, err := transactor.Prepare("f", []byte("body-f")); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := transactor.Rollback("f"); err != nil {
		t.Fatalf("pre-commit rollback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "f.staged")); !os.IsNotExist(err) {
		t.Fatal("rollback left staged bytes behind")
	}
	// The arm this test is named for: a clean Commit finalizes the
	// operation (AC-CLONE-005), so rollback past it must fail and the
	// committed bytes must survive the refused call untouched.
	body := []byte("body-final")
	if _, err := transactor.Prepare("g", body); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if _, err := transactor.Commit("g"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := transactor.Rollback("g"); !errors.Is(err, ErrRollbackPastFinalize) {
		t.Fatalf("post-commit rollback = %v, want %v", err, ErrRollbackPastFinalize)
	}
	readBack, err := transactor.CommittedBytes("g")
	if err != nil || string(readBack) != string(body) {
		t.Fatalf("refused rollback deleted the committed bytes: %q, %v", readBack, err)
	}
}

func TestTransactorCommitRequiresPrepare(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	transactor := NewTransactor(&Injector{}, root)
	// No receipt and no staged file: the refusal must name the
	// missing prepare, not merely inherit a filesystem error.
	if _, err := transactor.Commit("ghost"); err == nil || !strings.Contains(err.Error(), "without prepare") {
		t.Fatalf("commit without prepare = %v, want the without-prepare refusal", err)
	}
	// A planted staged file with no receipt: the filesystem would
	// admit this commit, so only the prepare check refuses it.
	if err := os.WriteFile(filepath.Join(root, "planted.staged"), []byte("x"), 0o600); err != nil {
		t.Fatalf("plant staged file: %v", err)
	}
	if _, err := transactor.Commit("planted"); err == nil || !strings.Contains(err.Error(), "without prepare") {
		t.Fatalf("commit with staged bytes but no prepare = %v, want the without-prepare refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "planted.committed")); !os.IsNotExist(err) {
		t.Fatal("refused commit left committed bytes behind")
	}
}

func TestTransactorCrashedCommitIsFinal(t *testing.T) {
	t.Parallel()
	// C1: the commit-apply fault fires AFTER the durable write, so the
	// operation is committed in every observable way. The leaf rule
	// (stated on Commit, in doc.go, and in the README) is that a
	// crashed commit counts as committed: Rollback afterwards is
	// forbidden with ErrRollbackPastFinalize and the bytes survive.
	// The alternative — admitting the rollback and deleting the
	// parked bytes — walks through both halves of AC-CLONE-005 at
	// once and is the one unacceptable answer.
	root := t.TempDir()
	injector := &Injector{}
	transactor := NewTransactor(injector, root)
	body := []byte("BODY")
	if _, err := transactor.Prepare("x", body); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	injector.Arm(PointCommitApply)
	_, err := transactor.Commit("x")
	fault, ok := err.(*Fault)
	if !ok {
		t.Fatalf("crashed Commit = %v, want *Fault", err)
	}
	if fault.Outcome() != OutcomeRecoverableParked {
		t.Fatalf("crashed Commit outcome = %q, want recoverable_parked_state", fault.Outcome())
	}
	if got := injector.RemainingArmed(); len(got) != 0 {
		t.Fatalf("armed point was never reached; remaining %q", got)
	}
	readBack, err := transactor.CommittedBytes("x")
	if err != nil || string(readBack) != string(body) {
		t.Fatalf("parked commit bytes = %q, %v; want the durable BODY", readBack, err)
	}
	if err := transactor.Rollback("x"); !errors.Is(err, ErrRollbackPastFinalize) {
		t.Fatalf("rollback after crashed commit = %v, want %v", err, ErrRollbackPastFinalize)
	}
	readBack, err = transactor.CommittedBytes("x")
	if err != nil || string(readBack) != string(body) {
		t.Fatalf("refused rollback deleted the parked bytes: %q, %v", readBack, err)
	}
	// The parked operation is terminal for the other phases too.
	if _, err := transactor.Commit("x"); !errors.Is(err, ErrCommitPastFinalize) {
		t.Fatalf("second Commit after parked commit = %v, want %v", err, ErrCommitPastFinalize)
	}
	if _, err := transactor.Prepare("x", body); !errors.Is(err, ErrPreparePastFinalize) {
		t.Fatalf("Prepare after parked commit = %v, want %v", err, ErrPreparePastFinalize)
	}
	readBack, err = transactor.CommittedBytes("x")
	if err != nil || string(readBack) != string(body) {
		t.Fatalf("terminal refusals disturbed the parked bytes: %q, %v", readBack, err)
	}
}

func TestTransactorTerminalNeighbours(t *testing.T) {
	t.Parallel()
	// C1b: every neighbour of the finalize gate names its state
	// through the public API. Two of these previously refused only by
	// inheriting a filesystem ENOENT, the exact shape
	// TestTransactorCommitRequiresPrepare declares unacceptable.
	t.Run("rollback-twice", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		if _, err := transactor.Prepare("rr", []byte("body-rr")); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if err := transactor.Rollback("rr"); err != nil {
			t.Fatalf("first rollback: %v", err)
		}
		if err := transactor.Rollback("rr"); err == nil || !strings.Contains(err.Error(), "without prepare") {
			t.Fatalf("second rollback = %v, want the without-prepare refusal", err)
		}
	})
	t.Run("commit-twice", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		body := []byte("body-cc")
		if _, err := transactor.Prepare("cc", body); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if _, err := transactor.Commit("cc"); err != nil {
			t.Fatalf("first Commit: %v", err)
		}
		second, err := transactor.Commit("cc")
		if !errors.Is(err, ErrCommitPastFinalize) {
			t.Fatalf("second Commit = %v (%+v), want %v", err, second, ErrCommitPastFinalize)
		}
		if second.OperationID != "" || second.BodyDigest != "" || len(second.Bytes) != 0 {
			t.Fatalf("refused second Commit returned a live receipt: %+v", second)
		}
		readBack, err := transactor.CommittedBytes("cc")
		if err != nil || string(readBack) != string(body) {
			t.Fatalf("second Commit disturbed the committed bytes: %q, %v", readBack, err)
		}
	})
	t.Run("commit-after-rollback", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		if _, err := transactor.Prepare("cr", []byte("body-cr")); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if err := transactor.Rollback("cr"); err != nil {
			t.Fatalf("Rollback: %v", err)
		}
		if _, err := transactor.Commit("cr"); err == nil || !strings.Contains(err.Error(), "without prepare") {
			t.Fatalf("Commit after rollback = %v, want the without-prepare refusal", err)
		}
		if _, err := os.Stat(filepath.Join(root, "cr.committed")); !os.IsNotExist(err) {
			t.Fatal("refused Commit after rollback left committed bytes behind")
		}
	})
	t.Run("prepare-after-clean-commit", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		body := []byte("body-pc")
		if _, err := transactor.Prepare("pc", body); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if _, err := transactor.Commit("pc"); err != nil {
			t.Fatalf("Commit: %v", err)
		}
		if _, err := transactor.Prepare("pc", body); !errors.Is(err, ErrPreparePastFinalize) {
			t.Fatalf("Prepare after clean Commit = %v, want %v", err, ErrPreparePastFinalize)
		}
	})
	t.Run("prepare-after-rollback-restages", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		body := []byte("body-re")
		if _, err := transactor.Prepare("re", body); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if err := transactor.Rollback("re"); err != nil {
			t.Fatalf("Rollback: %v", err)
		}
		if _, err := transactor.Prepare("re", body); err != nil {
			t.Fatalf("re-Prepare after rollback (the recovery retry): %v", err)
		}
		if _, err := transactor.Commit("re"); err != nil {
			t.Fatalf("Commit after re-Prepare: %v", err)
		}
		readBack, err := transactor.CommittedBytes("re")
		if err != nil || string(readBack) != string(body) {
			t.Fatalf("recovered bytes = %q, %v", readBack, err)
		}
	})
	t.Run("commit-without-staged-names-state", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		transactor := NewTransactor(&Injector{}, root)
		if _, err := transactor.Prepare("sm", []byte("body-sm")); err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if err := os.Remove(filepath.Join(root, "sm.staged")); err != nil {
			t.Fatalf("remove staged: %v", err)
		}
		_, err := transactor.Commit("sm")
		if err == nil || !strings.Contains(err.Error(), "without staged") {
			t.Fatalf("Commit with receipt but no staged bytes = %v, want the without-staged refusal", err)
		}
	})
}

func TestTransactorRefusesEmptyOperationID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	transactor := NewTransactor(&Injector{}, root)
	if _, err := transactor.Prepare("", []byte("x")); err == nil || !strings.Contains(err.Error(), "empty operation") {
		t.Fatalf("Prepare with empty ID = %v, want the empty-operation refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".staged")); !os.IsNotExist(err) {
		t.Fatal("empty-ID Prepare staged a file")
	}
}

func TestInjectorArmedPointIsConsumedByTheDrive(t *testing.T) {
	t.Parallel()
	injector := &Injector{}
	injector.Arm(PointPrepareEnter)
	injector.Arm(PointCommitApply)
	if got := injector.RemainingArmed(); len(got) != 2 {
		t.Fatalf("RemainingArmed = %q, want the two armed points", got)
	}
	if err := injector.MaybeFail(PointPrepareEnter); err == nil {
		t.Fatal("armed point did not fire")
	}
	if got := injector.RemainingArmed(); len(got) != 1 || got[0] != PointCommitApply {
		t.Fatalf("RemainingArmed = %q, want exactly [%q]", got, PointCommitApply)
	}
	injector.Disarm()
	if got := injector.RemainingArmed(); len(got) != 0 {
		t.Fatalf("RemainingArmed after Disarm = %q, want empty", got)
	}
}

func TestCrashPointVocabularyIsPinned(t *testing.T) {
	t.Parallel()
	// The five point names are instrument-local phase boundaries, not
	// the SPEC registry: CR-MAT-01..08 names nothing here, and a
	// registry-style name must be refused as unknown rather than armed
	// silently. Adding a sixth point without extending this list must
	// fail the Classify arm below only if it is wired; the residual
	// bound (a new site without an arm) is stated in doc.go.
	want := map[string]Outcome{
		PointPrepareEnter:  OutcomeSafeRetry,
		PointPrepareCommit: OutcomeExplicitRollback,
		PointCommitEnter:   OutcomeExplicitRollback,
		PointCommitApply:   OutcomeRecoverableParked,
		PointRollbackEnter: OutcomeExplicitRollback,
	}
	if len(want) != 5 {
		t.Fatalf("vocabulary has %d points, want exactly 5", len(want))
	}
	for point, outcome := range want {
		if got, err := Classify(point); err != nil || got != outcome {
			t.Errorf("Classify(%q) = %v, %v; want %v", point, got, err, outcome)
		}
	}
	for _, registry := range []string{"CR-MAT-01", "CR-MAT-08", "CR-MAT-09", "CR-MAT-prepare"} {
		if _, err := Classify(registry); err == nil {
			t.Errorf("Classify(%q) succeeded; the SPEC registry is not this vocabulary", registry)
		}
	}
}
