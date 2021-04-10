package webhookhandler

import (
	"bytes"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SendMessage sends the message to the given uri with the given method and data.
func (h *webhookHandler) SendMessage(uri string, method string, dataType string, data []byte) (*http.Response, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"uri":    uri,
			"method": method,
			"data":   data,
		},
	)

	// create request
	req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("Could not create request. err: %v", err)
		return nil, err
	}

	if data != nil && dataType != "" {
		req.Header.Set("Content-Type", string(dataType))
	}

	var resp *http.Response
	for i := 0; i < 3; i++ {
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			log.Errorf("Could not send the request correctly. Making a retrying: %d, err: %v", i, err)
			time.Sleep(time.Second * 1)
			continue
		}

		break
	}
	if err != nil {
		log.Errorf("Could not send the request. err: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.WithFields(
		logrus.Fields{
			"response": resp,
		},
	).Debugf("Sent the event correctly. response_status: %d", resp.StatusCode)
	return resp, nil
}
