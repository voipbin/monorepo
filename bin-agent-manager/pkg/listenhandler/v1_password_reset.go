package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/listenhandler/models/request"
)

// processV1PasswordResetPost handles POST /v1/password-reset request
func (h *listenHandler) processV1PasswordResetPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1PasswordResetPost",
	})
	log.Debug("Executing processV1PasswordResetPost.")

	var reqData request.V1DataPasswordResetPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.agentHandler.PasswordReset(ctx, reqData.Token, reqData.Password); err != nil {
		log.Errorf("Could not process password reset. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
