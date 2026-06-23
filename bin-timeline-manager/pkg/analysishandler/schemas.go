package analysishandler

import "encoding/json"

// JSON schemas for the gateway's response_format=json_schema (Strict:true).
// Every object sets additionalProperties:false and lists ALL properties in
// required (OpenAI strict json_schema requirement).

// verdictSchema is the final diagnostic verdict (matches verdict.RawVerdict).
var verdictSchema = json.RawMessage(`{
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
