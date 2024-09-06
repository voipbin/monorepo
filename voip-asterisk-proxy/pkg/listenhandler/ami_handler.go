package listenhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) listenHandlerAMI(request *sock.Request) (*sock.Response, error) {
	logrus.Debugf("listenHandlerAMI. message: %v", request)

	// send the request to Asterisk
	statusCode, resData, err := h.sendRequestToAsteriskAMI(request)
	if err != nil {
		return nil, err
	}

	response := &sock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       resData,
	}
	logrus.Debugf("listenHandlerAMI. result: %v", response)

	return response, nil
}

// sendRequestToAsteriskAMI sends the request to the Asterisk's AMI.
// returns status_code, response_message, error
func (h *listenHandler) sendRequestToAsteriskAMI(m *sock.Request) (int, []byte, error) {

	var req map[string]string
	if err := json.Unmarshal(m.Data, &req); err != nil {
		return 0, nil, err
	}

	tmp, err := h.amiSock.Action(req)
	if err != nil {
		return 0, nil, err
	}

	res, err := json.Marshal(tmp)
	if err != nil {
		return 0, nil, err
	}

	return 200, res, nil
}
