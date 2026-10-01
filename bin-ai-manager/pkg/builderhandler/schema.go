package builderhandler

import (
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/builder"
)

// ResponseSchema is the JSON Schema sent as response_format json_schema
// (Strict:false: the Gemini OpenAI-compatible endpoint does not enforce strict
// mode, so Parse still validates everything). "message" is declared first so a
// truncated response keeps the useful part. v1 has no suggested_replies,
// captured or open_topics (design 2.4). tool_names is constrained to the
// allow-list here as well as in Parse.
var ResponseSchema = buildResponseSchema()

// ResponseSchemaName is the json_schema name sent to the provider.
const ResponseSchemaName = "builder_turn"

func buildResponseSchema() string {
	var enum []string
	for _, t := range builder.AllowedTools {
		enum = append(enum, fmt.Sprintf("%q", string(t)))
	}
	return `{
  "type": "object",
  "properties": {
    "message": {"type": "string"},
    "draft": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "detail": {"type": "string"},
        "init_prompt": {"type": "string"},
        "tool_names": {"type": "array", "items": {"type": "string", "enum": [` + strings.Join(enum, ", ") + `]}}
      },
      "required": ["name", "init_prompt"]
    },
    "assumptions": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["message"]
}`
}
