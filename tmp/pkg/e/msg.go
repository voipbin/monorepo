package e

var MsgFlags = map[int]string{
	SUCCESS:        "ok",
	ERROR:          "fail",
	INVALID_PARAMS: "invalid parameters",

	ERROR_AUTH_CHECK_TOKEN_FAIL:    "token fail",
	ERROR_AUTH_CHECK_TOKEN_TIMEOUT: "token timeout",
	ERROR_AUTH_TOKEN:               "token failed",
	ERROR_AUTH:                     "auth failed",
}

// GetMsg get error information based on Code
func GetMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}

	return MsgFlags[ERROR]
}
