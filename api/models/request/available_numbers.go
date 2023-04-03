package request

// ParamAvailableNumbersGET is rquest param define for
// GET /v1.0/available_numbers
type ParamAvailableNumbersGET struct {
	PageSize   uint64 `form:"page_size"`
	CountyCode string `form:"country_code"`
}
