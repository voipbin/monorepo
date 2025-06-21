package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"strings"

	"github.com/sirupsen/logrus"
)

// processV1RecoveryPost handles /v1/recovery request
func (h *listenHandler) processV1RecoveryPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1RecoveryPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	var req request.V1DataRecoveryPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not marshal the request. err: %v", err)
		return nil, err
	}

	if errRecovery := h.callHandler.RecoveryStart(context.Background(), req.AsteriskID); errRecovery != nil {
		log.Errorf("Could not run recovery for asterisk ID %s. err: %v", req.AsteriskID, errRecovery)
		return nil, errRecovery
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
