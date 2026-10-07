package action

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

// TestEveryUUIDFieldIsTagged is the drift-lock for the Flow AI Builder's
// `ref` struct tag scheme (VOIP-1573 design doc §2.3). It walks every
// OptionStructByType entry and requires:
//
//   - every top-level uuid.UUID field, and every map[string]uuid.UUID field,
//     carries a `ref:"action"` or `ref:"resource"` tag;
//   - any type whose option carries a UUID nested inside a slice or struct
//     field (a shape the builder's label<->UUID transcoder does not reach)
//     is listed in BuilderExcludedTypes, so exclusion is visible instead of
//     silently mis-tagged.
//
// A field added later with no tag, or a new nested-UUID shape on a type that
// is not excluded, fails this test — see the design doc for why the
// transcoder only walks scalar and map-value uuid.UUID fields.
func Test_EveryUUIDFieldIsTagged(t *testing.T) {
	uuidType := reflect.TypeOf(uuid.UUID{})
	mapUUIDType := reflect.TypeOf(map[string]uuid.UUID{})

	for ty, optAny := range OptionStructByType {
		optType := reflect.TypeOf(optAny)
		if optType.Kind() != reflect.Struct {
			continue // e.g. struct{}{} for mute/stop: no fields, nothing to check
		}

		hasNestedUUID := false
		for i := 0; i < optType.NumField(); i++ {
			f := optType.Field(i)

			switch f.Type {
			case uuidType, mapUUIDType:
				tag := f.Tag.Get("ref")
				if tag != "action" && tag != "resource" {
					t.Errorf("%s.%s is a %s field with ref tag %q; want `ref:\"action\"` or `ref:\"resource\"` (VOIP-1573)",
						optType.Name(), f.Name, f.Type, tag)
				}
				continue
			}

			if fieldContainsUUID(f.Type) {
				hasNestedUUID = true
			}
		}

		if hasNestedUUID && !BuilderExcludedTypes[ty] {
			t.Errorf("%s (action type %q) has a UUID nested inside a slice/struct field the builder transcoder does not reach; add it to BuilderExcludedTypes or flatten the field (VOIP-1573)",
				optType.Name(), ty)
		}
	}
}

// fieldContainsUUID reports whether t is a slice/array of structs, or a
// pointer to or plain struct type, that itself (one level deep) contains a
// uuid.UUID field. It intentionally does not recurse further — one level is
// enough to catch OptionCall.Actions ([]Action, which has an ID/NextID
// uuid.UUID) and OptionEmailSend.Attachments ([]ememail.Attachment, which
// has a ReferenceID uuid.UUID).
func fieldContainsUUID(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return fieldContainsUUID(t.Elem())
	case reflect.Pointer:
		return fieldContainsUUID(t.Elem())
	case reflect.Struct:
		uuidType := reflect.TypeOf(uuid.UUID{})
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Type == uuidType {
				return true
			}
		}
		return false
	default:
		return false
	}
}
