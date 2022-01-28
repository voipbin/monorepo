package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
)

// NumberGets sends a request to getting a list of numbers
// It sends a request to the number-manager to getting a list of numbers.
// it returns list of numbers if it succeed.
func (h *serviceHandler) NumberGets(u *cscustomer.Customer, size uint64, token string) ([]*number.Number, error) {
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
	res := []*number.Number{}
	for _, tmp := range tmps {
		c := number.ConvertNumber(&tmp)
		res = append(res, c)
	}

	return res, nil
}

// NumberCreate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberCreate(u *cscustomer.Customer, num string) (*number.Number, error) {
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
	tmp, err := h.reqHandler.NMV1NumberCreate(ctx, u.ID, num)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	res := number.ConvertNumber(tmp)
	return res, nil
}

// NumberGet handles number get request.
// It sends a request to the number-manager to get a existed number.
// it returns got number information if it succeed.
func (h *serviceHandler) NumberGet(u *cscustomer.Customer, id uuid.UUID) (*number.Number, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      id,
	})

	// get number info
	res, err := h.reqHandler.NMV1NumberGet(ctx, id)
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

	tmp := number.ConvertNumber(res)

	return tmp, nil
}

// NumberDelete handles number delete request.
// It sends a request to the number-manager to delete a existed number.
// it returns deleted number information if it succeed.
func (h *serviceHandler) NumberDelete(u *cscustomer.Customer, id uuid.UUID) (*number.Number, error) {
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

	// delete numbers
	res, err := h.reqHandler.NMV1NumberDelete(ctx, id)
	if err != nil {
		log.Infof("Could not delete numbers info. err: %v", err)
		return nil, err
	}

	tmpNum := number.ConvertNumber(res)
	return tmpNum, nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdate(u *cscustomer.Customer, numb *number.Number) (*number.Number, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"number":      numb,
	})

	// get number
	tmpNumb, err := h.reqHandler.NMV1NumberGet(ctx, numb.ID)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpNumb.CustomerID {
		log.Errorf("The user has no permission for this number. user: %s, number_user: %s", u.ID, tmpNumb.CustomerID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmpNumb.TMDelete != defaultTimestamp {
		log.WithField("number", tmpNumb).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// update number
	nNumb := number.CreateNumber(numb)
	tmp, err := h.reqHandler.NMV1NumberUpdate(ctx, nNumb)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := number.ConvertNumber(tmp)
	return res, nil
}
