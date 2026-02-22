package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/pkg/listenhandler/models/request"
)

// processV1CustomersSignupPost handles POST /v1/customers/signup request
func (h *listenHandler) processV1CustomersSignupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersSignupPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersSignupPost.")

	var reqData request.V1DataCustomersSignupPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.Signup(
		ctx,
		reqData.Name,
		reqData.Detail,
		reqData.Email,
		reqData.PhoneNumber,
		reqData.Address,
		reqData.WebhookMethod,
		reqData.WebhookURI,
		reqData.ClientIP,
	)
	if err != nil {
		log.Errorf("Could not signup customer. err: %v", err)
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

// processV1CustomersEmailVerifyPost handles POST /v1/customers/email_verify request
func (h *listenHandler) processV1CustomersEmailVerifyPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersEmailVerifyPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersEmailVerifyPost.")

	var reqData request.V1DataCustomersEmailVerifyPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.EmailVerify(ctx, reqData.Token)
	if err != nil {
		log.Errorf("Could not verify customer email. err: %v", err)
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
