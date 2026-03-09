package gcsuploader

//go:generate mockgen -source=main.go -destination=mock_main.go -package=gcsuploader

// Uploader uploads files to GCS.
type Uploader interface {
	// Upload uploads a local file to the specified GCS object path.
	// Returns the GCS URI (gs://bucket/path) on success.
	Upload(localPath string, objectPath string) (string, error)

	// Close releases resources.
	Close() error
}
