package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/pkg/listenhandler/models/request"
)

// processV1TransfersPost handles POST /v1/transfers request
func (h *listenHandler) processV1TransfersPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ServicesTypeTransferPost",
		"request": m,
	})

	var req request.V1DataTransfersPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// start the transfer
	tmp, err := h.transferHandler.ServiceStart(ctx, req.Type, req.TransfererCallID, req.TransfereeAddresses)
	if err != nil {
		log.Errorf("Could not start the transfer service. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
