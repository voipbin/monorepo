package request

// ParamChatroommessagesGET is rquest param define for
// GET /v1.0/chatroommessages
type ParamChatroommessagesGET struct {
	ChatroomID string `form:"chatroom_id"`
	Pagination
}
