package listenhandler

import (
	"context"
	"encoding/json"
	"errors"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/listenhandler/models/request"
)

// processV1CustomersCompleteSignupPost handles POST /v1/customers/complete_signup request
func (h *listenHandler) processV1CustomersCompleteSignupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersCompleteSignupPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersCompleteSignupPost.")

	var reqData request.V1DataCustomersCompleteSignupPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.CompleteSignup(ctx, reqData.TempToken, reqData.Code)
	if err != nil {
		log.Errorf("Could not complete signup. err: %v", err)
		if errors.Is(err, customerhandler.ErrTooManyAttempts) {
			return simpleResponse(429), nil
		}
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
