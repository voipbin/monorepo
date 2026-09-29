// Package mcpschema rewrites a customer MCP tool's input schema into a
// provider-neutral JSON Schema subset before it is advertised to an LLM
// (design docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md section 15).
//
// It is a pure package: no handler, no I/O, standard library only.
package mcpschema

// Reason* are the fixed values of Report.DropReason for a dropped tool.
const (
	// ReasonRootNotObject: the root schema is not an object.
	ReasonRootNotObject = "root_not_object"
	// ReasonBadType: a type that is not a JSON Schema type string, or an
	// empty type list.
	ReasonBadType = "bad_type"
	// ReasonBadShape: a subschema that is not a JSON object (for example a
	// boolean subschema).
	ReasonBadShape = "bad_shape"
	// ReasonTypeless: a subschema with no type and nothing to infer one from
	// ("any JSON value").
	ReasonTypeless = "typeless"
	// ReasonArrayItems: an array without a single object items subschema.
	ReasonArrayItems = "array_without_items"
	// ReasonAnyOf: an anyOf left with no usable non-null member.
	ReasonAnyOf = "anyof_without_usable_member"
	// ReasonAllOf: an allOf with two or more members.
	ReasonAllOf = "allof_multiple_members"
	// ReasonRef: a $ref that is not local, is missing, is cyclic, or exceeds
	// the expansion limits.
	ReasonRef = "unresolvable_ref"
	// ReasonTooDeep: a subschema nested deeper than maxDepth.
	ReasonTooDeep = "too_deep"
	// ReasonTooLarge: the node cap or the per-tool output cap was exceeded.
	ReasonTooLarge = "too_large"
)

// Report describes what Normalize changed. Paths are JSON-pointer-like
// ("/properties/owner/x-mcp-header"). It never contains schema values.
type Report struct {
	ToolDropped   bool
	DropReason    string   // one of the Reason* constants, "" when kept
	DropPath      string   // where the fatal unusable subschema was found
	DroppedProps  []string // first maxReportedPaths paths of optional properties removed (cascade)
	DroppedPropsN int      // total count of optional properties removed (may exceed len(DroppedProps))
	DroppedKeys   int      // count of non-allowlisted keys removed
	Rewrites      int      // oneOf, const, allOf, $ref, list type, type inference
	OutBytes      int      // output charge; 0 for a nil schema
}

// JSON Schema type names.
const (
	typeString  = "string"
	typeNumber  = "number"
	typeInteger = "integer"
	typeBoolean = "boolean"
	typeObject  = "object"
	typeArray   = "array"
	typeNull    = "null"
)

// Limits (design section 15.3 R5 and R12).
const (
	// maxReportedPaths bounds Report.DroppedProps.
	maxReportedPaths = 8
	// maxDepth is the deepest subschema nesting level kept. One level is one
	// step into a properties value, items, or an anyOf/oneOf member; a $ref
	// expansion adds none.
	maxDepth = 32
	// maxRefDepth bounds nested $ref expansions on any one path.
	maxRefDepth = 8
	// maxRefExpansions bounds $ref expansions per tool.
	maxRefExpansions = 256
	// maxNodes bounds emitted subschemas per tool.
	maxNodes = 4096
)

// Output charge (R12): an estimate of the output size that also bounds the
// work done, since every visited subschema is charged whether it is emitted
// or later dropped. Emitted strings cost their UTF-8 length plus 3.
const (
	visitBytes  = 32
	numberBytes = 24
)

// validTypes are the type strings a subschema may carry (R2).
var validTypes = map[string]bool{
	typeString: true, typeNumber: true, typeInteger: true, typeBoolean: true,
	typeObject: true, typeArray: true, typeNull: true,
}

// keepKeys are the only keys ever emitted (R1).
var keepKeys = map[string]bool{
	"type": true, "description": true, "properties": true, "required": true,
	"items": true, "enum": true, "anyOf": true, "format": true,
	"minimum": true, "maximum": true, "minItems": true, "maxItems": true,
}

// consumedKeys are converted by R5 and never emitted, so they are not
// counted as dropped.
var consumedKeys = map[string]bool{
	"oneOf": true, "allOf": true, "const": true, "$ref": true,
}
