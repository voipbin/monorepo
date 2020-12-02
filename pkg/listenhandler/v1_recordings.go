package listenhandler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/listenhandler/models/response"
)

// v1RecordingsIDGet handles /v1/recordings/<id> POST request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1RecordingsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	tmpRecordingID := uriItems[3]
	recordingID, err := url.QueryUnescape(tmpRecordingID)
	if err != nil {
		return nil, fmt.Errorf("could not unescape the recording id. err: %v", err)
	}

	// get recording
	url, err := h.bucketHandler.RecordingGetDownloadURL(recordingID, time.Now().Add(24*time.Hour))
	if err != nil {
		logrus.Errorf("Could not get download url for recording. err: %v", err)
		return nil, err
	}

	resMsg := &response.V1ResponseRecordingsIDGet{
		URL: url,
	}

	data, err := json.Marshal(resMsg)
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
