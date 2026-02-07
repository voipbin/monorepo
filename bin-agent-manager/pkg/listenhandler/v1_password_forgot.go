package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/listenhandler/models/request"
)

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

	if err := h.agentHandler.PasswordForgot(ctx, reqData.Username, agenthandler.PasswordResetEmailTypeForgot); err != nil {
		log.Infof("Could not process password forgot. err: %v", err)
		return simpleResponse(404), nil
	}

	return simpleResponse(200), nil
}
