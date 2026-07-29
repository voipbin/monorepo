package ai

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for AI queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Name       string    `filter:"name"`
	Detail     string    `filter:"detail"`
	Type       Type      `filter:"type"`

	// IsInsightActive must stay declared here: utilhandler.ConvertFilters treats
	// FieldStruct as an allowlist and silently drops any filter key that is not
	// declared, so omitting it would make an is_insight_active=true query
	// degrade to an unfiltered list with no error.
	IsInsightActive bool `filter:"is_insight_active"`

	EngineModel EngineModel `filter:"engine_model"`
	TTSType     TTSType     `filter:"tts_type"`
	STTType     STTType     `filter:"stt_type"`
	STTLanguage string      `filter:"stt_language"`
	Deleted     bool        `filter:"deleted"`
}
