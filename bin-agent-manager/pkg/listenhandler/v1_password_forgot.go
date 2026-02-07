package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/listenhandler/models/request"
)

// passwordForgotResponse is the response struct for password forgot
type passwordForgotResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// processV1PasswordForgotPost handles POST /v1/password-forgot request
func (h *listenHandler) processV1PasswordForgotPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1PasswordForgotPost",
	})
	log.Debug("Executing processV1PasswordForgotPost.")

	var reqData request.V1DataPasswordForgotPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	token, username, err := h.agentHandler.PasswordForgot(ctx, reqData.Username)
	if err != nil {
		log.Infof("Could not process password forgot. err: %v", err)
		return simpleResponse(404), nil
	}

	resp := passwordForgotResponse{
		Token:    token,
		Username: username,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", resp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
