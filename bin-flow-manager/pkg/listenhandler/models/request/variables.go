package request

// V1DataVariablesIDVariablesPost is
// v1 data type request struct for
// /v1/variables/<variable-id>/variables POST
type V1DataVariablesIDVariablesPost struct {
	Variables map[string]string `json:"variables"`
}

// V1DataVariablesIDSubstitutePost is
// v1 data type request struct for
// /v1/variables/<variable-id>/substitute POST
type V1DataVariablesIDSubstitutePost struct {
	Data string `json:"data"`
}
