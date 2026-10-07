package action

import (
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// ref.go exposes the `ref` struct-tag metadata on OptionXxx structs (see
// option.go's tags and meta.go's doc comment) as a small reflect-based API,
// so the Flow AI Builder (bin-ai-manager) and this package's own drift-lock
// test (ref_test.go) read the same authoritative source instead of each
// re-implementing the walk.

// RefKind says what a tagged option field refers to.
type RefKind string

const (
	RefKindAction   RefKind = "action"   // another action in the same Flow, by id
	RefKindResource RefKind = "resource" // a customer-owned resource (queue, assistance, flow, ...)
)

// RefField describes one top-level option field carrying a `ref` tag.
type RefField struct {
	// JSONName is the field's json tag name (the key the builder/LLM uses
	// in the option map), e.g. "target_id", "queue_id".
	JSONName string
	Kind     RefKind
	// IsMap is true for a map[string]uuid.UUID field (e.g.
	// OptionBranch.TargetIDs): the builder must resolve every value in
	// the map, not the field itself.
	IsMap bool
}

// RefFieldsOf returns every ref-tagged top-level option field for t, reading
// OptionStructByType via reflection. An unknown type or a type with no
// struct option (mute, stop: struct{}{}) returns nil.
func RefFieldsOf(t Type) []RefField {
	optAny, ok := OptionStructByType[t]
	if !ok {
		return nil
	}
	optType := reflect.TypeOf(optAny)
	if optType.Kind() != reflect.Struct {
		return nil
	}

	uuidType := reflect.TypeOf(uuid.UUID{})
	mapUUIDType := reflect.TypeOf(map[string]uuid.UUID{})

	var out []RefField
	for i := 0; i < optType.NumField(); i++ {
		f := optType.Field(i)
		tag := f.Tag.Get("ref")
		if tag == "" {
			continue
		}
		jsonName := strings.Split(f.Tag.Get("json"), ",")[0]
		kind := RefKind(tag)
		switch f.Type {
		case uuidType:
			out = append(out, RefField{JSONName: jsonName, Kind: kind})
		case mapUUIDType:
			out = append(out, RefField{JSONName: jsonName, Kind: kind, IsMap: true})
		}
	}
	return out
}
