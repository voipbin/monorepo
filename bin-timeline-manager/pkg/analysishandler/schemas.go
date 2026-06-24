package analysishandler

import "encoding/json"

// JSON schemas for the gateway's response_format=json_schema (Strict:true).
// Every object sets additionalProperties:false and lists ALL properties in
// required (OpenAI strict json_schema requirement).

// stage3VerdictSchema is the staged-path diagnosis schema (Stage 3). It is the
// diagnosis verdict WITHOUT interactions: on the staged path interactions are
// carried forward from the Stage 2 content pass, so forcing Stage 3 to re-emit
// them would waste tokens and let Stage 3 paraphrase/lose the Stage 2 detail.
// This is byte-identical to the pre-VOIP-1200 verdict schema.
var stage3VerdictSchema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["overall_status", "resources_used", "narrative", "issues"],
  "properties": {
    "overall_status": { "type": "string", "enum": ["ok", "warning", "error"] },
    "resources_used": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["type", "count"],
        "properties": {
          "type": { "type": "string" },
          "count": { "type": "integer" }
        }
      }
    },
    "narrative": { "type": "string" },
    "issues": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["severity", "area", "summary", "evidence_index"],
        "properties": {
          "severity": { "type": "string", "enum": ["info", "warning", "error"] },
          "area": { "type": "string" },
          "summary": { "type": "string" },
          "evidence_index": {
            "type": "array",
            "items": { "type": "integer" }
          }
        }
      }
    }
  }
}`)

// interactionsProperty is the single property that distinguishes the combined
// (single-call) verdict schema from the staged diagnosis schema. It is injected
// into stage3VerdictSchema at init to derive verdictSchema, so the two schemas
// share one source of truth for the common diagnosis fields (no drift).
var interactionsProperty = json.RawMessage(`{
  "type": "array",
  "items": {
    "type": "object",
    "additionalProperties": false,
    "required": ["resource_type", "summary"],
    "properties": {
      "resource_type": { "type": "string" },
      "summary": { "type": "string" }
    }
  }
}`)

// verdictSchema is the combined / single-call verdict schema: the Stage 3
// diagnosis schema PLUS the interactions property (in properties AND required,
// as OpenAI/Gemini strict json_schema demands). It is derived from
// stage3VerdictSchema at init so the shared diagnosis fields never drift.
var verdictSchema = mustInjectInteractions(stage3VerdictSchema, interactionsProperty)

// mustInjectInteractions returns base with the interactions property added to
// .properties and "interactions" appended to .required. It panics on malformed
// input because both inputs are compile-time-constant literals in this package;
// a panic here is a build-time programming error, never a runtime data error.
func mustInjectInteractions(base, interactions json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(base, &doc); err != nil {
		panic("verdict schema: base is not a JSON object: " + err.Error())
	}

	var props map[string]json.RawMessage
	if err := json.Unmarshal(doc["properties"], &props); err != nil {
		panic("verdict schema: properties is not a JSON object: " + err.Error())
	}
	props["interactions"] = interactions
	propsRaw, err := json.Marshal(props)
	if err != nil {
		panic("verdict schema: could not marshal properties: " + err.Error())
	}
	doc["properties"] = propsRaw

	var required []string
	if err := json.Unmarshal(doc["required"], &required); err != nil {
		panic("verdict schema: required is not a JSON array: " + err.Error())
	}
	required = append(required, "interactions")
	requiredRaw, err := json.Marshal(required)
	if err != nil {
		panic("verdict schema: could not marshal required: " + err.Error())
	}
	doc["required"] = requiredRaw

	out, err := json.Marshal(doc)
	if err != nil {
		panic("verdict schema: could not marshal derived schema: " + err.Error())
	}
	return out
}

// stage1Schema — inventory + indexed event outline.
var stage1Schema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["resources_used", "event_outline"],
  "properties": {
    "resources_used": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["type", "count"],
        "properties": {
          "type": { "type": "string" },
          "count": { "type": "integer" }
        }
      }
    },
    "event_outline": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["evidence_index", "label"],
        "properties": {
          "evidence_index": { "type": "integer" },
          "label": { "type": "string" }
        }
      }
    }
  }
}`)

// stage2Schema — content summary + overall narrative.
var stage2Schema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["interactions", "overall_narrative"],
  "properties": {
    "interactions": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["resource_type", "summary"],
        "properties": {
          "resource_type": { "type": "string" },
          "summary": { "type": "string" }
        }
      }
    },
    "overall_narrative": { "type": "string" }
  }
}`)
