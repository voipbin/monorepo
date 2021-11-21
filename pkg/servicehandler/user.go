package servicehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

func (h *serviceHandler) UserCreate(username, password string, permission uint64) (*user.User, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"Username":   username,
		"Permission": permission,
	})
	log.Debug("Creating a new user.")

	tmp, err := h.reqHandler.UMV1UserCreate(ctx, username, password, permission)
	if err != nil {
		log.Errorf("Could not create a new user. err: %v", err)
		return nil, err
	}

	res := user.ConvertUser(tmp)
	return res, nil
}

// UserGet returns user info of given userID.
func (h *serviceHandler) UserGet(userID uint64) (*user.User, error) {
	ctx := context.Background()

	tmp, err := h.reqHandler.UMV1UserGet(ctx, userID)
	if err != nil {
		logrus.Errorf("Could not get user info. err: %v", err)
		return nil, err
	}

	res := user.ConvertUser(tmp)
	return res, nil
}

// UserGets returns list of all users
func (h *serviceHandler) UserGets(size uint64, token string) ([]*user.User, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":  "UserGets",
		"size":  size,
		"token": token,
	})
	log.Debug("Received request detail.")

	tmp, err := h.reqHandler.UMV1UserGets(ctx, token, size)
	if err != nil {
		logrus.Errorf("Could not get users info. err: %v", err)
		return nil, err
	}

	res := []*user.User{}

	for _, u := range tmp {
		t := user.ConvertUser(&u)
		res = append(res, t)
	}

	return res, nil
}
