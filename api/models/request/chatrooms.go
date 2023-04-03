package request

// ParamChatroomsGET is rquest param define for
// GET /v1.0/chatrooms
type ParamChatroomsGET struct {
	OwnerID string `form:"owner_id"`
	Pagination
}
