package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// conferenceGet vaildates the user's ownership and returns the conference info.
func (h *serviceHandler) conferenceGet(ctx context.Context, u *user.User, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "conferenceGet",
			"user_id":       u.ID,
			"conference_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.CFV1ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", tmp).Debug("Received result.")

	if u.Permission != user.PermissionAdmin && u.ID != tmp.UserID {
		log.Info("The user has no permission for this conference.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := conference.ConvertToConference(tmp)
	return res, nil
}

// ConferenceGet gets the conference.
// It returns conference info if it succeed.
func (h *serviceHandler) ConferenceGet(u *user.User, id uuid.UUID) (*conference.Conference, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"user":       u.ID,
		"username":   u.Username,
		"conference": id,
	})
	log.Debugf("Get conference. conference: %s", id)

	// get conference
	res, err := h.conferenceGet(ctx, u, id)
	if err != nil {
		log.Infof("Could not get conference info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ConferenceGets gets the list of conference.
// It returns list of calls if it succeed.
func (h *serviceHandler) ConferenceGets(u *user.User, size uint64, token string) ([]*conference.Conference, error) {
	ctx := context.Background()
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
	tmps, err := h.reqHandler.CFV1ConferenceGets(ctx, u.ID, token, size, "conference")
	if err != nil {
		log.Infof("Could not get conferences info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*conference.Conference{}
	for _, tmp := range tmps {
		c := conference.ConvertToConference(&tmp)
		res = append(res, c)
	}

	return res, nil
}

// ConferenceCreate is a service handler for conference creating.
func (h *serviceHandler) ConferenceCreate(
	u *user.User,
	confType conference.Type,
	name string,
	detail string,
	webhookURI string,
	preActions []action.Action,
	postActions []action.Action,
) (*conference.Conference, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"user":         u.ID,
			"username":     u.Username,
			"type":         confType,
			"name":         name,
			"detail":       detail,
			"webhook_uri":  webhookURI,
			"pre_actions":  preActions,
			"post_actions": postActions,
		},
	)
	log.Debugf("Creating a conference.")

	fmPreActions := []fmaction.Action{}
	for _, a := range preActions {
		fmPreActions = append(fmPreActions, *action.CreateAction(&a))
	}

	fmPostActions := []fmaction.Action{}
	for _, a := range postActions {
		fmPostActions = append(fmPostActions, *action.CreateAction(&a))
	}

	conf, err := h.reqHandler.CFV1ConferenceCreate(ctx, u.ID, cfconference.Type(confType), name, detail, 0, webhookURI, map[string]interface{}{}, fmPreActions, fmPostActions)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	res := conference.ConvertToConference(conf)
	return res, nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceDelete(u *user.User, confID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"user":       u.ID,
			"username":   u.Username,
			"conference": confID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, confID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}

	// destroy
	log.Debug("Destroying conference.")
	if err := h.reqHandler.CFV1ConferenceDelete(ctx, confID); err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return err
	}

	return nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceKick(u *user.User, confID uuid.UUID, callID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"user":          u.ID,
			"username":      u.Username,
			"conference_id": confID,
			"call_id":       callID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, confID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}

	// kick the call from the conference
	log.Debug("Kick")
	if err := h.reqHandler.CFV1ConferenceKick(ctx, confID, callID); err != nil {
		log.Errorf("Could not kick the call from the conference. err: %v", err)
		return err
	}

	return nil
}
