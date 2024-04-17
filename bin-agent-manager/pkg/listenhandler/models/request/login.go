package request

// V1DataLoginPost is
// v1 data type request struct for
// /v1/login POST
type V1DataLoginPost struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
