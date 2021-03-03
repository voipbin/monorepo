package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmconference"
)

// ConferenceGet gets the conference.
// It returns conference info if it succeed.
func (h *serviceHandler) ConferenceGet(u *models.User, id uuid.UUID) (*models.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":       u.ID,
		"username":   u.Username,
		"conference": id,
	})
	log.Debugf("Get conference. conference: %s", id)

	// get conference
	res, err := h.reqHandler.CMConferenceGet(id)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}
	c := res.Convert()

	// check permission
	if u.Permission != models.UserPermissionAdmin && u.ID != c.UserID {
		log.Info("The user has no permission for this conference.")
		return nil, fmt.Errorf("user has no permission")
	}

	return c, nil
}

// ConferenceGets gets the list of conference.
// It returns list of calls if it succeed.
func (h *serviceHandler) ConferenceGets(u *models.User, size uint64, token string) ([]*models.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	log.Debugf("Get conferences. token: %s", token)
	// get calls
	ctx := context.Background()
	res, err := h.dbHandler.ConferenceGetsByUserID(ctx, u.ID, token, size)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return []*models.Conference{}, nil
	}

	return res, nil
}

// ConferenceCreate is a service handler for conference creating.
func (h *serviceHandler) ConferenceCreate(u *models.User, confType models.ConferenceType, name, detail string) (*models.Conference, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"user":     u.ID,
			"username": u.Username,
			"type":     confType,
			"name":     name,
			"detail":   detail,
		},
	)

	conf, err := h.reqHandler.CMConferenceCreate(u.ID, cmconference.Type(confType), name, detail)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	// create conference
	res := &models.Conference{
		ID:     conf.ID,
		Type:   models.ConferenceType(conf.Type),
		UserID: conf.UserID,

		Status: models.ConferenceStatus(conf.Status),
		Name:   conf.Name,
		Detail: conf.Detail,

		CallIDs: conf.CallIDs,

		TMCreate: conf.TMCreate,
		TMUpdate: conf.TMUpdate,
		TMDelete: conf.TMDelete,
	}

	return res, nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceDelete(u *models.User, confID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"user":       u.ID,
			"username":   u.Username,
			"conference": confID,
		},
	)

	// get conference
	cf, err := h.reqHandler.CMConferenceGet(confID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}

	// check onwer
	if cf.UserID != u.ID {
		log.Error("The user does not have permission to delete this conference.")
		return fmt.Errorf("%s", "not owned conference")
	}

	// destroy
	log.Debug("Destroying conference.")
	if err := h.reqHandler.CMConferenceDelete(confID); err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return err
	}

	return nil
}
