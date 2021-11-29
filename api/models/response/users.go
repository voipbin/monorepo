package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

// BodyUsersGET is response body define for GET /users
type BodyUsersGET struct {
	Result []*user.User `json:"result"`
	Pagination
}
