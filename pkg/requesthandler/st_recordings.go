package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/response"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// STRecordingGet sends a request to storage-manager
// to getting a recording download link.
// it returns download link if it succeed.
func (r *requestHandler) STRecordingGet(id string) (string, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	res, err := r.sendRequestStorage(uri, rabbitmqhandler.RequestMethodGet, resourceStorageRecording, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return "", err
	case res == nil:
		// not found
		return "", fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return "", fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data response.STV1ResponseRecordingsIDGet
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return "", err
	}

	return data.URL, nil
}
