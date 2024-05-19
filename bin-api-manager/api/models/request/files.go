package request

// BodyFilesPOST is rquest body define for
// POST /v1.0/files
type BodyFilesPOST struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`
}
