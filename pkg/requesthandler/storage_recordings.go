package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	smbucketfile "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// StorageV1RecordingGet sends a request to storage-manager
// to getting a recording download link.
// it returns download link if it succeed.
// requestTimeout: milliseconds
func (r *requestHandler) StorageV1RecordingGet(ctx context.Context, id uuid.UUID, requestTimeout int) (*smbucketfile.BucketFile, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	res, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceStorageRecording, requestTimeout, 0, ContentTypeJSON, nil)
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
