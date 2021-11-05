package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// NumberGets sends a request to getting a list of numbers
// It sends a request to the number-manager to getting a list of numbers.
// it returns list of numbers if it succeed.
func (h *serviceHandler) NumberGets(u *user.User, size uint64, token string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    "token",
	})

	// get available numbers
	tmps, err := h.reqHandler.NMNumberGets(u.ID, token, size)
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
func (h *serviceHandler) NumberCreate(u *user.User, num string) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"numbers":  num,
	})

	if num == "" {
		log.Errorf("Not acceptable number. num: %s", num)
		return nil, fmt.Errorf("not acceptable number. num: %s", num)
	}

	// create numbers
	tmp, err := h.reqHandler.NMNumberCreate(u.ID, num)
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
func (h *serviceHandler) NumberGet(u *user.User, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"number":   id,
	})

	// get number info
	res, err := h.reqHandler.NMNumberGet(id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(user.PermissionAdmin) && res.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, res.UserID)
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
func (h *serviceHandler) NumberDelete(u *user.User, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"number":   id,
	})

	// get number info
	tmp, err := h.reqHandler.NMNumberGet(id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(user.PermissionAdmin) && tmp.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, tmp.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmp.TMDelete != defaultTimestamp {
		log.WithField("number", tmp).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// delete numbers
	res, err := h.reqHandler.NMNumberDelete(id)
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
func (h *serviceHandler) NumberUpdate(u *user.User, numb *number.Number) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"number":   numb,
	})

	// get number
	tmpNumb, err := h.reqHandler.NMNumberGet(numb.ID)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	// permission check
	if !u.HasPermission(user.PermissionAdmin) && tmpNumb.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, tmpNumb.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmpNumb.TMDelete != defaultTimestamp {
		log.WithField("number", tmpNumb).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	// update number
	nNumb := number.CreateNumber(numb)
	tmp, err := h.reqHandler.NMNumberUpdate(nNumb)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := number.ConvertNumber(tmp)
	return res, nil
}
