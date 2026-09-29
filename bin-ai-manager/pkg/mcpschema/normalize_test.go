package mcpschema

import (
	"encoding/json"
	"reflect"
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
