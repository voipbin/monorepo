package request

// V1DataChatroomsIDPut is
// v1 data type request struct for
// /v1/chatrooms/{id} PUT
type V1DataChatroomsIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
}
