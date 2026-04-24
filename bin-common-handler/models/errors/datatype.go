package errors

// DataTypeVoipbinError is the sock.Response.DataType value used by
// internal managers to signal that Data contains a JSON-serialized
// VoipbinError. bin-api-manager's requesthandler inspects this to
// decide whether to unmarshal the payload as a typed error.
const DataTypeVoipbinError = "voipbin_error"
