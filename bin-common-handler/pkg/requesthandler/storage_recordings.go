package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"

	"github.com/gofrs/uuid"
)

// StorageV1RecordingGet sends a request to storage-manager
// to getting a recording download link.
// it returns download link if it succeed.
// requestTimeout: milliseconds
func (r *requestHandler) StorageV1RecordingGet(ctx context.Context, id uuid.UUID, requestTimeout int) (*smbucketfile.BucketFile, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	res, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/recording", requestTimeout, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data smbucketfile.BucketFile
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// StorageV1RecordingDelete sends a request to storage-manager
// to deleting a recording files.
// it returns error if it fails
func (r *requestHandler) StorageV1RecordingDelete(ctx context.Context, recordingID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/recordings/%s", recordingID)

	res, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodDelete, "storage/recording", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
