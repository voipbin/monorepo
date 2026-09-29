package mcpschema

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// refChain builds a root whose property x reaches a string through n
// nested $ref expansions.
func refChain(n int) string {
	defs := make([]string, 0, n)
	for i := 1; i < n; i++ {
		defs = append(defs, fmt.Sprintf(`"D%d":{"$ref":"#/$defs/D%d"}`, i, i+1))
	}
	defs = append(defs, fmt.Sprintf(`"D%d":{"type":"string"}`, n))
	return `{"type":"object","$defs":{` + strings.Join(defs, ",") + `},"properties":{"x":{"$ref":"#/$defs/D1"}}}`
}

// refFanOut builds a root with n properties p000.. each a $ref to one string.
func refFanOut(n int) string {
	props := make([]string, 0, n)
	for i := 0; i < n; i++ {
		props = append(props, fmt.Sprintf(`"p%03d":{"$ref":"#/$defs/T"}`, i))
	}
	return `{"type":"object","$defs":{"T":{"type":"string"}},"properties":{` + strings.Join(props, ",") + `}}`
}

// refFanOutWant is the normalized form of the first n properties of refFanOut.
func refFanOutWant(n int) string {
	props := make([]string, 0, n)
	for i := 0; i < n; i++ {
		props = append(props, fmt.Sprintf(`"p%03d":{"type":"string"}`, i))
	}
	return `{"type":"object","properties":{` + strings.Join(props, ",") + `}}`
}

// nestedObjects builds objects nested d levels deep (l1 at depth 1 ... ld at
// depth d). The leaf is {type: object}, or a $ref to #/$defs/A when defs is
// set (defs is spliced into the root as is).
func nestedObjects(d int, defs string) string {
	leaf := `{"type":"object"}`
	if defs != "" {
		leaf = `{"$ref":"#/$defs/A"}`
	}
	s := leaf
	for i := d - 1; i >= 1; i-- {
		s = fmt.Sprintf(`{"type":"object","properties":{"l%d":%s}}`, i+1, s)
	}
	return `{"type":"object",` + defs + `"properties":{"l1":` + s + `}}`
}

// nestedObjectsWant is the normalized form of objects nested d levels deep
// with a free-form leaf.
func nestedObjectsWant(d int) string {
	return nestedObjects(d, "")
}

// nestedPath is the path of level d in nestedObjects.
func nestedPath(d int) string {
	var b strings.Builder
	for i := 1; i <= d; i++ {
		fmt.Fprintf(&b, "/properties/l%d", i)
	}
	return b.String()
}

// longDescription builds a root with one string property whose description
// is n bytes long.
func longDescription(n int) string {
	return `{"type":"object","properties":{"d":{"type":"string","description":"` + strings.Repeat("x", n) + `"}}}`
}

// manyProperties builds a root with n string properties.
func manyProperties(n int) string {
	props := make([]string, 0, n)
	for i := 0; i < n; i++ {
		props = append(props, fmt.Sprintf(`"p%05d":{"type":"string"}`, i))
	}
	return `{"type":"object","properties":{` + strings.Join(props, ",") + `}}`
}

// refAmplification builds the rr1_s15_ref_amplification.py shape: one $defs
// string with a 56,000-byte description and 256 properties each a $ref to
// it. It stays under the 64 KiB raw limit and within maxRefExpansions.
func refAmplification(t *testing.T) string {
	props := map[string]any{}
	for i := 0; i < 256; i++ {
		props[fmt.Sprintf("p%d", i)] = map[string]any{"$ref": "#/$defs/big"}
	}
	return mustRaw(t, map[string]any{
		"type":       "object",
		"$defs":      map[string]any{"big": map[string]any{"type": "string", "description": strings.Repeat("x", 56000)}},
		"properties": props,
	})
}

// droppedPropsAmplification builds the rr2_s15_droppedprops_amplification.py
// shape: one $def with 5,411 typeless optional properties, referenced 256
// times. It stays under the 64 KiB raw limit and within maxRefExpansions.
func droppedPropsAmplification(t *testing.T) string {
	inner := map[string]any{}
	for i := 0; i < 5411; i++ {
		inner[fmt.Sprintf("a%d", i)] = map[string]any{}
	}
	props := map[string]any{}
	for j := 0; j < 256; j++ {
		props[fmt.Sprintf("p%d", j)] = map[string]any{"$ref": "#/$defs/T"}
	}
	return mustRaw(t, map[string]any{
		"type":       "object",
		"$defs":      map[string]any{"T": map[string]any{"properties": inner}},
		"properties": props,
	})
}

// allOfChainFanOut builds a $def that is an n-level chain of single-member
// allOf wrappers, each level carrying one non-allowlisted sibling key, and
// refs properties referencing it. Every chain step copies the whole merged
// map, so the work is quadratic in n and multiplied by refs, while nothing
// is emitted but the leaf type. It stays under the 64 KiB raw limit.
func allOfChainFanOut(t *testing.T, n, refs int) string {
	var sb strings.Builder
	sb.WriteString(`{"type":"object","$defs":{"D":`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `{"k%d":1,"allOf":[`, i)
	}
	sb.WriteString(`{"type":"string"}`)
	sb.WriteString(strings.Repeat(`]}`, n))
	sb.WriteString(`},"properties":{`)
	for i := 0; i < refs; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"%d":{"$ref":"#/$defs/D"}`, i)
	}
	sb.WriteString(`}}`)
	raw := sb.String()
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	return raw
}

// listFanOut builds a $def that carries one n-entry list under key, the kind
// of list the normalizer scans on every visit, referenced by refs
// properties. The list is valid enough to be scanned in full and its $def
// stays usable or is dropped only after the scan:
//   - required: n distinct names on an object with one property;
//   - enum: n "a" strings then a number on a string (rejected at the end);
//   - anyOf: n typed members plus a trailing typeless member on a string;
//   - type: a list type of "string" then n-1 "null" entries.
//
// It stays under the 64 KiB raw limit.
func listFanOut(t *testing.T, key string, n, refs int) string {
	var def string
	switch key {
	case "required":
		names := make([]string, n)
		for i := range names {
			names[i] = fmt.Sprintf(`"%x"`, i)
		}
		def = `{"type":"object","properties":{"a":{"type":"string"}},"required":[` + strings.Join(names, ",") + `]}`
	case "enum":
		def = `{"type":"string","enum":[` + strings.Repeat(`"a",`, n) + `0]}`
	case "anyOf":
		def = `{"type":"string","anyOf":[` + strings.Repeat(`{"type":"string"},`, n) + `{}]}`
	case "type":
		def = `{"type":["string"` + strings.Repeat(`,"null"`, n-1) + `]}`
	}
	props := make([]string, refs)
	for i := range props {
		props[i] = fmt.Sprintf(`"%d":{"$ref":"#/$defs/D"}`, i)
	}
	raw := `{"type":"object","$defs":{"D":` + def + `},"properties":{` + strings.Join(props, ",") + `}}`
	if len(raw) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(raw))
	}
	return raw
}

// mustRaw marshals v compactly and asserts it fits the 64 KiB raw input
// limit, so the shape is one decodeToolSchema would actually admit.
func mustRaw(t *testing.T, v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal: %v", err)
	}
	if len(b) > 64<<10 {
		t.Fatalf("shape is %d raw bytes, over the 64 KiB input limit", len(b))
	}
	return string(b)
}
