package listenhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

func (h *listenHandler) listenHandlerAMI(request *rabbitmq.Request) (*rabbitmq.Response, error) {
	logrus.Debugf("listenHandlerAMI. message: %v", request)

	// send the request to Asterisk
	statusCode, resData, err := h.sendRequestToAsteriskAMI(request)
	if err != nil {
		return nil, err
	}

	response := &rabbitmq.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       resData,
	}
	logrus.Debugf("listenHandlerAMI. result: %v", response)

	return response, nil
}

// sendRequestToAsteriskAMI sends the request to the Asterisk's AMI.
// returns status_code, response_message, error
func (h *listenHandler) sendRequestToAsteriskAMI(m *rabbitmq.Request) (int, string, error) {

	var req map[string]string
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return 0, "", err
	}

	tmp, err := h.amiSock.Action(req)
	if err != nil {
		return 0, "", err
	}

	res, err := json.Marshal(tmp)
	if err != nil {
		return 0, "", err
	}

	return 200, string(res), nil
}
