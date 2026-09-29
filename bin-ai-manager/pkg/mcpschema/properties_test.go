package mcpschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// mutationFixture exercises every rewrite path that builds new maps from
// the input: $ref, allOf, oneOf, const, list type, inference and cascade.
const mutationFixture = `{"type":"object","title":"T","$defs":{"M":{"type":"string","enum":["a","b"]},
	"O":{"type":"object","properties":{"x":{"type":"string"},"bad":{}},"required":["x"]}},
	"properties":{
		"m":{"allOf":[{"$ref":"#/$defs/M"}],"description":"mode"},
		"o":{"$ref":"#/$defs/O"},
		"u":{"oneOf":[{"type":"string"},{"type":"null"}]},
		"c":{"const":"k"},
		"l":{"type":["string","number","null"],"enum":["z"],"minimum":1},
		"e":{"enum":["p","q"]},
		"f":{"type":"object","properties":{},"x-extra":true},
		"arr":{"type":"array"}},
	"required":["m","o"]}`

func Test_Normalize_DoesNotMutateInput(t *testing.T) {
	in := decodeJSON(t, mutationFixture)
	orig := decodeJSON(t, mutationFixture)

	out, rep := Normalize(in, defaultMaxOut)
	if rep.ToolDropped || out == nil {
		t.Fatalf("fixture must be kept, got report %+v", rep)
	}
	if !reflect.DeepEqual(in, orig) {
		t.Errorf("Wrong match. input was mutated\nexpect: %v\ngot: %v", orig, in)
	}

	// Writing through the output must not reach the input either.
	out["properties"].(map[string]any)["m"].(map[string]any)["enum"].([]any)[0] = "changed"
	out["properties"].(map[string]any)["o"].(map[string]any)["required"].([]any)[0] = "changed"
	if !reflect.DeepEqual(in, orig) {
		t.Errorf("Wrong match. output shares a map with the input")
	}
}

func Test_Normalize_Deterministic(t *testing.T) {
	props := make([]string, 0, 40)
	for i := 0; i < 30; i++ {
		props = append(props, fmt.Sprintf(`"u%02d":{"description":"typeless"}`, i))
	}
	for i := 0; i < 10; i++ {
		props = append(props, fmt.Sprintf(`"k%02d":{"type":"string"}`, i))
	}
	props = append(props, `"req":{"type":"object","properties":{"z":{"type":"array"},"y":{}},"required":["y","z"]}`)
	fixture := `{"type":"object","properties":{` + strings.Join(props, ",") + `},"required":["k00"]}`

	var first Report
	var firstOut []byte
	for i := 0; i < 50; i++ {
		out, rep := Normalize(decodeJSON(t, fixture), defaultMaxOut)
		b, err := json.Marshal(out)
		if err != nil {
			t.Fatalf("could not marshal: %v", err)
		}
		if i == 0 {
			first, firstOut = rep, b
			if rep.DroppedPropsN != 31 || len(rep.DroppedProps) != 8 {
				t.Fatalf("fixture must drop 31 optional properties, got %d (%v)", rep.DroppedPropsN, rep.DroppedProps)
			}
			continue
		}
		if !reflect.DeepEqual(rep, first) {
			t.Fatalf("Wrong match. run %d report differs\nexpect: %+v\ngot: %+v", i, first, rep)
		}
		if string(b) != string(firstOut) {
			t.Fatalf("Wrong match. run %d output differs", i)
		}
	}

	// A dropped tool's DropPath is the first failure in sorted order.
	dropped := `{"type":"object","properties":{"b":{},"a":{},"c":{"type":"string"}},"required":["b","a"]}`
	for i := 0; i < 50; i++ {
		_, rep := Normalize(decodeJSON(t, dropped), defaultMaxOut)
		if rep.DropPath != "/properties/a" {
			t.Fatalf("Wrong match. run %d DropPath expect: /properties/a, got: %q", i, rep.DropPath)
		}
	}
}
