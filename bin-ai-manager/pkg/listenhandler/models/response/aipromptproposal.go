package response

// V1ResponseAIPromptProposalsExpire is
// v1 response type struct for
// /v1/aipromptproposals/expire POST
//
// Wraps the bare expired-row count — a scalar with no domain type (style B).
type V1ResponseAIPromptProposalsExpire struct {
	Expired int `json:"expired"`
}
