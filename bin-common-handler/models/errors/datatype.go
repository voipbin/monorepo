package errors

// DataTypeVoipbinError is the sock.Response.DataType value used by
// internal managers to signal that Data contains a JSON-serialized
// VoipbinError. bin-api-manager's requesthandler inspects this to
// decide whether to unmarshal the payload as a typed error.
//
// Wire-format contract: once this DataType is in use on main, the
// JSON shape is locked. Changes must be ADDITIVE-ONLY (new fields
// with omitempty). Removing or renaming a field requires bumping to a
// new DataType (e.g., "voipbin_error_v2") so old consumers continue
// to parse old payloads.
const DataTypeVoipbinError = "voipbin_error"
