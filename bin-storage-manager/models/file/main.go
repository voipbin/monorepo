package file

import "github.com/gofrs/uuid"

// File defines
type File struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	URIBucket   string `json:"uri_bucket"`   // uri for bucket
	URIDownload string `json:"uri_download"` // uri for download

	TMDownloadExpire string `json:"tm_download_expire"`
	TMCreate         string `json:"tm_create"`
	TMUpdate         string `json:"tm_update"`
	TMDelete         string `json:"tm_delete"`
}
