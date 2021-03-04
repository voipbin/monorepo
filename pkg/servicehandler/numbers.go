package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/nmnumber"
)

// NumberGets sends a request to getting a list of numbers
// It sends a request to the number-manager to getting a list of numbers.
// it returns list of numbers if it succeed.
func (h *serviceHandler) NumberGets(u *models.User, size uint64, token string) ([]*models.Number, error) {
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
	res := []*models.Number{}
	for _, tmp := range tmps {
		c := tmp.ConvertNumber()
		res = append(res, c)
	}

	return res, nil
}

// NumberCreate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberCreate(u *models.User, num string) (*models.Number, error) {
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

	return tmp.ConvertNumber(), nil
}

// NumberGet handles number get request.
// It sends a request to the number-manager to get a existed number.
// it returns got number information if it succeed.
func (h *serviceHandler) NumberGet(u *models.User, id uuid.UUID) (*models.Number, error) {
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
	if u.HasPermission(models.UserPermissionAdmin) != true && res.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, res.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	if res.TMDelete != "" {
		log.Debugf("Deleted item.")
		return nil, fmt.Errorf("not found")
	}

	return res.ConvertNumber(), nil
}

// NumberDelete handles number delete request.
// It sends a request to the number-manager to delete a existed number.
// it returns deleted number information if it succeed.
func (h *serviceHandler) NumberDelete(u *models.User, id uuid.UUID) (*models.Number, error) {
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
	if u.HasPermission(models.UserPermissionAdmin) != true && tmp.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, tmp.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmp.TMDelete != "" {
		log.Debugf("Deleted item.")
		return nil, fmt.Errorf("not found")
	}

	// delete numbers
	res, err := h.reqHandler.NMNumberDelete(id)
	if err != nil {
		log.Infof("Could not delete numbers info. err: %v", err)
		return nil, err
	}

	return res.ConvertNumber(), nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdate(u *models.User, numb *models.Number) (*models.Number, error) {
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
	if u.HasPermission(models.UserPermissionAdmin) != true && tmpNumb.UserID != u.ID {
		log.Errorf("The user has no permission for this number. user: %d, number_user: %d", u.ID, tmpNumb.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	if tmpNumb.TMDelete != "" {
		log.Debugf("Deleted item.")
		return nil, fmt.Errorf("not found")
	}

	// update number
	nNumb := nmnumber.CreateNumber(numb)
	tmp, err := h.reqHandler.NMNumberUpdate(nNumb)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertNumber(), nil
}
