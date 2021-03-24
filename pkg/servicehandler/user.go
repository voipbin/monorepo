package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

func (h *serviceHandler) UserCreate(username, password string, permission uint64) (*user.User, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username":   username,
		"Permission": permission,
	})
	log.Debug("Creating a new user.")

	ctx := context.Background()
	tmp, err := h.dbHandler.UserGetByUsername(ctx, username)
	if tmp != nil {
		log.Info("User is already existing.")
		return nil, fmt.Errorf("user already exist")
	}

	// generate hash password
	hashPassword, err := generateHash(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	// create user
	u := &user.User{
		Username:     username,
		PasswordHash: hashPassword,
		Permission:   user.Permission(permission),
	}

	if err := h.dbHandler.UserCreate(ctx, u); err != nil {
		log.Errorf("Could not create a new user. err: %v", err)
		return nil, err
	}

	res, err := h.dbHandler.UserGetByUsername(ctx, username)
	if err != nil {
		log.Errorf("Could not get created user info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UserGet returns user info of given userID.
func (h *serviceHandler) UserGet(userID uint64) (*user.User, error) {
	ctx := context.Background()
	res, err := h.dbHandler.UserGet(ctx, userID)
	if err != nil {
		logrus.Errorf("Could not get user info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UserGets returns list of all users
func (h *serviceHandler) UserGets() ([]*user.User, error) {
	ctx := context.Background()
	res, err := h.dbHandler.UserGets(ctx)
	if err != nil {
		logrus.Errorf("Could not get users info. err: %v", err)
		return nil, err
	}

	return res, nil
}
