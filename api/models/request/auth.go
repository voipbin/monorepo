package request

// BodyLoginPOST is rquest body define for
// POST /login
type BodyLoginPOST struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
