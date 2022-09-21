package request

// ParamChatroommessagesGET is rquest param define for GET /chatroommessages
type ParamChatroommessagesGET struct {
	ChatroomID string `form:"chatroom_id"`
	Pagination
}
