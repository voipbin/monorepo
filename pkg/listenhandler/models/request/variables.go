package request

// V1DataVariablesIDVariablesPost is
// v1 data type request struct for
// /v1/variables/<variable-id>/variables POST
type V1DataVariablesIDVariablesPost struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
