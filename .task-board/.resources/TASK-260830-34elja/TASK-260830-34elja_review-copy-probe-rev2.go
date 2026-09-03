package main

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

func mustJSON(e *axerror.Error) string {
	b, err := json.Marshal(e)
	if err != nil {
		return "MARSHAL-ERR:" + err.Error()
	}
	return string(b)
}

func check(name, before, after string) {
	if before == after {
		fmt.Printf("HOLD  %-46s bytes unchanged (%d)\n", name, len(after))
	} else {
		fmt.Printf("LEAK  %-46s BEFORE=%s\n                                                    AFTER =%s\n", name, before, after)
	}
}

func main() {
	// A graph exercising: map-in-map, map-in-slice, slice-in-slice, slice-in-map,
	// at depths 1..3, plus scalars of every admitted kind.
	deepMap := map[string]any{"leaf": "v1"}
	sliceInMap := []any{"a", map[string]any{"inner": "v2"}}
	mapInSlice := map[string]any{"m": "v3"}
	sliceInSlice := []any{[]any{"n1", map[string]any{"deepest": "v4"}}}
	level2 := map[string]any{"d2": deepMap, "s2": sliceInMap}
	original := axerror.Details{
		"context":  level2,
		"arr":      []any{mapInSlice, sliceInSlice},
		"scalar_s": "plain",
		"scalar_b": true,
		"scalar_n": json.Number("42"),
		"scalar_z": nil,
	}

	failure, err := axerror.New(axerror.Spec{
		Version:   axerror.Version("1.3.0"),
		Code:      axerror.Code("not_found"),
		Message:   "probe",
		Retryable: false,
		IDs:       axerror.NoIDs(),
		Details:   original,
	})
	if err != nil {
		fmt.Println("New failed, trying another code:", err)
		return
	}
	base := mustJSON(failure)
	fmt.Println("BASE:", base)

	// ---- INBOUND: mutate every container the caller still holds ----
	original["password"] = "hunter2"
	check("inbound top-level map add", base, mustJSON(failure))
	delete(original, "password")

	deepMap["password"] = "hunter2"
	check("inbound nested map depth3 (via retained ref)", base, mustJSON(failure))
	delete(deepMap, "password")

	sliceInMap[0] = "MUTATED"
	check("inbound slice element depth2", base, mustJSON(failure))
	sliceInMap[0] = "a"

	sliceInMap[1].(map[string]any)["password"] = "x"
	check("inbound map-inside-slice depth3", base, mustJSON(failure))
	delete(sliceInMap[1].(map[string]any), "password")

	mapInSlice["password"] = "x"
	check("inbound map at slice index 0", base, mustJSON(failure))
	delete(mapInSlice, "password")

	sliceInSlice[0].([]any)[0] = "MUT"
	check("inbound slice-in-slice element", base, mustJSON(failure))
	sliceInSlice[0].([]any)[0] = "n1"

	sliceInSlice[0].([]any)[1].(map[string]any)["password"] = "x"
	check("inbound deepest map in slice-in-slice", base, mustJSON(failure))
	delete(sliceInSlice[0].([]any)[1].(map[string]any), "password")

	level2["password"] = "x"
	check("inbound level2 map", base, mustJSON(failure))
	delete(level2, "password")

	// ---- OUTBOUND: mutate everything Detail() hands back ----
	v, _ := failure.Detail("context")
	m := v.(map[string]any)
	m["password"] = "hunter2"
	check("outbound Detail top container", base, mustJSON(failure))

	v2, _ := failure.Detail("context")
	m2 := v2.(map[string]any)
	m2["d2"].(map[string]any)["password"] = "hunter2"
	check("outbound Detail nested map depth3", base, mustJSON(failure))

	v3, _ := failure.Detail("context")
	m3 := v3.(map[string]any)
	m3["s2"].([]any)[1].(map[string]any)["password"] = "x"
	check("outbound Detail map-inside-slice", base, mustJSON(failure))

	v4, _ := failure.Detail("arr")
	a4 := v4.([]any)
	a4[0].(map[string]any)["password"] = "x"
	check("outbound Detail map at slice idx", base, mustJSON(failure))

	v5, _ := failure.Detail("arr")
	a5 := v5.([]any)
	a5[1].([]any)[0].([]any)[1].(map[string]any)["password"] = "x"
	check("outbound Detail slice-in-slice deepest", base, mustJSON(failure))

	v6, _ := failure.Detail("arr")
	v6.([]any)[0] = "REPLACED"
	check("outbound Detail slice element replace", base, mustJSON(failure))

	// DetailKeys aliasing
	k := failure.DetailKeys()
	if len(k) > 0 {
		k[0] = "ZZZ"
	}
	k2 := failure.DetailKeys()
	fmt.Printf("DetailKeys after caller mutation: %v (fresh=%v)\n", k2, k2[0] != "ZZZ")

	// Detail returns independent copies per call?
	c1, _ := failure.Detail("context")
	c2, _ := failure.Detail("context")
	fmt.Printf("Detail() returns distinct allocations per call: %v\n",
		fmt.Sprintf("%p", c1.(map[string]any)) != fmt.Sprintf("%p", c2.(map[string]any)))

	// ---- DECODE path aliasing ----
	decoded, derr := axerror.Decode(axerror.Version("1.3.0"), []byte(base))
	if derr != nil {
		fmt.Println("Decode of our own output FAILED:", derr)
	} else {
		dbase := mustJSON(decoded)
		dv, _ := decoded.Detail("context")
		dv.(map[string]any)["password"] = "x"
		check("decoded object outbound Detail", dbase, mustJSON(decoded))
		fmt.Printf("Decode(own output) round-trips byte-identical: %v\n", dbase == base)
	}

	// ---- can a caller get an int / unsupported type in? ----
	bad := axerror.Details{"count": 7}
	if _, e := axerror.New(axerror.Spec{Version: "1.3.0", Code: "not_found", Message: "m", IDs: axerror.NoIDs(), Details: bad}); e == nil {
		fmt.Println("LEAK  Go int admitted as detail value")
	} else {
		fmt.Println("HOLD  Go int refused:", e)
	}

	// named-type map (Details as a nested value) must be refused, not silently aliased
	named := axerror.Details{"context": axerror.Details{"inner": "x"}}
	if _, e := axerror.New(axerror.Spec{Version: "1.3.0", Code: "not_found", Message: "m", IDs: axerror.NoIDs(), Details: named}); e == nil {
		fmt.Println("LEAK  named map type axerror.Details admitted as nested value (clone default arm would alias it)")
	} else {
		fmt.Println("HOLD  named nested map type refused:", e)
	}
}
