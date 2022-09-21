package request

// ParamChatroomsGET is rquest param define for GET /chatrooms
type ParamChatroomsGET struct {
	OwnerID string `form:"owner_id"`
	Pagination
}
