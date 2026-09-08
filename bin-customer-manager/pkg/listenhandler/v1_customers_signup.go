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

	// Signup creates both a customer and an accesskey; log both created resources'
	// IDs, consistent with other creation endpoints. Neither Customer.WebhookSecret
	// nor the freshly-issued Accesskey.RawToken (present in tmp) is logged.
	resultLog := log.WithFields(logrus.Fields{"customer_id": tmp.Customer.ID, "accesskey_id": tmp.Accesskey.ID})

	data, err := json.Marshal(tmp)
	if err != nil {
		resultLog.Debugf("Could not marshal the result data. err: %v", err)
		return simpleResponse(500), nil
	}
	resultLog.Debug("Sending result.")

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
		log.WithField("customer_id", tmp.Customer.ID).Debugf("Could not marshal the result data. err: %v", err)
		return simpleResponse(500), nil
	}
	log.WithField("customer_id", tmp.Customer.ID).Debug("Sending result.")

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersEmailVerifyResendPost handles POST /v1/customers/email_verify_resend request
func (h *listenHandler) processV1CustomersEmailVerifyResendPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersEmailVerifyResendPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersEmailVerifyResendPost.")

	var reqData request.V1DataCustomersEmailVerifyResendPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// EmailVerifyResend swallows every non-fault outcome, so a 200 here says
	// "request accepted", never "that address exists".
	if err := h.customerHandler.EmailVerifyResend(ctx, reqData.Email); err != nil {
		log.Errorf("Could not resend the verification email. err: %v", err)
		return simpleResponse(500), nil
	}

	return simpleResponse(200), nil
}
