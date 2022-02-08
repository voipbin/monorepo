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
func (h *serviceHandler) NumberGets(u *cscustomer.Customer, size uint64, token string) ([]*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       "token",
	})

	// get available numbers
	tmps, err := h.reqHandler.NMV1NumberGets(ctx, u.ID, token, size)
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
func (h *serviceHandler) NumberCreate(u *cscustomer.Customer, num string, flowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
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
	tmp, err := h.reqHandler.NMV1NumberCreate(ctx, u.ID, num, flowID, name, detail)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberGet handles number get request.
// It sends a request to the number-manager to get a existed number.
// it returns got number information if it succeed.
func (h *serviceHandler) NumberGet(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      id,
	})

	// get number info
	tmp, err := h.reqHandler.NMV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, tmp.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmp.TMDelete != defaultTimestamp {
		log.WithField("number", tmp).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// NumberDelete handles number delete request.
// It sends a request to the number-manager to delete a existed number.
// it returns deleted number information if it succeed.
func (h *serviceHandler) NumberDelete(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      id,
	})

	// get number info
	num, err := h.reqHandler.NMV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != num.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, num.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if num.TMDelete != defaultTimestamp {
		log.WithField("number", num).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// delete numbers
	tmp, err := h.reqHandler.NMV1NumberDelete(ctx, id)
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
func (h *serviceHandler) NumberUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"number_id":   id,
	})

	// get number
	num, err := h.reqHandler.NMV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != num.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, num.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if num.TMDelete != defaultTimestamp {
		log.WithField("number", num).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// update number
	tmp, err := h.reqHandler.NMV1NumberUpdateBasicInfo(ctx, id, name, detail)
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
func (h *serviceHandler) NumberUpdateFlowID(u *cscustomer.Customer, id, flowID uuid.UUID) (*nmnumber.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"number_id":   id,
	})

	// get number
	num, err := h.reqHandler.NMV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != num.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, num.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if num.TMDelete != defaultTimestamp {
		log.WithField("number", num).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// update number
	tmp, err := h.reqHandler.NMV1NumberUpdateFlowID(ctx, id, flowID)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
