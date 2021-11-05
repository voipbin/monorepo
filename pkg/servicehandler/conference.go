package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// ConferenceGet gets the conference.
// It returns conference info if it succeed.
func (h *serviceHandler) ConferenceGet(u *user.User, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":       u.ID,
		"username":   u.Username,
		"conference": id,
	})
	log.Debugf("Get conference. conference: %s", id)

	// get conference
	res, err := h.reqHandler.CFConferenceGet(id)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}
	c := conference.Convert(res)

	// check permission
	if u.Permission != user.PermissionAdmin && u.ID != c.UserID {
		log.Info("The user has no permission for this conference.")
		return nil, fmt.Errorf("user has no permission")
	}

	return c, nil
}

// ConferenceGets gets the list of conference.
// It returns list of calls if it succeed.
func (h *serviceHandler) ConferenceGets(u *user.User, size uint64, token string) ([]*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get conferences
	tmps, err := h.reqHandler.CFConferenceGets(u.ID, token, size, "conference")
	if err != nil {
		log.Infof("Could not get conferences info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*conference.Conference{}
	for _, tmp := range tmps {
		c := conference.Convert(&tmp)
		res = append(res, c)
	}

	return res, nil
}

// ConferenceCreate is a service handler for conference creating.
func (h *serviceHandler) ConferenceCreate(u *user.User, confType conference.Type, name string, detail string, webhookURI string) (*conference.Conference, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"user":        u.ID,
			"username":    u.Username,
			"type":        confType,
			"name":        name,
			"detail":      detail,
			"webhook_uri": webhookURI,
		},
	)

	conf, err := h.reqHandler.CFConferenceCreate(u.ID, cfconference.Type(confType), name, detail, webhookURI)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	// create conference
	res := &conference.Conference{
		ID:     conf.ID,
		Type:   conference.Type(conf.Type),
		UserID: conf.UserID,

		Status: conference.Status(conf.Status),
		Name:   conf.Name,
		Detail: conf.Detail,

		CallIDs:      conf.CallIDs,
		RecordingIDs: conf.RecordingIDs,

		WebhookURI: conf.WebhookURI,

		TMCreate: conf.TMCreate,
		TMUpdate: conf.TMUpdate,
		TMDelete: conf.TMDelete,
	}

	return res, nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceDelete(u *user.User, confID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"user":       u.ID,
			"username":   u.Username,
			"conference": confID,
		},
	)

	// get conference
	cf, err := h.reqHandler.CFConferenceGet(confID)
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
	if err := h.reqHandler.CFConferenceDelete(confID); err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return err
	}

	return nil
}
