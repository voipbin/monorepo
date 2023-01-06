package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// v1RecordingsIDGet handles /v1/recordings/<id> POST request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1RecordingsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	tmpRecordingID, err := url.QueryUnescape(uriItems[3])
	if err != nil {
		return nil, fmt.Errorf("could not unescape the recording id. err: %v", err)
	}
	recordingID := uuid.FromStringOrNil(tmpRecordingID)

	// get recording
	rec, err := h.storageHandler.GetRecording(context.Background(), recordingID)
	if err != nil {
		logrus.Errorf("Could not get download url for recording. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(rec)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
