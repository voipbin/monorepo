package response

// BodyLoginPOST is response body define for POST /login
type BodyLoginPOST struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}
