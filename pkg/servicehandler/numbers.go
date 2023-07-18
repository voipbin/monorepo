package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// NumberGets sends a request to getting a list of numbers
// It sends a request to the number-manager to getting a list of numbers.
// it returns list of numbers if it succeed.
func (h *serviceHandler) NumberGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       "token",
	})

	// get available numbers
	tmps, err := h.reqHandler.NumberV1NumberGets(ctx, u.ID, token, size)
	if err != nil {
		log.Infof("Could not get numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*nmnumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// NumberCreate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberCreate(ctx context.Context, u *cscustomer.Customer, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"numbers":     num,
	})

	if num == "" {
		log.Errorf("Not acceptable number. num: %s", num)
		return nil, fmt.Errorf("not acceptable number. num: %s", num)
	}

	// create numbers
	tmp, err := h.reqHandler.NumberV1NumberCreate(ctx, u.ID, num, callFlowID, messageFlowID, name, detail)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// numberGet validates the number's ownership and returns the number info.
func (h *serviceHandler) numberGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*nmnumber.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "numberGet",
		"customer_id": u.ID,
		"number_id":   id,
	})

	// get number info
	res, err := h.reqHandler.NumberV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, res.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if res.TMDelete != defaultTimestamp {
		log.WithField("number", res).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// NumberGet handles number get request.
// It sends a request to the number-manager to get a existed number.
// it returns got number information if it succeed.
func (h *serviceHandler) NumberGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      id,
	})

	// get number info
	tmp, err := h.numberGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// NumberDelete handles number delete request.
// It sends a request to the number-manager to delete a existed number.
// it returns deleted number information if it succeed.
func (h *serviceHandler) NumberDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      id,
	})

	// get number info
	_, err := h.numberGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// delete numbers
	tmp, err := h.reqHandler.NumberV1NumberDelete(ctx, id)
	if err != nil {
		log.Infof("Could not delete numbers info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"number_id":   id,
	})

	// get number
	_, err := h.numberGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// update number
	tmp, err := h.reqHandler.NumberV1NumberUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdateFlowIDs(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberUpdateFlowIDs",
		"customer_id": u.ID,
		"number_id":   id,
	})

	// get number
	_, err := h.numberGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// update number
	tmp, err := h.reqHandler.NumberV1NumberUpdateFlowID(ctx, id, callFlowID, messageFlowID)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberRenew handles number renew request.
// It sends a request to the number-manager to renew the numbers.
// it returns renewed numbers information if it succeed.
func (h *serviceHandler) NumberRenew(ctx context.Context, u *cscustomer.Customer, tmRenew string) ([]*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberRenew",
		"customer_id": u.ID,
		"tm_renew":    tmRenew,
	})

	// need a admin permission
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("User has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// get number
	tmps, err := h.reqHandler.NumberV1NumberRenewByTmRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	res := []*nmnumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}
