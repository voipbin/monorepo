package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	umuser "gitlab.com/voipbin/bin-manager/user-manager.git/models/user"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// userGet validates the user's ownership and returns the userf info.
func (h *serviceHandler) userGet(ctx context.Context, u *user.User, userID uint64) (*user.User, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "tagGet",
			"user_id": u.ID,
			"tag_id":  userID,
		},
	)

	if u.Permission != user.PermissionAdmin && u.ID != userID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.UMV1UserGet(ctx, userID)
	if err != nil {
		log.Errorf("Could not get the user. err: %v", err)
		return nil, err
	}
	log.WithField("user", tmp).Debug("Received result.")

	// create result
	res := user.ConvertToUser(tmp)
	return res, nil
}

func (h *serviceHandler) UserCreate(u *user.User, username, password, name, detail string, permission user.Permission) (*user.User, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"Username":   username,
		"Permission": permission,
	})
	log.Debug("Creating a new user.")

	// check permission
	// only admin permssion can create a new user.
	if !u.HasPermission(user.PermissionAdmin) {
		log.Info("The user has no permission")
		return nil, fmt.Errorf("has no permission")
	}

	p := umuser.Permission(permission)
	tmp, err := h.reqHandler.UMV1UserCreate(ctx, username, password, name, detail, p)
	if err != nil {
		log.Errorf("Could not create a new user. err: %v", err)
		return nil, err
	}

	res := user.ConvertToUser(tmp)
	return res, nil
}

// UserGet returns user info of given userID.
func (h *serviceHandler) UserGet(u *user.User, userID uint64) (*user.User, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func":    "UserGet",
			"user_id": u.ID,
		},
	)

	res, err := h.userGet(ctx, u, userID)
	if err != nil {
		log.Errorf("Could not validate the user info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UserGets returns list of all users
func (h *serviceHandler) UserGets(u *user.User, size uint64, token string) ([]*user.User, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":  "UserGets",
		"size":  size,
		"token": token,
	})
	log.Debug("Received request detail.")

	if u.Permission != user.PermissionAdmin {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.UMV1UserGets(ctx, token, size)
	if err != nil {
		log.Errorf("Could not get users info. err: %v", err)
		return nil, err
	}

	res := []*user.User{}
	for _, u := range tmp {
		t := user.ConvertToUser(&u)
		res = append(res, t)
	}

	return res, nil
}

// UserUpdate sends a request to user-manager
// to update the user's basic info.
func (h *serviceHandler) UserUpdate(u *user.User, id uint64, name, detail string) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "UserUpdate",
		"user_id":  u.ID,
		"username": u.Username,
	})

	_, err := h.userGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the user info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.UMV1UserUpdateBasicInfo(ctx, id, name, detail); err != nil {
		log.Infof("Could not update the user's basic info. err: %v", err)
		return err
	}

	return nil
}

// UserUpdatePassword sends a request to user-manager
// to update the user's password.
func (h *serviceHandler) UserUpdatePassword(u *user.User, id uint64, password string) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "UserUpdatePassword",
		"user_id":  u.ID,
		"username": u.Username,
	})

	_, err := h.userGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the user info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.UMV1UserUpdatePassword(ctx, id, password); err != nil {
		log.Infof("Could not update the user's password. err: %v", err)
		return err
	}

	return nil
}

// UserUpdatePermission sends a request to user-manager
// to update the user's permission.
func (h *serviceHandler) UserUpdatePermission(u *user.User, id uint64, permission user.Permission) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "UserUpdatePassword",
		"user_id":  u.ID,
		"username": u.Username,
	})

	if u.Permission != user.PermissionAdmin {
		log.Info("The user has no permission.")
		return fmt.Errorf("user has no permission")
	}

	// send request
	if err := h.reqHandler.UMV1UserUpdatePermission(ctx, id, umuser.Permission(permission)); err != nil {
		log.Infof("Could not update the user's permission. err: %v", err)
		return err
	}

	return nil
}
