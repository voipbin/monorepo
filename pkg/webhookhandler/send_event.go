package webhookhandler

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SendEvent sends the webhook event to the given uri with the given method and data.
func (h *webhookHandler) SendEvent(uri string, method MethodType, dataType DataType, data []byte) (*http.Response, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"uri":    uri,
			"method": method,
			"data":   data,
		},
	)

	// create request
	req, err := http.NewRequest(string(method), uri, bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("Could not create request. err: %v", err)
		return nil, err
	}

	if data != nil && dataType != DataTypeEmpty {
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

func (h *webhookHandler) Test() {
	fmt.Printf("hello world")
}
