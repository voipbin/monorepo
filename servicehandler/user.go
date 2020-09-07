package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
)

func (h *servicHandler) UserCreate(username, password string) (*user.User, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username": username,
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
