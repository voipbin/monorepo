package request

// V1DataPasswordForgotPost is
// v1 data type request struct for
// /v1/password-forgot POST
type V1DataPasswordForgotPost struct {
	Username string `json:"username"`
}

// V1DataPasswordResetPost is
// v1 data type request struct for
// /v1/password-reset POST
type V1DataPasswordResetPost struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
