package compress_file

import (
	"github.com/gofrs/uuid"
)

// BucketFile struct
type CompressFile struct {
	FileIDs []uuid.UUID `json:"file_ids"`

	DownloadURI      string `json:"download_uri"`       // download link
	TMDownloadExpire string `json:"tm_download_expire"` // timestamp for download link expire
}
