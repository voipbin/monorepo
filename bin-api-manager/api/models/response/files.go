package response

import smfile "monorepo/bin-storage-manager/models/file"

// BodyFilesGET is rquest body define for
// GET /v1.0/files
type BodyFilesGET struct {
	Result []*smfile.WebhookMessage `json:"result"`
	Pagination
}
