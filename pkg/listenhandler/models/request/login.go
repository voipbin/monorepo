package request

// V1DataLogin is
// v1 data type request struct for
// /v1/login POST
type V1DataLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
