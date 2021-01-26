package request

// V1DataRecordingsGET is rquest param define for GET /recordings
type V1DataRecordingsGET struct {
	UserID uint64 `json:"user_id"`
	Pagination
}
