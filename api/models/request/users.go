package request

// ParamUsersGET is rquest param define for GET /users
type ParamUsersGET struct {
	Pagination
}

// BodyUsersPOST is rquest body define for POST /users
type BodyUsersPOST struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	Permission uint64 `json:"permission"`
}

// BodyUsersIDPUT is rquest body define for PUT /users/<user-id>
type BodyUsersIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyUsersIDPasswordPUT is rquest body define for PUT /users/<user-id>/password
type BodyUsersIDPasswordPUT struct {
	Password string `json:"password"`
}

// BodyUsersIDPermissionPUT is rquest body define for PUT /users/<user-id>/permission
type BodyUsersIDPermissionPUT struct {
	Permission uint64 `json:"permission"`
}
