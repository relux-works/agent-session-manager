package merkleinventory

import (
	"encoding/base64"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

func TestDispatchRejectsEveryNonExactOperationBodyShape(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	shapes := []struct {
		operation string
		members   map[string]string
	}{
		{operation: "inventory.roots", members: map[string]string{"namespaces": `["record"]`}},
		{operation: "inventory.children", members: map[string]string{"namespace": `"record"`, "prefix": `""`}},
		{operation: "objects.get", members: map[string]string{"object_ids": `["` + zeroDigest + `"]`, "encodings": `["json"]`}},
	}
	for _, shape := range shapes {
		shape := shape
		operationName := strings.ReplaceAll(shape.operation, ".", "_")
		for _, field := range sortedRawMemberNames(shape.members) {
			valid := shape.members[field]
			for _, variant := range []string{"casefold", "missing", "extra", "duplicate"} {
				variant := variant
				t.Run(operationName+"_"+field+"_"+variant, func(t *testing.T) {
					body := rewriteRawObject(shape.members, field, variant, valid)
					assertDispatchRejectedBody(t, index, shape.operation, body)
				})
			}
			for _, kind := range jsonKindsForWireTests {
				kind := kind
				t.Run(operationName+"_"+field+"_wrong_"+kind.name, func(t *testing.T) {
					wrong := kind.value
					if kind.name == "string" {
						switch field {
						case "namespaces":
							wrong = `"unknown"`
						case "namespace":
							wrong = `"unknown"`
						case "prefix":
							wrong = `"g"`
						case "object_ids":
							wrong = `"` + zeroDigest + `"`
						case "encodings":
							wrong = `"json"`
						}
					}
					assertDispatchRejectedBody(t, index, shape.operation, rewriteRawObject(shape.members, field, "value", wrong))
				})
			}
		}
	}
}

func TestDispatchRejectsEveryNonExactOuterRequestEnvelopeShape(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]string{
		"protocol":         `"` + rpcwire.Protocol + `"`,
		"protocol_version": `"2.0.0"`,
		"request_id":       `"` + testRequestID + `"`,
		"operation":        `"inventory.roots"`,
		"body":             `{"namespaces":["record"]}`,
	}
	for _, field := range sortedRawMemberNames(valid) {
		value := valid[field]
		for _, variant := range []string{"casefold", "missing", "extra", "duplicate"} {
			variant := variant
			t.Run(field+"_"+variant, func(t *testing.T) {
				line := []byte(rewriteRawObject(valid, field, variant, value))
				assertRawDispatchRejected(t, index, line)
			})
		}
		for _, kind := range jsonKindsForWireTests {
			kind := kind
			t.Run(field+"_wrong_"+kind.name, func(t *testing.T) {
				wrong := kind.value
				if kind.name == "string" {
					switch field {
					case "protocol":
						wrong = `"urn:wrong:protocol"`
					case "protocol_version":
						wrong = `"not-a-version"`
					case "request_id":
						wrong = `"not-a-uuid"`
					case "operation":
						wrong = `"unknown.operation"`
					case "body":
						wrong = `"not-an-object"`
					}
				}
				assertRawDispatchRejected(t, index, []byte(rewriteRawObject(valid, field, "value", wrong)))
			})
		}
	}
}

func TestFetchObjectsRejectsEveryNonExactSuccessWireShape(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	id := membership.ID.String()
	wireObject := map[string]string{
		"object_id":  `"` + id + `"`,
		"media_type": `"application/json"`,
		"encoding":   `"json"`,
		"data":       `"` + base64.RawURLEncoding.EncodeToString(data) + `"`,
	}
	objectsBody := map[string]string{"objects": `[` + renderRawObject(wireObject) + `]`}
	envelope := map[string]string{
		"protocol":         `"` + rpcwire.Protocol + `"`,
		"protocol_version": `"2.0.0"`,
		"request_id":       `"` + testRequestID + `"`,
		"ok":               `true`,
		"body":             renderRawObject(objectsBody),
	}
	errorEnvelope := map[string]string{
		"protocol":         `"` + rpcwire.Protocol + `"`,
		"protocol_version": `"2.0.0"`,
		"request_id":       `"` + testRequestID + `"`,
		"ok":               `false`,
		"error":            `{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"incompatible_protocol","message":"peer refused","exit_code":6,"retryable":false,"details":{}}`,
	}
	for _, shape := range []struct {
		name    string
		members map[string]string
	}{
		{name: "response_envelope", members: envelope},
		{name: "failure_envelope", members: errorEnvelope},
		{name: "objects_body", members: objectsBody},
		{name: "wireobject", members: wireObject},
	} {
		shape := shape
		for _, field := range sortedRawMemberNames(shape.members) {
			valid := shape.members[field]
			for _, variant := range []string{"casefold", "missing", "extra", "duplicate"} {
				variant := variant
				t.Run(shape.name+"_"+field+"_"+variant, func(t *testing.T) {
					mutated := rewriteRawObject(shape.members, field, variant, valid)
					response := renderFetchResponse(shape.name, envelope, objectsBody, mutated)
					assertFetchRefused(t, id, response)
				})
			}
			for _, kind := range jsonKindsForWireTests {
				kind := kind
				t.Run(shape.name+"_"+field+"_wrong_"+kind.name, func(t *testing.T) {
					wrong := kind.value
					if kind.name == "string" {
						wrong = `"wrong-value"`
					}
					mutated := rewriteRawObject(shape.members, field, "value", wrong)
					response := renderFetchResponse(shape.name, envelope, objectsBody, mutated)
					assertFetchRefused(t, id, response)
				})
			}
		}
	}
}

func TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	id := membership.ID.String()
	wireObject := `{"object_id":"` + id + `","media_type":"application/json","encoding":"json","data":"` + base64.RawURLEncoding.EncodeToString(data) + `"}`
	for _, kind := range jsonKindsForWireTests {
		kind := kind
		t.Run("array_item_"+kind.name, func(t *testing.T) {
			body := `{"objects":[` + kind.value + `]}`
			assertFetchRefused(t, id, successResponseForBody(body))
		})
	}
	t.Run("empty_array", func(t *testing.T) {
		assertFetchRefused(t, id, successResponseForBody(`{"objects":[]}`))
	})
	t.Run("two_items", func(t *testing.T) {
		assertFetchRefused(t, id, successResponseForBody(`{"objects":[`+wireObject+`,`+wireObject+`]}`))
	})
}

func TestFetchObjectsRefusesWellFormedRPCFailureEnvelope(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFetchRefused(t, membership.ID.String(), validRPCFailureResponse())
}

func TestFetchObjectsRefusesCBORLabelAfterJSONRequest(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	object := map[string]string{
		"object_id":  `"` + membership.ID.String() + `"`,
		"media_type": `"application/json"`,
		"encoding":   `"cbor"`,
		"data":       `"` + base64.RawURLEncoding.EncodeToString(data) + `"`,
	}
	body := renderRawObject(map[string]string{"objects": `[` + renderRawObject(object) + `]`})
	response := map[string]string{
		"protocol":         `"` + rpcwire.Protocol + `"`,
		"protocol_version": `"2.0.0"`,
		"request_id":       `"` + testRequestID + `"`,
		"ok":               `true`,
		"body":             body,
	}
	objects, err := FetchObjects(&rawResponseSource{build: func(request rpcwire.Request) []byte {
		response["request_id"] = `"` + request.ID() + `"`
		return []byte(renderRawObject(response))
	}}, "record", []string{membership.ID.String()}, rpcwire.MaxLineBytes, func() (string, error) { return testRequestID, nil })
	if objects != nil || err == nil || err.Error() != ErrInvalidObject.Error() || !errors.Is(err, ErrInvalidObject) {
		t.Fatalf("CBOR-labelled response returned %d objects, error=%v; want no objects and invalid inventory object", len(objects), err)
	}
}

func TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings(t *testing.T) {
	scanDir := os.Getenv("MERKLE_INVENTORY_GUARD_SCAN_DIR")
	if scanDir == "" {
		scanDir = "."
	}
	violations, err := wireDecoderViolations(scanDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("wire decoder guard violations: %s", strings.Join(violations, "; "))
	}
}

func wireDecoderViolations(packageDir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(packageDir, "*.go"))
	if err != nil {
		return nil, err
	}
	var violations []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.AllErrors)
		if err != nil {
			return nil, err
		}
		aliases := map[string]bool{}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil || importPath != "encoding/json" {
				continue
			}
			alias := "json"
			if imported.Name != nil {
				alias = imported.Name.Name
			}
			if alias == "." {
				violations = append(violations, path+": dot-imported encoding/json bypasses the strict decoder")
			} else if alias != "_" {
				aliases[alias] = true
			}
		}
		parents := make(map[ast.Node]ast.Node)
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				return false
			}
			if len(stack) > 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			base, ok := selector.X.(*ast.Ident)
			if !ok || !aliases[base.Name] {
				return true
			}
			if selector.Sel.Name == "NewDecoder" {
				position := fileSet.Position(selector.Pos())
				violations = append(violations, fmt.Sprintf("%s:%d: json.NewDecoder bypasses the strict decoder", path, position.Line))
			}
			if selector.Sel.Name == "Unmarshal" {
				allowed := filepath.Base(path) == "strict_json.go" && enclosingFuncName(selector, parents) == "strictJSON"
				call, isCall := parents[selector].(*ast.CallExpr)
				if !isCall || call.Fun != selector || len(call.Args) != 2 {
					allowed = false
				} else if first, ok := call.Args[0].(*ast.Ident); !ok || first.Name != "data" {
					allowed = false
				} else if second, ok := call.Args[1].(*ast.Ident); !ok || second.Name != "destination" {
					allowed = false
				}
				if !allowed {
					position := fileSet.Position(selector.Pos())
					violations = append(violations, fmt.Sprintf("%s:%d: json.Unmarshal bypasses the allowlisted strictJSON destination", path, position.Line))
				}
			}
			return true
		})
		ast.Inspect(file, func(node ast.Node) bool {
			function, ok := node.(*ast.FuncDecl)
			if ok && strings.HasPrefix(function.Name.Name, "raw") {
				position := fileSet.Position(function.Pos())
				violations = append(violations, fmt.Sprintf("%s:%d: lenient raw accessor %q is forbidden", path, position.Line, function.Name.Name))
			}
			return true
		})
	}
	sort.Strings(violations)
	return violations, nil
}

type strictWireJSONKind struct {
	name  string
	value string
}

var jsonKindsForWireTests = []strictWireJSONKind{
	{name: "string", value: `"not-a-value"`},
	{name: "number", value: `1`},
	{name: "boolean", value: `false`},
	{name: "null", value: `null`},
	{name: "array", value: `[]`},
	{name: "object", value: `{}`},
}

func assertDispatchRejectedBody(t *testing.T, index *Index, operation, body string) {
	t.Helper()
	line := rawRPCRequest(t, operation, body)
	context := []DispatchContext(nil)
	if operation == "objects.get" {
		context = append(context, DispatchContext{Namespace: "record", MaxLineBytes: rpcwire.MaxLineBytes})
	}
	response, err := index.Dispatch(line, context...)
	if response != nil || err == nil || err.Error() != invalidRequestLiteral {
		t.Fatalf("Dispatch(%s) body=%q returned body=%t error=%v; want no body and literal %q", operation, body, response != nil, err, invalidRequestLiteral)
	}
}

func assertRawDispatchRejected(t *testing.T, index *Index, line []byte) {
	t.Helper()
	response, err := index.Dispatch(line)
	if response != nil || err == nil || err.Error() != invalidRequestLiteral {
		t.Fatalf("Dispatch(raw envelope) returned body=%t error=%v; want no body and literal %q", response != nil, err, invalidRequestLiteral)
	}
}

func rewriteRawObject(source map[string]string, field, mode, replacement string) string {
	members := make(map[string]string, len(source)+1)
	for key, value := range source {
		members[key] = value
	}
	if mode == "duplicate" {
		return renderRawObjectDuplicate(members, field)
	}
	switch mode {
	case "casefold":
		delete(members, field)
		members[strings.ToUpper(field)] = source[field]
	case "missing":
		delete(members, field)
	case "extra":
		members["extension_member"] = `null`
	case "value":
		members[field] = replacement
	default:
		panic("unknown object rewrite mode " + mode)
	}
	return renderRawObject(members)
}

func sortedRawMemberNames(members map[string]string) []string {
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderRawObject(members map[string]string) string {
	keys := sortedRawMemberNames(members)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, strconv.Quote(key)+":"+members[key])
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func renderRawObjectDuplicate(members map[string]string, duplicate string) string {
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		member := strconv.Quote(key) + ":" + members[key]
		parts = append(parts, member)
		if key == duplicate {
			parts = append(parts, member)
		}
	}
	return "{" + strings.Join(parts, ",") + "}"
}

type rawResponseSource struct {
	build func(rpcwire.Request) []byte
}

func (source *rawResponseSource) ObjectsGet(line []byte, _ string, _ uint64) ([]byte, error) {
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return nil, err
	}
	return source.build(request), nil
}

func renderFetchResponse(shape string, envelope, objectsBody map[string]string, mutated string) string {
	switch shape {
	case "response_envelope", "failure_envelope":
		return mutated
	case "objects_body":
		outer := make(map[string]string, len(envelope))
		for key, value := range envelope {
			outer[key] = value
		}
		outer["body"] = mutated
		return renderRawObject(outer)
	case "wireobject":
		body := make(map[string]string, len(objectsBody))
		for key, value := range objectsBody {
			body[key] = value
		}
		body["objects"] = `[` + mutated + `]`
		outer := make(map[string]string, len(envelope))
		for key, value := range envelope {
			outer[key] = value
		}
		outer["body"] = renderRawObject(body)
		return renderRawObject(outer)
	default:
		panic("unknown fetch response shape " + shape)
	}
}

func enclosingFuncName(node ast.Node, parents map[ast.Node]ast.Node) string {
	for parent := parents[node]; parent != nil; parent = parents[parent] {
		if function, ok := parent.(*ast.FuncDecl); ok {
			return function.Name.Name
		}
	}
	return ""
}

func assertFetchRefused(t *testing.T, id, response string) {
	t.Helper()
	objects, err := FetchObjects(&rawResponseSource{build: func(request rpcwire.Request) []byte {
		return []byte(strings.ReplaceAll(response, `"`+testRequestID+`"`, `"`+request.ID()+`"`))
	}}, "record", []string{id}, rpcwire.MaxLineBytes, func() (string, error) { return testRequestID, nil })
	if objects != nil || err == nil || err.Error() != ErrInvalidObject.Error() || !errors.Is(err, ErrInvalidObject) {
		t.Fatalf("FetchObjects returned %d objects, error=%v; want no objects and literal %q", len(objects), err, ErrInvalidObject.Error())
	}
}

func successResponseForBody(body string) string {
	return `{"protocol":"` + rpcwire.Protocol + `","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":` + body + `}`
}

func validRPCFailureResponse() string {
	return `{"protocol":"` + rpcwire.Protocol + `","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"incompatible_protocol","message":"peer refused","exit_code":6,"retryable":false,"details":{}}}`
}

func TestStrictJSONRejectsStructDestinations(t *testing.T) {
	var destination struct{ Value string }
	err := strictJSON([]byte(`{"Value":"case-folded"}`), &destination)
	if err == nil || !errors.Is(err, errStrictJSON) {
		t.Fatalf("strictJSON struct destination error=%v, want strict JSON refusal", err)
	}
	if destination.Value != "" {
		t.Fatalf("strictJSON partially populated forbidden struct destination: %#v", destination)
	}
}

func TestWireDecoderGuardScansProductionAliases(t *testing.T) {
	// This positive control compiles a small parsed file with a renamed
	// encoding/json import and a function-value binding. The same AST guard used
	// by TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings must catch it.
	directory := t.TempDir()
	probePath := filepath.Join(directory, "aliased.go")
	probe := `package merkleinventory
import jsoncodec "encoding/json"
var decodeJSON = jsoncodec.Unmarshal
`
	if err := os.WriteFile(probePath, []byte(probe), 0600); err != nil {
		t.Fatal(err)
	}
	violations, err := wireDecoderViolations(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 || !strings.Contains(strings.Join(violations, " "), "Unmarshal") {
		t.Fatalf("AST guard missed aliased Unmarshal function value: %v", violations)
	}
}
