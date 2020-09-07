package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmconference"
)

// ConferenceCreate is a service handler for conference creating.
func (h *servicHandler) ConferenceCreate(u *user.User, confType conference.Type, name, detail string) (*conference.Conference, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"user":     u.ID,
			"username": u.Username,
			"type":     confType,
			"name":     name,
			"detail":   detail,
		},
	)

	conf, err := h.reqHandler.CallConferenceCreate(u.ID, cmconference.Type(confType), name, detail)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	// create conference
	res := &conference.Conference{
		ID:   conf.ID,
		Type: conference.Type(conf.Type),

		Status: conference.Status(conf.Status),
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
func (h *servicHandler) ConferenceDelete(u *user.User, confID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"user":       u.ID,
			"username":   u.Username,
			"conference": confID,
		},
	)

	// get conference
	cf, err := h.reqHandler.CallConferenceGet(confID)
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
	if err := h.reqHandler.CallConferenceDelete(confID); err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return err
	}

	return nil
}
