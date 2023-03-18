package request

// Pagination is pagination structure for request
type Pagination struct {
	PageSize  uint64 `form:"page_size" json:"page_size,omitempty"`
	PageToken string `form:"page_token" json:"page_token,omitempty"`
}
