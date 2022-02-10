package request

// V1DataQueuesIDExecutePost is
// v1 data type request struct for
// /v1/queuecalls/<queuecall-id>/execute POST
type V1DataQueuesIDExecutePost struct {
	SearchDelay int `json:"search_delay"` // delay for start to search agent(ms)
}
