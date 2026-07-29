package ai_test

import (
	"testing"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_FieldStruct_TypeFilterRoundTrips(t *testing.T) {
	filters, err := utilhandler.ConvertFilters[ai.FieldStruct, ai.Field](ai.FieldStruct{}, map[string]any{
		"type": "insight",
	})
	if err != nil {
		t.Fatalf("ConvertFilters() unexpected error: %v", err)
	}

	got, ok := filters[ai.FieldType]
	if !ok {
		t.Fatalf("ConvertFilters() dropped the type filter -- FieldStruct is missing a Type field")
	}
	// ConvertFilters' string-kind branch has no reflect.Convert step for
	// named string types -- it returns the dynamic type plain `string`, not
	// `ai.Type`, even though the FieldStruct field is typed `ai.Type`. This
	// is expected, harmless behavior (squirrel/ApplyFields bind the string
	// value correctly either way) -- assert against the underlying string,
	// not the named type.
	if got != string(ai.TypeInsight) {
		t.Errorf("ConvertFilters() type filter = %v, want %v", got, string(ai.TypeInsight))
	}
}

// ConvertFilters treats FieldStruct as an allowlist: an undeclared key is
// dropped silently, with no error. Without an IsInsightActive entry, the Case
// panel's dedicated "which Insight AI is active?" query would degrade to an
// unfiltered most-recently-created list and the whole feature would go inert.
func Test_FieldStruct_IsInsightActiveFilterRoundTrips(t *testing.T) {
	for _, want := range []bool{true, false} {
		filters, err := utilhandler.ConvertFilters[ai.FieldStruct, ai.Field](ai.FieldStruct{}, map[string]any{
			"is_insight_active": want,
		})
		if err != nil {
			t.Fatalf("ConvertFilters() unexpected error: %v", err)
		}

		got, ok := filters[ai.FieldIsInsightActive]
		if !ok {
			t.Fatalf("ConvertFilters() dropped the is_insight_active filter -- FieldStruct is missing an IsInsightActive field")
		}
		if got != want {
			t.Errorf("ConvertFilters() is_insight_active filter = %v, want %v", got, want)
		}
	}
}
