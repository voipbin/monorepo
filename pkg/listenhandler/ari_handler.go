package listenhandler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ariSendRequestToAsterisk sends the request to the Asterisk's ARI.
// returns status_code, response_message, error
func (h *listenHandler) ariSendRequestToAsterisk(m *rabbitmqhandler.Request) (int, []byte, error) {
	url := fmt.Sprintf("http://%s%s", h.ariAddr, m.URI)
	logrus.WithFields(logrus.Fields{
		"request": m,
	}).Debug("Sending ARI request.")

	req, err := http.NewRequest(string(m.Method), url, bytes.NewReader(m.Data))
	if err != nil {
		return 0, nil, err
	}

	// basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(h.ariAccount))
	req.Header.Add("Authorization", "Basic "+auth)

	// content-type
	if m.DataType != "" {
		req.Header.Set("Content-Type", m.DataType)
	}

	// send
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	res, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, res, nil
}

func (h *listenHandler) listenHandlerARI(request *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	// send the request to Asterisk
	statusCode, resData, err := h.ariSendRequestToAsterisk(request)
	if err != nil {
		return nil, err
	}

	response := &rabbitmqhandler.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       resData,
	}

	return response, nil
}
