package mcpschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// defaultMaxOut is the per-tool output cap the production caller passes
// (aicallhandler's mcpMaxToolSchemaBytes).
const defaultMaxOut = 64 << 10

// normalizeRow is one Test_Normalize_Rules case. in and want are JSON text;
// want "null" means a nil result. check, when set, asserts Report fields the
// common assertions do not cover.
type normalizeRow struct {
	name   string
	in     string
	maxOut int

	want             string
	wantDropped      bool
	wantReason       string
	wantDropPath     string
	wantDroppedProps []string
	check            func(t *testing.T, rep Report)
}

func decodeJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad test JSON: %v\n%s", err, s)
	}
	return m
}

func runNormalizeRow(t *testing.T, tt normalizeRow) {
	t.Helper()
	var in map[string]any
	if tt.in != "null" {
		in = decodeJSON(t, tt.in)
	}
	maxOut := tt.maxOut
	if maxOut == 0 {
		maxOut = defaultMaxOut
	}

	got, rep := Normalize(in, maxOut)

	var want map[string]any
	if tt.want != "null" {
		want = decodeJSON(t, tt.want)
	}
	// Compare through a JSON round trip so number and slice types line up
	// with the decoded expectation.
	var gotRT map[string]any
	if got != nil {
		b, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("could not marshal result: %v", err)
		}
		gotRT = decodeJSON(t, string(b))
	}
	if !reflect.DeepEqual(gotRT, want) {
		gb, _ := json.Marshal(got)
		t.Errorf("Wrong match.\nexpect: %s\ngot:    %s", tt.want, gb)
	}
	if rep.ToolDropped != tt.wantDropped {
		t.Errorf("Wrong match. ToolDropped expect: %v, got: %v (reason %q, path %q)", tt.wantDropped, rep.ToolDropped, rep.DropReason, rep.DropPath)
	}
	if rep.DropReason != tt.wantReason {
		t.Errorf("Wrong match. DropReason expect: %q, got: %q", tt.wantReason, rep.DropReason)
	}
	if rep.DropPath != tt.wantDropPath {
		t.Errorf("Wrong match. DropPath expect: %q, got: %q", tt.wantDropPath, rep.DropPath)
	}
	// A dropped tool's DroppedProps is whatever was collected before the
	// fatal subschema; only kept tools pin it.
	if tt.wantDropped {
		if tt.check != nil {
			tt.check(t, rep)
		}
		return
	}
	if len(rep.DroppedProps) != 0 || len(tt.wantDroppedProps) != 0 {
		if !reflect.DeepEqual(rep.DroppedProps, tt.wantDroppedProps) {
			t.Errorf("Wrong match. DroppedProps expect: %v, got: %v", tt.wantDroppedProps, rep.DroppedProps)
		}
	}
	if rep.DroppedPropsN != len(tt.wantDroppedProps) && tt.check == nil {
		t.Errorf("Wrong match. DroppedPropsN expect: %d, got: %d", len(tt.wantDroppedProps), rep.DroppedPropsN)
	}
	if tt.check != nil {
		tt.check(t, rep)
	}
}

func Test_Normalize_Rules(t *testing.T) {
	tests := []normalizeRow{
		{
			name: "nil input returns nil and an empty report",
			in:   "null",
			want: "null",
			check: func(t *testing.T, rep Report) {
				if !reflect.DeepEqual(rep, Report{}) {
					t.Errorf("Wrong match. expect: empty report, got: %+v", rep)
				}
			},
		},
		{
			name: "keep list strips x-*, title, default, $schema, additionalProperties",
			in: `{"$schema":"http://json-schema.org/draft-07/schema#","title":"T","type":"object","additionalProperties":false,
				"properties":{"owner":{"type":"string","description":"Repository owner","x-mcp-header":"owner","default":"a","title":"Owner"}},
				"required":["owner"]}`,
			want: `{"type":"object","properties":{"owner":{"type":"string","description":"Repository owner"}},"required":["owner"]}`,
			check: func(t *testing.T, rep Report) {
				if rep.DroppedKeys != 6 {
					t.Errorf("Wrong match. DroppedKeys expect: 6, got: %d", rep.DroppedKeys)
				}
			},
		},
		{
			name: "root without type is an object and always emits properties",
			in:   `{}`,
			want: `{"type":"object","properties":{}}`,
		},
		{
			name: "root with only a description is a no-argument object",
			in:   `{"description":"no args"}`,
			want: `{"type":"object","description":"no args","properties":{}}`,
		},
		{
			name: "top-level empty properties kept",
			in:   `{"type":"object","properties":{}}`,
			want: `{"type":"object","properties":{}}`,
		},
		{
			name:        "root of another type drops the tool",
			in:          `{"type":"string"}`,
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonRootNotObject,
		},
		{
			name:        "root list type drops the tool",
			in:          `{"type":["object","null"],"properties":{}}`,
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonRootNotObject,
		},
		{
			name: "root anyOf and oneOf are dropped",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"required":["a"]}],"oneOf":[{"required":["a"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}}}`,
			check: func(t *testing.T, rep Report) {
				if rep.DroppedKeys != 2 {
					t.Errorf("Wrong match. DroppedKeys expect: 2, got: %d", rep.DroppedKeys)
				}
			},
		},

		// R2 value shapes.
		{
			name:             "bad type string makes an optional property unusable",
			in:               `{"type":"object","properties":{"a":{"type":"date"},"b":{"type":"string"}}}`,
			want:             `{"type":"object","properties":{"b":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name:         "bad type on a required property drops the tool",
			in:           `{"type":"object","properties":{"a":{"type":"date"}},"required":["a"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonBadType,
			wantDropPath: "/properties/a",
		},
		{
			name:             "boolean subschema is unusable",
			in:               `{"type":"object","properties":{"a":true,"b":false,"c":{"type":"integer"}}}`,
			want:             `{"type":"object","properties":{"c":{"type":"integer"}}}`,
			wantDroppedProps: []string{"/properties/a", "/properties/b"},
		},
		{
			name:             "tuple items make the array unusable",
			in:               `{"type":"object","properties":{"a":{"type":"array","items":[{"type":"string"}]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name:             "boolean items make the array unusable",
			in:               `{"type":"object","properties":{"a":{"type":"array","items":true}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name: "non-string required members are removed",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"required":["a",1,null,{"x":1}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`,
		},
		{
			name: "non-numeric bounds and bounds on the wrong type are dropped",
			in: `{"type":"object","properties":{
				"n":{"type":"number","minimum":"1","maximum":10},
				"i":{"type":"integer","minimum":-5,"maximum":true},
				"s":{"type":"string","minimum":1,"minItems":1},
				"a":{"type":"array","items":{"type":"string"},"minItems":1.5,"maxItems":3,"minimum":0},
				"b":{"type":"array","items":{"type":"string"},"minItems":-1,"maxItems":"3"}}}`,
			want: `{"type":"object","properties":{
				"n":{"type":"number","maximum":10},
				"i":{"type":"integer","minimum":-5},
				"s":{"type":"string"},
				"a":{"type":"array","items":{"type":"string"},"maxItems":3},
				"b":{"type":"array","items":{"type":"string"}}}}`,
		},
		{
			name: "non-string description is dropped",
			in:   `{"type":"object","description":5,"properties":{"a":{"type":"string","description":["x"]}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}}}`,
		},
		{
			name: "non-object properties are treated as absent",
			in:   `{"type":"object","properties":["a"]}`,
			want: `{"type":"object","properties":{}}`,
		},

		// R3 server-conservative format and enum.
		{
			name: "format allow pairs kept, uri and email dropped",
			in: `{"type":"object","properties":{
				"dt":{"type":"string","format":"date-time"},
				"u":{"type":"string","format":"uri"},
				"e":{"type":"string","format":"email"},
				"d":{"type":"string","format":"date"},
				"i32":{"type":"integer","format":"int32"},
				"i64":{"type":"integer","format":"int64"},
				"iw":{"type":"integer","format":"float"},
				"f":{"type":"number","format":"float"},
				"db":{"type":"number","format":"double"},
				"nw":{"type":"number","format":"int32"},
				"bf":{"type":"boolean","format":"date-time"}}}`,
			want: `{"type":"object","properties":{
				"dt":{"type":"string","format":"date-time"},
				"u":{"type":"string"},
				"e":{"type":"string"},
				"d":{"type":"string"},
				"i32":{"type":"integer","format":"int32"},
				"i64":{"type":"integer","format":"int64"},
				"iw":{"type":"integer"},
				"f":{"type":"number","format":"float"},
				"db":{"type":"number","format":"double"},
				"nw":{"type":"number"},
				"bf":{"type":"boolean"}}}`,
		},
		{
			name: "format enum kept only with an emitted enum",
			in: `{"type":"object","properties":{
				"with":{"type":"string","format":"enum","enum":["a","b"]},
				"without":{"type":"string","format":"enum"},
				"badenum":{"type":"string","format":"enum","enum":["a",1]}}}`,
			want: `{"type":"object","properties":{
				"with":{"type":"string","format":"enum","enum":["a","b"]},
				"without":{"type":"string"},
				"badenum":{"type":"string"}}}`,
		},
		{
			name: "string enum kept with duplicates, integer, mixed and empty enums dropped",
			in: `{"type":"object","properties":{
				"s":{"type":"string","enum":["b","a","b"]},
				"i":{"type":"integer","enum":[1,2]},
				"is":{"type":"integer","enum":["1"]},
				"m":{"type":"string","enum":["a",1]},
				"e":{"type":"string","enum":[]}}}`,
			want: `{"type":"object","properties":{
				"s":{"type":"string","enum":["b","a","b"]},
				"i":{"type":"integer"},
				"is":{"type":"integer"},
				"m":{"type":"string"},
				"e":{"type":"string"}}}`,
		},
		{
			name: "null type is kept bare",
			in:   `{"type":"object","properties":{"n":{"type":"null","description":"only null","enum":["x"]}}}`,
			want: `{"type":"object","properties":{"n":{"type":"null","description":"only null"}}}`,
		},

		// R4 list type.
		{
			name: "list type becomes anyOf with type-appropriate siblings and description on the parent",
			in: `{"type":"object","properties":{"v":{"type":["string","number","boolean"],"description":"Value",
				"enum":["a"],"format":"date-time","minimum":1,"items":{"type":"string"}}}}`,
			want: `{"type":"object","properties":{"v":{"description":"Value","anyOf":[
				{"type":"string","enum":["a"],"format":"date-time"},
				{"type":"number","minimum":1},
				{"type":"boolean"}]}}}`,
			check: func(t *testing.T, rep Report) {
				if rep.Rewrites != 1 {
					t.Errorf("Wrong match. Rewrites expect: 1, got: %d", rep.Rewrites)
				}
			},
		},
		{
			name: "list type with null gives a bare null member",
			in:   `{"type":"object","properties":{"v":{"type":["string","null"],"description":"d","enum":["x"]}}}`,
			want: `{"type":"object","properties":{"v":{"description":"d","anyOf":[{"type":"string","enum":["x"]},{"type":"null"}]}}}`,
		},
		{
			name: "single-element list type is the plain type",
			in:   `{"type":"object","properties":{"v":{"type":["integer"],"minimum":0}}}`,
			want: `{"type":"object","properties":{"v":{"type":"integer","minimum":0}}}`,
		},
		{
			name:             "empty list type is unusable",
			in:               `{"type":"object","properties":{"v":{"type":[]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/v"},
		},
		{
			name:             "list type with a bad member is unusable",
			in:               `{"type":"object","properties":{"v":{"type":["string","date"]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/v"},
		},
		{
			name: "list type member that is unusable is removed",
			in:   `{"type":"object","properties":{"v":{"type":["array","string"]}}}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"string"}]}}}`,
		},
		{
			name:             "list type left with only null is unusable",
			in:               `{"type":"object","properties":{"v":{"type":["array","null"]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/v"},
		},

		// R5 conversions.
		{
			name: "oneOf becomes anyOf",
			in:   `{"type":"object","properties":{"v":{"description":"d","oneOf":[{"type":"string"},{"type":"integer"}]}}}`,
			want: `{"type":"object","properties":{"v":{"description":"d","anyOf":[{"type":"string"},{"type":"integer"}]}}}`,
		},
		{
			name: "oneOf and anyOf are concatenated with anyOf first",
			in:   `{"type":"object","properties":{"v":{"oneOf":[{"type":"string"}],"anyOf":[{"type":"boolean"},{"type":"null"}]}}}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"boolean"},{"type":"null"},{"type":"string"}]}}}`,
		},
		{
			name: "string const becomes a string enum",
			in:   `{"type":"object","properties":{"a":{"const":"fixed"},"b":{"type":"string","const":"x","enum":["y"]}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string","enum":["fixed"]},"b":{"type":"string","enum":["x"]}}}`,
		},
		{
			name: "const with a non-string type is dropped",
			in:   `{"type":"object","properties":{"a":{"type":"integer","const":"1"}}}`,
			want: `{"type":"object","properties":{"a":{"type":"integer"}}}`,
		},
		{
			name:             "non-string const is dropped and leaves a typeless subschema",
			in:               `{"type":"object","properties":{"a":{"const":1},"b":{"type":"integer","const":1}}}`,
			want:             `{"type":"object","properties":{"b":{"type":"integer"}}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name: "single-member allOf is merged with parent keys winning",
			in:   `{"type":"object","properties":{"a":{"description":"outer","allOf":[{"type":"string","description":"inner","enum":["x"]}]}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string","description":"outer","enum":["x"]}}}`,
		},
		{
			name: "empty allOf is ignored",
			in:   `{"type":"object","properties":{"a":{"type":"string","allOf":[]}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}}}`,
		},
		{
			name:             "multi-member allOf on an optional property removes it",
			in:               `{"type":"object","properties":{"a":{"allOf":[{"type":"string"},{"minLength":1}]},"b":{"type":"string"}}}`,
			want:             `{"type":"object","properties":{"b":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name:         "multi-member allOf on a required property drops the tool",
			in:           `{"type":"object","properties":{"a":{"allOf":[{"type":"string"},{"minLength":1}]}},"required":["a"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAllOf,
			wantDropPath: "/properties/a",
		},

		// R6 type inference.
		{
			name: "type inferred from properties, items and a string enum",
			in: `{"type":"object","properties":{
				"o":{"properties":{"x":{"type":"string"}},"required":["x"]},
				"a":{"items":{"type":"integer"}},
				"e":{"enum":["a","b"]}}}`,
			want: `{"type":"object","properties":{
				"o":{"type":"object","properties":{"x":{"type":"string"}},"required":["x"]},
				"a":{"type":"array","items":{"type":"integer"}},
				"e":{"type":"string","enum":["a","b"]}}}`,
			check: func(t *testing.T, rep Report) {
				if rep.Rewrites != 3 {
					t.Errorf("Wrong match. Rewrites expect: 3, got: %d", rep.Rewrites)
				}
			},
		},
		{
			name: "typeless anyOf emits no type",
			in:   `{"type":"object","properties":{"v":{"anyOf":[{"type":"string","minLength":1},{"type":"null"}],"description":"d"}}}`,
			want: `{"type":"object","properties":{"v":{"description":"d","anyOf":[{"type":"string"},{"type":"null"}]}}}`,
		},
		{
			name:             "typeless subschema with nothing to infer is unusable",
			in:               `{"type":"object","properties":{"v":{"description":"any JSON value"},"m":{"enum":["a",1]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/m", "/properties/v"},
		},
		{
			name:         "typeless required property drops the tool",
			in:           `{"type":"object","properties":{"v":{}},"required":["v"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonTypeless,
			wantDropPath: "/properties/v",
		},

		// R7 anyOf cascade.
		{
			name: "anyOf with one bad member keeps the rest, a one-member anyOf is not flattened",
			in:   `{"type":"object","properties":{"v":{"anyOf":[{"description":"typeless"},{"type":"string"}]}}}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"string"}]}}}`,
		},
		{
			name:             "anyOf with only a null member left is unusable",
			in:               `{"type":"object","properties":{"v":{"anyOf":[{"type":"array"},{"type":"null"}]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/v"},
		},
		{
			name:         "required anyOf with no usable member drops the tool",
			in:           `{"type":"object","properties":{"v":{"anyOf":[{"type":"array"}]}},"required":["v"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/v",
		},

		// R7 object cascade and R10 free-form objects.
		{
			name: "nested free-form object",
			in:   `{"type":"object","properties":{"o":{"type":"object","description":"free","additionalProperties":{"type":"string"},"required":["x"]}}}`,
			want: `{"type":"object","properties":{"o":{"type":"object","description":"free"}}}`,
		},
		{
			name: "nested empty properties is stripped",
			in:   `{"type":"object","properties":{"o":{"type":"object","properties":{}}}}`,
			want: `{"type":"object","properties":{"o":{"type":"object"}}}`,
		},
		{
			name:             "object whose properties all cascade away becomes free-form",
			in:               `{"type":"object","properties":{"o":{"type":"object","description":"d","properties":{"x":{},"y":{"type":"array"}}}}}`,
			want:             `{"type":"object","properties":{"o":{"type":"object","description":"d"}}}`,
			wantDroppedProps: []string{"/properties/o/properties/x", "/properties/o/properties/y"},
		},
		{
			name:             "unusable required inside an optional nested object removes the object and keeps the tool",
			in:               `{"type":"object","properties":{"o":{"type":"object","properties":{"x":{},"y":{"type":"string"}},"required":["x"]},"k":{"type":"string"}},"required":["k"]}`,
			want:             `{"type":"object","properties":{"k":{"type":"string"}},"required":["k"]}`,
			wantDroppedProps: []string{"/properties/o"},
		},
		{
			name:         "unusable required chain up to the root drops the tool",
			in:           `{"type":"object","properties":{"o":{"type":"object","properties":{"x":{"type":"array"}},"required":["x"]}},"required":["o"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonArrayItems,
			wantDropPath: "/properties/o/properties/x",
		},
		{
			name: "object with anyOf and a usable member becomes free-form",
			in: `{"type":"object","properties":{"o":{"type":"object","description":"d","oneOf":[
				{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},
				{"type":"object","properties":{"b":{"type":"integer"}},"required":["b"]}]}}}`,
			want: `{"type":"object","properties":{"o":{"type":"object","description":"d"}}}`,
		},
		{
			name: "object with anyOf and no usable member is unusable",
			in: `{"type":"object","properties":{"o":{"type":"object","oneOf":[
				{"type":"object","properties":{"v":{}},"required":["v"]}]},"k":{"type":"string"}}}`,
			want:             `{"type":"object","properties":{"k":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/o"},
		},

		// R11 arrays.
		{
			name:             "optional array without items is removed",
			in:               `{"type":"object","properties":{"a":{"type":"array"},"b":{"type":"string"}}}`,
			want:             `{"type":"object","properties":{"b":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/a"},
		},
		{
			name:         "required array without items drops the tool",
			in:           `{"type":"object","properties":{"a":{"type":"array"}},"required":["a"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonArrayItems,
			wantDropPath: "/properties/a",
		},
		{
			name:             "array whose items are unusable is unusable",
			in:               `{"type":"object","properties":{"a":{"type":"array","items":{"description":"any"}}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/a"},
		},

		// R8 required pruning.
		{
			name:             "required pruned to emitted properties and deduplicated in order",
			in:               `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"array"}},"required":["b","missing","a","b"]}`,
			want:             `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["b","a"]}`,
			wantDroppedProps: []string{"/properties/c"},
		},
		{
			name: "object with usable properties drops a usable anyOf and keeps its properties (R10a)",
			in: `{"type":"object","properties":{"o":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}},
				"oneOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},{"type":"object","properties":{"b":{"type":"integer"}},"required":["b"]}]}}}`,
			want: `{"type":"object","properties":{"o":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}}}}}`,
		},
		{
			name: "root single-element list type object is a plain object",
			in:   `{"type":["object"],"properties":{"a":{"type":"string"}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}}}`,
		},
		{
			name: "string const inside a list type becomes the string member's enum",
			in:   `{"type":"object","properties":{"v":{"type":["string","null"],"const":"x"}}}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"string","enum":["x"]},{"type":"null"}]}}}`,
		},
		{
			name: "a non-object type next to a usable anyOf keeps only the type (R10a)",
			in:   `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"type":"string"},{"type":"null"}]}}}`,
			want: `{"type":"object","properties":{"v":{"type":"string"}}}`,
		},
		{
			name:         "a type next to an anyOf with no usable member is still unusable (R10a gate)",
			in:           `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"type":"array"}]}},"required":["v"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/v",
		},

		// R7a constraint-only anyOf/oneOf members next to a type.
		{
			name: "required at-least-one-of anyOf on a required object is removed, the tool is kept",
			in: `{"type":"object","properties":{"q":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},
				"anyOf":[{"required":["a"]},{"required":["b"]}]}},"required":["q"]}`,
			want: `{"type":"object","properties":{"q":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}}}},"required":["q"]}`,
			check: func(t *testing.T, rep Report) {
				if rep.DroppedKeys != 1 {
					t.Errorf("Wrong match. DroppedKeys expect: 1, got: %d", rep.DroppedKeys)
				}
			},
		},
		{
			name: "format-only oneOf on a required string is removed, the tool is kept",
			in:   `{"type":"object","properties":{"d":{"type":"string","oneOf":[{"format":"date"},{"format":"date-time"}]}},"required":["d"]}`,
			want: `{"type":"object","properties":{"d":{"type":"string"}},"required":["d"]}`,
		},
		{
			name: "bound-only anyOf on an integer is removed, the property is kept",
			in:   `{"type":"object","properties":{"n":{"type":"integer","anyOf":[{"minimum":1},{"maximum":-1}]}}}`,
			want: `{"type":"object","properties":{"n":{"type":"integer"}}}`,
		},
		{
			name: "constraint-only anyOf next to a list type is removed, the list type is kept",
			in:   `{"type":"object","properties":{"x":{"type":["string","null"],"anyOf":[{"minLength":1}]}}}`,
			want: `{"type":"object","properties":{"x":{"anyOf":[{"type":"string"},{"type":"null"}]}}}`,
		},
		{
			name: "one constraint-only member is enough to treat the anyOf as a refinement",
			in:   `{"type":"object","properties":{"s":{"type":"string","anyOf":[{"type":"string","enum":["a"]},{"pattern":"^b"}]}},"required":["s"]}`,
			want: `{"type":"object","properties":{"s":{"type":"string"}},"required":["s"]}`,
		},
		{
			name:             "a typeless subschema's constraint-only anyOf is still evaluated (not a refinement of a type)",
			in:               `{"type":"object","properties":{"v":{"anyOf":[{"minLength":1}]}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/v"},
		},
		{
			name: "a constraint-only member behind a $ref is a refinement",
			in: `{"type":"object","$defs":{"A":{"required":["a"]},"B":{"required":["b"]}},
				"properties":{"q":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},
				"anyOf":[{"$ref":"#/$defs/A"},{"$ref":"#/$defs/B"}]}},"required":["q"]}`,
			want: `{"type":"object","properties":{"q":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}}}},"required":["q"]}`,
			check: func(t *testing.T, rep Report) {
				if rep.Rewrites != 0 {
					t.Errorf("Wrong match. the look-ahead must not count rewrites, got: %d", rep.Rewrites)
				}
			},
		},
		{
			name: "a constraint-only member behind an allOf wrapper is a refinement",
			in:   `{"type":"object","properties":{"q":{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"allOf":[{"required":["a"]}]}]}},"required":["q"]}`,
			want: `{"type":"object","properties":{"q":{"type":"object","properties":{"a":{"type":"string"}}}},"required":["q"]}`,
		},
		{
			name: "a nested constraint-only anyOf is a refinement",
			in:   `{"type":"object","properties":{"s":{"type":"string","anyOf":[{"anyOf":[{"minLength":1}]}]}},"required":["s"]}`,
			want: `{"type":"object","properties":{"s":{"type":"string"}},"required":["s"]}`,
		},
		{
			name: "the look-ahead into nested combinators leaves the report unchanged",
			in:   `{"type":"object","properties":{"s":{"type":"string","anyOf":[{"oneOf":{"x":1},"anyOf":[{"oneOf":[{"minLength":1}]}]}]}},"required":["s"]}`,
			want: `{"type":"object","properties":{"s":{"type":"string"}},"required":["s"]}`,
			check: func(t *testing.T, rep Report) {
				if rep.Rewrites != 0 || rep.DroppedKeys != 1 {
					t.Errorf("Wrong match. expect: 0 rewrites, 1 dropped key, got: %d, %d", rep.Rewrites, rep.DroppedKeys)
				}
			},
		},
		{
			name: "the look-ahead over a $ref DAG has its own expansion budget and the tool is kept",
			in:   refinementDAG(t, 7, 6),
			want: `{"type":"object","properties":{"p":{"type":"string"},"q":{"type":"string"}},"required":["q"]}`,
		},
		{
			name: "the look-ahead budget is per subschema: an optional DAG property does not turn a later required refinement into a drop",
			in: `{"type":"object","$defs":{"L0":{"anyOf":[{"$ref":"#/$defs/L1"},{"$ref":"#/$defs/L1"},{"$ref":"#/$defs/L1"},{"$ref":"#/$defs/L1"},{"$ref":"#/$defs/L1"},{"$ref":"#/$defs/L1"}]},
				"L1":{"anyOf":[{"$ref":"#/$defs/L2"},{"$ref":"#/$defs/L2"},{"$ref":"#/$defs/L2"},{"$ref":"#/$defs/L2"},{"$ref":"#/$defs/L2"},{"$ref":"#/$defs/L2"}]},
				"L2":{"anyOf":[{"$ref":"#/$defs/L3"},{"$ref":"#/$defs/L3"},{"$ref":"#/$defs/L3"},{"$ref":"#/$defs/L3"},{"$ref":"#/$defs/L3"},{"$ref":"#/$defs/L3"}]},
				"L3":{"anyOf":[{"$ref":"#/$defs/L4"},{"$ref":"#/$defs/L4"},{"$ref":"#/$defs/L4"},{"$ref":"#/$defs/L4"},{"$ref":"#/$defs/L4"},{"$ref":"#/$defs/L4"}]},
				"L4":{"type":"string"},"F":{"required":["k"]}},
				"properties":{"a":{"type":"string","anyOf":[{"$ref":"#/$defs/L0"},{"minLength":1}]},"b":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/F"}]}},"required":["b"]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["b"]}`,
		},
		{
			name: "a member behind an unresolvable $ref is not a refinement and is removed by the evaluation",
			in:   `{"type":"object","properties":{"s":{"type":"string","anyOf":[{"$ref":"#/$defs/missing"},{"type":"string","enum":["x"]}]}}}`,
			want: `{"type":"object","properties":{"s":{"anyOf":[{"type":"string","enum":["x"]}]}}}`,
		},

		// R10a documented-enum shape.
		{
			name: "a string with a oneOf of documented consts keeps the members and drops the parent type",
			in: `{"type":"object","properties":{"m":{"type":"string","description":"mode","oneOf":[
				{"const":"a","description":"first"},{"const":"b","description":"second"}]}}}`,
			want: `{"type":"object","properties":{"m":{"description":"mode","anyOf":[
				{"type":"string","enum":["a"],"description":"first"},{"type":"string","enum":["b"],"description":"second"}]}}}`,
		},
		{
			name: "same-typed members next to a parent enum keep only the parent",
			in:   `{"type":"object","properties":{"m":{"type":"string","enum":["a","b"],"oneOf":[{"const":"a"},{"const":"b"}]}}}`,
			want: `{"type":"object","properties":{"m":{"type":"string","enum":["a","b"]}}}`,
		},
		{
			name: "an integer with same-typed bounded members keeps the members",
			in:   `{"type":"object","properties":{"n":{"type":"integer","anyOf":[{"type":"integer","minimum":1},{"type":"integer","maximum":-1}]}}}`,
			want: `{"type":"object","properties":{"n":{"anyOf":[{"type":"integer","minimum":1},{"type":"integer","maximum":-1}]}}}`,
		},
		{
			name: "an integer with a oneOf of documented consts keeps the bare type and the tool",
			in: `{"type":"object","properties":{"n":{"type":"integer","description":"level","oneOf":[
				{"const":1,"description":"low"},{"const":2,"description":"high"}]}},"required":["n"]}`,
			want: `{"type":"object","properties":{"n":{"type":"integer","description":"level"}},"required":["n"]}`,
		},
		{
			name: "a boolean with a oneOf of documented consts keeps the bare type and the tool",
			in:   `{"type":"object","properties":{"b":{"type":"boolean","oneOf":[{"const":true,"description":"on"},{"const":false,"description":"off"}]}},"required":["b"]}`,
			want: `{"type":"object","properties":{"b":{"type":"boolean"}},"required":["b"]}`,
		},
		{
			name: "typed members that add no value constraint keep the bare type",
			in:   `{"type":"object","properties":{"n":{"type":"integer","oneOf":[{"type":"integer","const":1},{"type":"integer","const":2}]}}}`,
			want: `{"type":"object","properties":{"n":{"type":"integer"}}}`,
		},
		{
			name: "a string enum member without a type takes the parent type",
			in:   `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"const":"a"},{"enum":["b","c"]}]}}}`,
			want: `{"type":"object","properties":{"s":{"anyOf":[{"type":"string","enum":["a"]},{"type":"string","enum":["b","c"]}]}}}`,
		},
		{
			name: "integer documented consts behind $refs take the parent type",
			in:   `{"type":"object","$defs":{"One":{"const":1,"description":"low"},"Two":{"const":2,"description":"high"}},"properties":{"n":{"type":"integer","oneOf":[{"$ref":"#/$defs/One"},{"$ref":"#/$defs/Two"}]}},"required":["n"]}`,
			want: `{"type":"object","properties":{"n":{"type":"integer"}},"required":["n"]}`,
			check: func(t *testing.T, rep Report) {
				// oneOf, two $refs, two inherited types.
				if rep.Rewrites != 5 {
					t.Errorf("Wrong match. expect: 5 rewrites, got: %d", rep.Rewrites)
				}
			},
		},
		{
			name: "nullable integer documented consts take the non-null list type",
			in:   `{"type":"object","properties":{"n":{"type":["integer","null"],"oneOf":[{"const":1,"description":"low"},{"const":2,"description":"high"}]}},"required":["n"]}`,
			want: `{"type":"object","properties":{"n":{"anyOf":[{"type":"integer"},{"type":"null"}]}},"required":["n"]}`,
		},
		{
			name: "a number documented const keeps the bare type and the tool",
			in:   `{"type":"object","properties":{"x":{"type":"number","oneOf":[{"const":1.5},{"const":2}]}},"required":["x"]}`,
			want: `{"type":"object","properties":{"x":{"type":"number"}},"required":["x"]}`,
		},
		{
			name:         "a member whose const is not of the parent type does not take it and stays unusable",
			in:           `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"const":true}]}},"required":["s"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/s",
		},
		{
			name: "a member whose enum has no value of the parent type does not take it",
			in:   `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"enum":[1,2]},{"const":"a"}]}}}`,
			want: `{"type":"object","properties":{"s":{"anyOf":[{"type":"string","enum":["a"]}]}}}`,
		},
		{
			name: "a typed member of another scalar type keeps its own type",
			in:   `{"type":"object","properties":{"x":{"type":"number","oneOf":[{"type":"integer","minimum":1},{"type":"integer","maximum":-1}]}}}`,
			want: `{"type":"object","properties":{"x":{"type":"number"}}}`,
		},
		{
			name: "a member that is only a combinator passes the parent type to its own members",
			in:   `{"type":"object","properties":{"n":{"type":"integer","oneOf":[{"anyOf":[{"const":1},{"const":2}]}]}},"required":["n"]}`,
			want: `{"type":"object","properties":{"n":{"type":"integer"}},"required":["n"]}`,
			check: func(t *testing.T, rep Report) {
				// oneOf, and one inherited type per const member.
				if rep.Rewrites != 3 {
					t.Errorf("Wrong match. expect: 3 rewrites, got: %d", rep.Rewrites)
				}
			},
		},
		{
			name:         "a member with properties has a type of its own and does not take the parent type",
			in:           `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"properties":{"x":{"$ref":"#/$defs/missing"}},"required":["x"]}]}},"required":["s"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/s",
		},
		{
			name:         "a member with items has a type of its own and does not take the parent type",
			in:           `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"items":true}]}},"required":["s"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/s",
		},
		{
			name: "an integer enum member takes the integer parent type",
			in:   `{"type":"object","properties":{"n":{"type":"integer","oneOf":[{"enum":[1,2]}]}},"required":["n"]}`,
			want: `{"type":"object","properties":{"n":{"type":"integer"}},"required":["n"]}`,
		},
		{
			name: "a member takes a later scalar of a list type when the first does not fit",
			in:   `{"type":"object","properties":{"v":{"type":["string","integer"],"oneOf":[{"const":1}]}},"required":["v"]}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"string"},{"type":"integer"}]}},"required":["v"]}`,
		},
		{
			name: "an enum that is not a list is ignored when inheriting the parent type",
			in:   `{"type":"object","properties":{"p":{"type":"string","oneOf":[{"enum":"a"},{"const":"b"}]}},"required":["p"]}`,
			want: `{"type":"object","properties":{"p":{"type":"string"}},"required":["p"]}`,
		},
		{
			name: "a memoized refinement verdict is reused for a second property",
			in:   `{"type":"object","$defs":{"F":{"required":["k"]}},"properties":{"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/F"}]},"b":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/F"}]}},"required":["a","b"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"b":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["a","b"]}`,
		},
		{
			name:         "a self-referencing $ref member is not a refinement",
			in:           `{"type":"object","$defs":{"A":{"anyOf":[{"$ref":"#/$defs/A"}]}},"properties":{"s":{"type":"string","anyOf":[{"$ref":"#/$defs/A"}]}},"required":["s"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/s",
		},
		{
			name: "a $ref member met again inside an outer judgement is not memoized as not a refinement",
			in: `{"type":"object","$defs":{"X":{"anyOf":[{"$ref":"#/$defs/B"},{"$ref":"#/$defs/L"}]},"B":{"anyOf":[{"$ref":"#/$defs/X"}]},"L":{"required":["k"]}},` +
				`"properties":{"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/X"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/B"}]}},"required":["z"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
		},
		{
			name: "a $ref member two hops inside an outer judgement is not memoized as not a refinement",
			in: `{"type":"object","$defs":{"O":{"anyOf":[{"$ref":"#/$defs/P"},{"$ref":"#/$defs/L"}]},"P":{"anyOf":[{"$ref":"#/$defs/Y"}]},"Y":{"anyOf":[{"$ref":"#/$defs/O"}]},"L":{"required":["k"]}},` +
				`"properties":{"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/O"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/P"}]}},"required":["z"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
		},
		{
			name: "a ref judged not a refinement earlier is not made one by a later refinement",
			in: `{"type":"object","$defs":{"F":{"anyOf":[{"$ref":"#/$defs/F"},{"type":"array"}]},"L":{"required":["k"]}},"properties":{` +
				`"a":{"type":"string","anyOf":[{"$ref":"#/$defs/F"},{"type":"string"}]},` +
				`"b":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/L"}]},` +
				`"c":{"type":"string","anyOf":[{"$ref":"#/$defs/F"}]}},"required":["c"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/c",
		},
		{
			name: "a ref met again inside its own judgement is not taken as decided before it is",
			in: `{"type":"object","$defs":{"A":{"anyOf":[{"$ref":"#/$defs/C"},{"$ref":"#/$defs/D"},{"$ref":"#/$defs/L"}]},"C":{"anyOf":[{"$ref":"#/$defs/S"}]},"S":{"type":"string"},` +
				`"D":{"anyOf":[{"$ref":"#/$defs/A"}]},"L":{"required":["k"]}},"properties":{` +
				`"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/A"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/D"}]}},"required":["z"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
		},
		{
			name: "a ref still being judged is not settled by a ref judged inside it",
			in: `{"type":"object","$defs":{"A":{"anyOf":[{"$ref":"#/$defs/B"},{"$ref":"#/$defs/L"}]},"B":{"type":"string"},"L":{"required":["k"]}},"properties":{` +
				`"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/A"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/A"}]}},"required":["z"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
		},
		{
			name: "a ref that meets no ref on the stack is settled even after a sibling met one",
			in: `{"type":"object","$defs":{"P":{"anyOf":[{"$ref":"#/$defs/W"},{"$ref":"#/$defs/Q"},{"$ref":"#/$defs/L"}]},"W":{"anyOf":[{"$ref":"#/$defs/P"}]},"Q":{"type":"array"},"L":{"required":["k"]}},"properties":{` +
				`"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/P"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/Q"}]}},"required":["z"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/z",
		},
		{
			name: "settling a ref leaves the refs judged before it on the stack",
			in: `{"type":"object","$defs":{"P":{"anyOf":[{"$ref":"#/$defs/W"},{"$ref":"#/$defs/Q"},{"$ref":"#/$defs/L"}]},"W":{"anyOf":[{"$ref":"#/$defs/P"}]},"Q":{"type":"array"},"L":{"required":["k"]}},"properties":{` +
				`"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/P"}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/W"}]}},"required":["z"]}`,
			want: `{"type":"object","properties":{"a":{"type":"object","properties":{"k":{"type":"string"}}},"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
		},
		{
			name: "a $defs and a definitions target of the same name have their own verdicts",
			in: `{"type":"object","$defs":{"A":{"required":["k"]}},"definitions":{"A":{"type":"array"}},` +
				`"properties":{"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/A"}]},` +
				`"b":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/definitions/A"}]}},"required":["b"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/b",
		},
		{
			name: "a memoized $ref member is judged without the expansion stack that reached it",
			in: `{"type":"object","$defs":{"X":{"required":["k"]},"R":{"$ref":"#/$defs/X","type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/X"}]}},` +
				`"properties":{"z":{"$ref":"#/$defs/R"}},"required":["z"]}`,
			want: `{"type":"object","properties":{"z":{"type":"object","properties":{"k":{"type":"string"}},"required":["k"]}},"required":["z"]}`,
		},
		{
			name:         "an empty enum member does not take the parent type",
			in:           `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"enum":[]}]}},"required":["s"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonAnyOf,
			wantDropPath: "/properties/s",
		},
		{
			name: "a $ref member with a shape of its own is not judged by its target's verdict",
			in:   `{"type":"object","$defs":{"F":{"format":"date"}},"properties":{"s":{"type":"string","anyOf":[{"$ref":"#/$defs/F"}]},"t":{"type":"string","anyOf":[{"$ref":"#/$defs/F","type":"string","enum":["a"]}]}}}`,
			want: `{"type":"object","properties":{"s":{"type":"string"},"t":{"anyOf":[{"type":"string","enum":["a"]}]}}}`,
		},
		{
			name: "a format member counts as a constrained member",
			in:   `{"type":"object","properties":{"s":{"type":"string","oneOf":[{"const":"a"},{"type":"string","format":"date-time"}]}}}`,
			want: `{"type":"object","properties":{"s":{"anyOf":[{"type":"string","enum":["a"]},{"type":"string","format":"date-time"}]}}}`,
		},
		{
			name: "a shaped anyOf discarded next to a list type is counted as a dropped key",
			in:   `{"type":"object","properties":{"x":{"type":["string","null"],"anyOf":[{"type":"string"}]}}}`,
			want: `{"type":"object","properties":{"x":{"anyOf":[{"type":"string"},{"type":"null"}]}}}`,
			check: func(t *testing.T, rep Report) {
				if rep.DroppedKeys != 1 {
					t.Errorf("Wrong match. DroppedKeys expect: 1, got: %d", rep.DroppedKeys)
				}
			},
		},
		{
			name: "duplicate list type entries give one member per distinct type",
			in:   `{"type":"object","properties":{"v":{"type":["string","string","null"],"enum":["x"]}}}`,
			want: `{"type":"object","properties":{"v":{"anyOf":[{"type":"string","enum":["x"]},{"type":"null"}]}}}`,
		},
		{
			name: "20 dropped optional properties report 8 paths and the full count",
			in: `{"type":"object","properties":{
				"p00":{},"p01":{},"p02":{},"p03":{},"p04":{},"p05":{},"p06":{},"p07":{},"p08":{},"p09":{},
				"p10":{},"p11":{},"p12":{},"p13":{},"p14":{},"p15":{},"p16":{},"p17":{},"p18":{},"p19":{},
				"ok":{"type":"string"}}}`,
			want: `{"type":"object","properties":{"ok":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/p00", "/properties/p01", "/properties/p02", "/properties/p03",
				"/properties/p04", "/properties/p05", "/properties/p06", "/properties/p07"},
			check: func(t *testing.T, rep Report) {
				if rep.DroppedPropsN != 20 || len(rep.DroppedProps) != 8 {
					t.Errorf("Wrong match. expect: 20 dropped, 8 paths, got: %d dropped, %d paths", rep.DroppedPropsN, len(rep.DroppedProps))
				}
			},
		},
		{
			name:             "property names with slash and tilde are pointer-escaped in paths",
			in:               `{"type":"object","properties":{"a/b~c":{}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/a~1b~0c"},
		},

		// R5 $ref.
		{
			name: "$ref to $defs is inlined with referencing siblings winning",
			in: `{"type":"object","$defs":{"Color":{"type":"string","description":"inner","enum":["red","blue"],"title":"Color"}},
				"properties":{"c":{"$ref":"#/$defs/Color","description":"outer"}}}`,
			want: `{"type":"object","properties":{"c":{"type":"string","description":"outer","enum":["red","blue"]}}}`,
		},
		{
			name: "$ref to definitions with pointer escapes",
			in:   `{"type":"object","definitions":{"a/b~c":{"type":"integer"}},"properties":{"x":{"$ref":"#/definitions/a~1b~0c"}}}`,
			want: `{"type":"object","properties":{"x":{"type":"integer"}}}`,
		},
		{
			name: "single-member allOf wrapping a $ref (Pydantic)",
			in: `{"type":"object","$defs":{"Mode":{"enum":["a","b"],"type":"string"}},
				"properties":{"m":{"allOf":[{"$ref":"#/$defs/Mode"}],"description":"mode","default":"a"}},"required":["m"]}`,
			want: `{"type":"object","properties":{"m":{"type":"string","description":"mode","enum":["a","b"]}},"required":["m"]}`,
		},
		{
			name: "root $ref is resolved before the absent-type default",
			in:   `{"$ref":"#/$defs/Args","$defs":{"Args":{"properties":{"q":{"type":"string"}},"required":["q"]}}}`,
			want: `{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}`,
		},
		{
			name: "nested $ref inside a $defs target",
			in: `{"type":"object","$defs":{"Outer":{"type":"object","properties":{"in":{"$ref":"#/$defs/Inner"}}},"Inner":{"type":"boolean"}},
				"properties":{"o":{"$ref":"#/$defs/Outer"}}}`,
			want: `{"type":"object","properties":{"o":{"type":"object","properties":{"in":{"type":"boolean"}}}}}`,
		},
		{
			name:             "missing $ref target is unusable",
			in:               `{"type":"object","properties":{"x":{"$ref":"#/$defs/Nope"},"y":{"type":"string"}}}`,
			want:             `{"type":"object","properties":{"y":{"type":"string"}}}`,
			wantDroppedProps: []string{"/properties/x"},
		},
		{
			name:         "remote $ref on a required property drops the tool",
			in:           `{"type":"object","properties":{"x":{"$ref":"https://example.com/s.json"}},"required":["x"]}`,
			want:         "null",
			wantDropped:  true,
			wantReason:   ReasonRef,
			wantDropPath: "/properties/x",
		},
		{
			name: "whole-document and nested-pointer refs are unusable",
			in: `{"type":"object","$defs":{"A":{"type":"object","properties":{"b":{"type":"string"}}}},
				"properties":{"w":{"$ref":"#"},"n":{"$ref":"#/$defs/A/properties/b"},"s":{"$ref":5}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/n", "/properties/s", "/properties/w"},
		},
		{
			name:             "direct $ref cycle is unusable",
			in:               `{"type":"object","$defs":{"A":{"$ref":"#/$defs/A"}},"properties":{"x":{"$ref":"#/$defs/A"}}}`,
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/x"},
		},
		{
			name: "indirect $ref cycle through properties is cut at the re-entry",
			in: `{"type":"object","$defs":{"Node":{"type":"object","properties":{"name":{"type":"string"},"child":{"$ref":"#/$defs/Node"}}}},
				"properties":{"tree":{"$ref":"#/$defs/Node"}}}`,
			want:             `{"type":"object","properties":{"tree":{"type":"object","properties":{"name":{"type":"string"}}}}}`,
			wantDroppedProps: []string{"/properties/tree/properties/child"},
		},
		{
			name: "8 nested $ref expansions are allowed",
			in:   refChain(8),
			want: `{"type":"object","properties":{"x":{"type":"string"}}}`,
		},
		{
			name:             "9 nested $ref expansions are unusable",
			in:               refChain(9),
			want:             `{"type":"object","properties":{}}`,
			wantDroppedProps: []string{"/properties/x"},
		},
		{
			name:             "more than 256 $ref expansions per tool are unusable past the limit",
			in:               refFanOut(257),
			want:             refFanOutWant(256),
			wantDroppedProps: []string{"/properties/p256"},
		},

		// R12 depth.
		{
			name: "depth 32 is kept",
			in:   nestedObjects(32, ""),
			want: nestedObjectsWant(32),
		},
		{
			name:             "depth 33 is unusable",
			in:               nestedObjects(33, ""),
			want:             nestedObjectsWant(32),
			wantDroppedProps: []string{nestedPath(33)},
		},
		{
			name: "a $ref chain adds no depth",
			in:   nestedObjects(32, `"$defs":{"A":{"$ref":"#/$defs/B"},"B":{"$ref":"#/$defs/C"},"C":{"type":"object"}},`),
			want: nestedObjectsWant(32),
		},

		// R12 output charge and resource limits.
		{
			// Visits: root, a, n, e = 4 x 32 = 128. Strings: root type
			// "object" 9; a: "string" 9, "hi" 5, name 4; n: "integer" 10,
			// "int32" 8, name 4; e: "string" 9, "x" 4, "yy" 5, name 4;
			// required "a" 4. Numbers: minimum 24. Total 227.
			name: "small plain schema charge matches the formula",
			in: `{"type":"object","properties":{"a":{"type":"string","description":"hi"},
				"n":{"type":"integer","minimum":1,"format":"int32"},"e":{"type":"string","enum":["x","yy"]}},"required":["a"]}`,
			want: `{"type":"object","properties":{"a":{"type":"string","description":"hi"},
				"n":{"type":"integer","minimum":1,"format":"int32"},"e":{"type":"string","enum":["x","yy"]}},"required":["a"]}`,
			check: func(t *testing.T, rep Report) {
				if rep.OutBytes != 227 {
					t.Errorf("Wrong match. OutBytes expect: 227, got: %d", rep.OutBytes)
				}
			},
		},
		{
			// 2 visits (64) + "object" 9 + "string" 9 + name "d" 4 + the
			// description (len + 3) = 89 + len, exactly the cap.
			name: "schema charged exactly at the cap is kept",
			in:   longDescription(defaultMaxOut - 89),
			want: longDescription(defaultMaxOut - 89),
			check: func(t *testing.T, rep Report) {
				if rep.OutBytes != defaultMaxOut {
					t.Errorf("Wrong match. OutBytes expect: %d, got: %d", defaultMaxOut, rep.OutBytes)
				}
			},
		},
		{
			name:        "schema charged one byte over the cap drops the tool",
			in:          longDescription(defaultMaxOut - 88),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name:        "$ref amplification is stopped by the output cap",
			in:          refAmplification(t),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check: func(t *testing.T, rep Report) {
				if rep.OutBytes > defaultMaxOut+56003+64 {
					t.Errorf("Wrong match. build must stop soon after the cap, got OutBytes %d", rep.OutBytes)
				}
			},
		},
		{
			name:        "dropped-subtree amplification is stopped by the visit charge",
			in:          droppedPropsAmplification(t),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check: func(t *testing.T, rep Report) {
				if len(rep.DroppedProps) > 8 {
					t.Errorf("Wrong match. DroppedProps must stay capped at 8, got %d", len(rep.DroppedProps))
				}
				if rep.DroppedPropsN > defaultMaxOut/32 {
					t.Errorf("Wrong match. build must stop after about %d visits, got %d dropped", defaultMaxOut/32, rep.DroppedPropsN)
				}
			},
		},
		{
			name:        "allOf chain fan-out is stopped by the work cap",
			in:          allOfChainFanOut(t, 2000, 256),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name:        "a long required list fanned out by $ref is stopped by the work cap",
			in:          listFanOut(t, "required", 7500, 256),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name:        "a long rejected enum fanned out by $ref is stopped by the work cap",
			in:          listFanOut(t, "enum", 12000, 256),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name:        "a long anyOf fanned out by $ref is stopped by the work cap",
			in:          listFanOut(t, "anyOf", 2000, 256),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name:        "a long list type fanned out by $ref is stopped by the work cap",
			in:          listFanOut(t, "type", 7000, 256),
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
		{
			name: "a short allOf chain is resolved and its sibling keys dropped",
			in:   `{"type":"object","properties":{"a":{"k0":1,"allOf":[{"k1":1,"allOf":[{"type":"string"}]}]}}}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}}}`,
			check: func(t *testing.T, rep Report) {
				if rep.DroppedKeys != 2 || rep.Rewrites != 2 {
					t.Errorf("Wrong match. expect: 2 dropped keys, 2 rewrites, got: %d, %d", rep.DroppedKeys, rep.Rewrites)
				}
			},
		},
		{
			name:   "4096 emitted subschemas are within the node cap",
			in:     manyProperties(4095),
			maxOut: 1 << 30,
			want:   manyProperties(4095),
		},
		{
			name:        "more than 4096 emitted subschemas drop the tool",
			in:          manyProperties(4096),
			maxOut:      1 << 30,
			want:        "null",
			wantDropped: true,
			wantReason:  ReasonTooLarge,
			check:       func(t *testing.T, rep Report) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runNormalizeRow(t, tt)
		})
	}
}

func Test_Normalize_LookAheadBudgetIndependentOfBuild(t *testing.T) {
	raw, want := refinementAfterBuildRefs(t, 252, 6)
	runNormalizeRow(t, normalizeRow{
		name: "the look-ahead budget does not start from the build's expansion count",
		in:   raw,
		want: want,
		check: func(t *testing.T, rep Report) {
			// The build's 252 expansions stay counted; the look-ahead's do not.
			if rep.Rewrites != 252 {
				t.Errorf("Wrong match. expect: 252 rewrites, got: %d", rep.Rewrites)
			}
		},
	})
}

func Test_Normalize_InheritedEnumScanIsCharged(t *testing.T) {
	// Each member resolves to an enum whose only integer value is last, so
	// inheriting the parent type scans the whole list; nothing else charges
	// it. 30 such scans exceed maxWork.
	vals := strings.TrimSuffix(strings.Repeat(`"a",`, 9000), ",")
	refs := strings.TrimSuffix(strings.Repeat(`{"$ref":"#/$defs/E"},`, 30), ",")
	raw := `{"type":"object","$defs":{"E":{"enum":[` + vals + `,1]}},"properties":{"n":{"type":"integer","oneOf":[` + refs + `]}}}`
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	runNormalizeRow(t, normalizeRow{
		name:        "the enum scan of an inherited type is charged as work",
		in:          raw,
		want:        "null",
		wantDropped: true,
		wantReason:  ReasonTooLarge,
	})
}

func Test_Normalize_LookAheadMemoizesRefVerdicts(t *testing.T) {
	// A long property name and 250 $ref members to one 1000-member anyOf,
	// then a refinement. Judged once per $ref, the look-ahead stays cheap
	// and the refinement removes the combinator. Judged per reference it
	// reaches the same verdict, but builds about 16 GB of paths, so the
	// allocation bound is what this test pins.
	name := strings.Repeat("x", 58000)
	ones := strings.TrimSuffix(strings.Repeat("1,", 1000), ",")
	refs := strings.TrimSuffix(strings.Repeat(`{"$ref":"#/$defs/A"},`, 250), ",")
	raw := `{"type":"object","$defs":{"A":{"anyOf":[` + ones + `]}},"properties":{"` + name +
		`":{"type":"string","anyOf":[` + refs + `,{"minLength":1}]}}}`
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	row := normalizeRow{
		name: "a $ref member's refinement verdict is judged once per tool",
		in:   raw,
		want: `{"type":"object","properties":{"` + name + `":{"type":"string"}}}`,
	}
	runNormalizeRow(t, row)
	checkAllocBound(t, raw, 16<<20)
}

func Test_Normalize_MemoizedRefVerdictHasItsOwnBudget(t *testing.T) {
	// 255 allOf-wrapped $ref members (not memoized) spend the look-ahead's
	// budget before a plain $ref member whose refinement is 6 hops away. The
	// memoized judgement starts from a fresh budget, so it still finds the
	// refinement and the required property is kept.
	members := make([]string, 0, 256)
	for i := 0; i < 255; i++ {
		members = append(members, `{"allOf":[{"$ref":"#/$defs/S"}]}`)
	}
	members = append(members, `{"$ref":"#/$defs/C0"}`)
	// S is unusable (an array without items), so without the refinement
	// the anyOf has no usable member and the required z drops the tool.
	defs := []string{`"S":{"type":"array"}`}
	for k := 0; k < 5; k++ {
		defs = append(defs, fmt.Sprintf(`"C%d":{"$ref":"#/$defs/C%d"}`, k, k+1))
	}
	defs = append(defs, `"C5":{"required":["k"]}`)
	raw := `{"type":"object","$defs":{` + strings.Join(defs, ",") + `},"properties":{"z":{"type":"object",` +
		`"properties":{"k":{"type":"string"}},"anyOf":[` + strings.Join(members, ",") + `]}},"required":["z"]}`
	runNormalizeRow(t, normalizeRow{
		name: "a memoized $ref member is judged with a fresh budget",
		in:   raw,
		want: `{"type":"object","properties":{"z":{"type":"object","properties":{"k":{"type":"string"}}}},"required":["z"]}`,
	})
}

func Test_Normalize_LookAheadChargesEachVisitedMember(t *testing.T) {
	// 240 allOf-wrapped (not memoized) $refs to a 1000-member anyOf of
	// non-objects, then a refinement. Listing A's members is charged 1000
	// per visit; visiting each of them is charged 1000 more, which takes
	// the look-ahead past maxWork.
	ones := strings.TrimSuffix(strings.Repeat("1,", 1000), ",")
	refs := strings.TrimSuffix(strings.Repeat(`{"allOf":[{"$ref":"#/$defs/A"}]},`, 240), ",")
	raw := `{"type":"object","$defs":{"A":{"anyOf":[` + ones + `]}},"properties":{"s":{"type":"string","anyOf":[` +
		refs + `,{"minLength":1}]}}}`
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	runNormalizeRow(t, normalizeRow{
		name:        "each member the look-ahead visits is charged as work",
		in:          raw,
		want:        "null",
		wantDropped: true,
		wantReason:  ReasonTooLarge,
	})
}

// checkAllocBound fails when normalizing raw allocates more than limit
// bytes. Normalize is single-goroutine and deterministic, so its
// allocation is stable across runs; the bounds are several times the
// measured cost and far below the regressions they guard.
func checkAllocBound(t *testing.T, raw string, limit uint64) {
	t.Helper()
	in := decodeJSON(t, raw)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	Normalize(in, defaultMaxOut)
	runtime.ReadMemStats(&after)
	if got := after.TotalAlloc - before.TotalAlloc; got > limit {
		t.Errorf("Wrong match. expect: at most %d bytes allocated, got: %d", limit, got)
	}
}

func Test_Normalize_LookAheadBuildsNoPaths(t *testing.T) {
	// A long property name and 100 allOf-wrapped (not memoized) $refs to a
	// 1000-member anyOf, then a refinement. The look-ahead visits every
	// member; building a path per visit would copy the name each time.
	name := strings.Repeat("x", 58000)
	ones := strings.TrimSuffix(strings.Repeat("1,", 1000), ",")
	refs := strings.TrimSuffix(strings.Repeat(`{"allOf":[{"$ref":"#/$defs/A"}]},`, 100), ",")
	raw := `{"type":"object","$defs":{"A":{"anyOf":[` + ones + `]}},"properties":{"` + name +
		`":{"type":"string","anyOf":[` + refs + `,{"minLength":1}]}}}`
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	runNormalizeRow(t, normalizeRow{
		name: "the look-ahead builds no path strings",
		in:   raw,
		want: `{"type":"object","properties":{"` + name + `":{"type":"string"}}}`,
	})
	checkAllocBound(t, raw, 64<<20)
}

func Test_Normalize_RefLookupIsCachedPerRef(t *testing.T) {
	// A 20000-byte def name reached from 128 allOf-wrapped members of W,
	// which every property's look-ahead expands. Building the def's stack
	// key on every expansion copies the name each time.
	long := strings.Repeat("q", 20000)
	sm := strings.TrimSuffix(strings.Repeat(`{"allOf":[{"$ref":"#/$defs/S"}]},`, 128), ",")
	defs := `"` + long + `":{"type":"integer"},"S":{"$ref":"#/$defs/` + long + `"},"W":{"anyOf":[` + sm + `]}`
	props := []string{}
	for i := 0; ; i++ {
		p := fmt.Sprintf(`"%x":{"type":"string","anyOf":[{"allOf":[{"$ref":"#/$defs/W"}]}]}`, i)
		cand := `{"type":"object","$defs":{` + defs + `},"properties":{` + strings.Join(append(props, p), ",") + `}}`
		if len(cand) > 64<<10 {
			break
		}
		props = append(props, p)
	}
	raw := `{"type":"object","$defs":{` + defs + `},"properties":{` + strings.Join(props, ",") + `}}`
	if len(props) < 100 {
		t.Fatalf("shape has only %d properties", len(props))
	}
	// The first property's build spends the tool's $ref expansions, so
	// every later one is an optional property dropped by the cascade (R12);
	// each is still looked ahead first.
	runNormalizeRow(t, normalizeRow{
		name:             "a long def name is not copied per expansion",
		in:               raw,
		want:             `{"type":"object","properties":{"0":{"type":"string"}}}`,
		wantDroppedProps: []string{"/properties/1", "/properties/10", "/properties/100", "/properties/101", "/properties/102", "/properties/103", "/properties/104", "/properties/105"},
		check: func(t *testing.T, rep Report) {
			if rep.DroppedPropsN != len(props)-1 {
				t.Errorf("Wrong match. expect: %d dropped properties, got: %d", len(props)-1, rep.DroppedPropsN)
			}
		},
	})
	checkAllocBound(t, raw, 256<<20)
}

func Test_Normalize_VerdictKeysKeepAMemberOutOfTheMemo(t *testing.T) {
	// F is a refinement. The optional a is judged first with F next to a
	// key that gives the member a shape (or a verdict) of its own; were
	// that verdict memoized under F's ref, the required z, whose member is
	// plain F, would lose the refinement and drop the tool.
	siblings := map[string]string{
		"type":       `"type":"integer"`,
		"properties": `"properties":{"x":{"type":"string"}}`,
		"items":      `"items":{"type":"string"}`,
		"enum":       `"enum":[1]`,
		"const":      `"const":1`,
		"anyOf":      `"anyOf":[{"type":"integer"}]`,
		"oneOf":      `"oneOf":[{"type":"integer"}]`,
		"allOf":      `"allOf":[{"type":"integer"}]`,
	}
	for _, k := range verdictKeys {
		sib, ok := siblings[k]
		if !ok {
			t.Fatalf("no row for verdict key %q", k)
		}
		t.Run(k, func(t *testing.T) {
			in := `{"type":"object","$defs":{"F":{"required":["k"]}},"properties":{` +
				`"a":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/F",` + sib + `}]},` +
				`"z":{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/F"}]}},"required":["z"]}`
			_, rep := Normalize(decodeJSON(t, in), defaultMaxOut)
			if rep.ToolDropped {
				t.Errorf("Wrong match. expect: tool kept, got: dropped (%s at %s)", rep.DropReason, rep.DropPath)
			}
		})
	}
	if len(siblings) != len(verdictKeys) {
		t.Errorf("Wrong match. expect: %d verdict keys, got: %d", len(siblings), len(verdictKeys))
	}
}

func Test_Normalize_MemoizedRefVerdictIgnoresTheCallerDepth(t *testing.T) {
	// A required object nested 29 levels deep whose only anyOf member is
	// a $ref to a refinement behind four anyOf levels. Judged from the
	// member's own depth the refinement is past maxDepth; judged from its
	// target alone it is found and the tool is kept.
	inner := `{"type":"object","properties":{"k":{"type":"string"}},"anyOf":[{"$ref":"#/$defs/C"}]}`
	wrap := inner
	for i := 0; i < 29; i++ {
		wrap = `{"type":"object","properties":{"n":` + wrap + `},"required":["n"]}`
	}
	in := `{"type":"object","$defs":{"C":{"anyOf":[{"anyOf":[{"anyOf":[{"anyOf":[{"required":["k"]}]}]}]}]}},"properties":{"r":` + wrap + `},"required":["r"]}`
	_, rep := Normalize(decodeJSON(t, in), defaultMaxOut)
	if rep.ToolDropped {
		t.Errorf("Wrong match. expect: tool kept, got: dropped (%s at %s)", rep.DropReason, rep.DropPath)
	}
}

func Test_Normalize_CyclicRefVerdictsAreJudgedOncePerRef(t *testing.T) {
	// A 16-level $defs DAG, 2 $ref members per level, whose last level
	// refers back to the first. A ref met again while it is judged makes
	// every ref on the cycle wait for the first one; were the waiting
	// verdicts judged again on each path, the look-ahead would take 2^16
	// paths and drop the tool before it reached the refinement after them.
	var defs []string
	for i := 0; i < 16; i++ {
		defs = append(defs, fmt.Sprintf(`"D%d":{"anyOf":[{"$ref":"#/$defs/D%d"},{"$ref":"#/$defs/D%d"}]}`, i, i+1, i+1))
	}
	defs = append(defs, `"D16":{"anyOf":[{"$ref":"#/$defs/D0"},{"type":"string"}]}`)
	raw := `{"type":"object","$defs":{` + strings.Join(defs, ",") + `},"properties":{"p":{"type":"string","anyOf":[{"$ref":"#/$defs/D0"},{"minLength":1}]}},"required":["p"]}`
	runNormalizeRow(t, normalizeRow{
		name: "cyclic $ref verdicts are judged once per ref",
		in:   raw,
		want: `{"type":"object","properties":{"p":{"type":"string"}},"required":["p"]}`,
	})
}
