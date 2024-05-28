package request

// BodyStorageFilesPOST is rquest body define for
// POST /v1.0/storage_files
type BodyStorageFilesPOST struct {
	Name     string `json:"name,omitempty" binding:"omitempty"`
	Detail   string `json:"detail,omitempty" binding:"omitempty"`
	Filename string `json:"filename,omitempty" binding:"omitempty"`
}

// ParamFilesGET is rquest param define for
// GET /v1.0/storage_files
type ParamStorageFilesGET struct {
	Pagination
}
