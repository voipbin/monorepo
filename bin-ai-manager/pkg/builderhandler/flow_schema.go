package builderhandler

import (
	"encoding/json"
	"sort"

	fmaction "monorepo/bin-flow-manager/models/action"
)

// FlowResponseSchemaName is the json_schema name sent to the provider.
const FlowResponseSchemaName = "flow_builder_turn"

// FlowResponseSchema builds the response_format schema for one request. The
// node "type" enum is the request's allowed set (design doc 2.5), so the
// model sees only types it may use. Strict is false (the Gemini compatible
// endpoint does not enforce it), so FlowParse still validates everything.
// "message" is declared first so a truncated answer keeps the useful part.
func FlowResponseSchema(allowed []fmaction.Type) string {
	names := make([]string, 0, len(allowed))
	for _, t := range allowed {
		names = append(names, string(t))
	}
	sort.Strings(names)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string"},
			"draft": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nodes": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"label":  map[string]any{"type": "string"},
								"type":   map[string]any{"type": "string", "enum": names},
								"option": map[string]any{"type": "object"},
								"next":   map[string]any{"type": "string"},
							},
							"required": []string{"label", "type"},
						},
					},
				},
				"required": []string{"nodes"},
			},
			"assumptions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"message"},
	}
	b, _ := json.Marshal(schema) // a map of plain values cannot fail to marshal
	return string(b)
}
