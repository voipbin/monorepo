package requesthandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	smrecordingpeak "monorepo/bin-storage-manager/models/recordingpeak"

	"github.com/gofrs/uuid"
)

// StorageV1RecordingGet sends a request to storage-manager
// to getting a recording download link.
// it returns download link if it succeed.
// requestTimeout: milliseconds
func (r *requestHandler) StorageV1RecordingGet(ctx context.Context, id uuid.UUID, requestTimeout int) (*smbucketfile.BucketFile, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/recording", requestTimeout, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res smbucketfile.BucketFile
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// StorageV1RecordingPeaks sends a request to storage-manager to get the
// recording's per-file waveform peaks and durations, keyed by filename.
// requestTimeout: milliseconds
func (r *requestHandler) StorageV1RecordingPeaks(ctx context.Context, id uuid.UUID, requestTimeout int) (map[string]smrecordingpeak.RecordingFilePeak, error) {
	uri := fmt.Sprintf("/v1/recordings/%s/peaks", id)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/recording", requestTimeout, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res map[string]smrecordingpeak.RecordingFilePeak
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// StorageV1RecordingDelete sends a request to storage-manager
// to deleting a recording files.
// it returns error if it fails
func (r *requestHandler) StorageV1RecordingDelete(ctx context.Context, recordingID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/recordings/%s", recordingID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodDelete, "storage/recording", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
