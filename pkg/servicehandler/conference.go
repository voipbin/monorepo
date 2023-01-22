package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// conferenceGet vaildates the customer's ownership and returns the conference info.
func (h *serviceHandler) conferenceGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "conferenceGet",
			"customer_id":   u.ID,
			"conference_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The customer has no permission for this conference.")
		return nil, fmt.Errorf("customer has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceGet gets the conference.
// It returns conference info if it succeed.
func (h *serviceHandler) ConferenceGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConferenceGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"conference":  id,
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
func (h *serviceHandler) ConferenceGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConferenceGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get conferences
	tmps, err := h.reqHandler.ConferenceV1ConferenceGets(ctx, u.ID, token, size, "conference")
	if err != nil {
		log.Infof("Could not get conferences info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cfconference.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// ConferenceCreate is a service handler for conference creating.
func (h *serviceHandler) ConferenceCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	confType cfconference.Type,
	name string,
	detail string,
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "ConferenceCreate",
			"customer_id":  u.ID,
			"username":     u.Username,
			"type":         confType,
			"name":         name,
			"detail":       detail,
			"pre_actions":  preActions,
			"post_actions": postActions,
		},
	)
	log.Debugf("Creating a conference.")

	tmp, err := h.reqHandler.ConferenceV1ConferenceCreate(ctx, u.ID, confType, name, detail, 0, map[string]interface{}{}, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceDelete(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "ConferenceDelete",
			"customer_id": u.ID,
			"username":    u.Username,
			"conference":  confID,
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
	if err := h.reqHandler.ConferenceV1ConferenceDelete(ctx, confID); err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return err
	}

	return nil
}

// ConferenceUpdate is a service handler for conference updating.
func (h *serviceHandler) ConferenceUpdate(
	ctx context.Context,
	u *cscustomer.Customer,
	cfID uuid.UUID,
	name string,
	detail string,
	timeout int,
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "ConferenceUpdate",
			"customer_id": u.ID,
			"username":    u.Username,
			"conference":  cfID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, cfID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.ConferenceV1ConferenceUpdate(
		ctx,
		cfID,
		name,
		detail,
		timeout,
		preActions,
		postActions,
	)
	if err != nil {
		log.Errorf("Could not update the conference info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceRecordingStart is a service handler for conference recording start.
func (h *serviceHandler) ConferenceRecordingStart(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "ConferenceRecordingStart",
			"customer_id":   u.ID,
			"username":      u.Username,
			"conference_id": confID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, confID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	// recording
	tmp, err := h.reqHandler.ConferenceV1ConferenceRecordingStart(ctx, confID)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceRecordingStop is a service handler for conference recording stop.
func (h *serviceHandler) ConferenceRecordingStop(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "ConferenceRecordingStop",
			"customer_id":   u.ID,
			"username":      u.Username,
			"conference_id": confID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, confID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	// recording
	tmp, err := h.reqHandler.ConferenceV1ConferenceRecordingStop(ctx, confID)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceTranscribeStart is a service handler for conference transcribe start.
func (h *serviceHandler) ConferenceTranscribeStart(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "ConferenceTranscribeStart",
			"customer_id":   u.ID,
			"username":      u.Username,
			"conference_id": conferenceID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.ConferenceV1ConferenceTranscribeStart(ctx, conferenceID, language)
	if err != nil {
		log.Errorf("Could not start the conference transcribe. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceTranscribeStop is a service handler for conference transcribe stop.
func (h *serviceHandler) ConferenceTranscribeStop(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "ConferenceTranscribeStop",
			"customer_id":   u.ID,
			"username":      u.Username,
			"conference_id": conferenceID,
		},
	)

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	// recording
	tmp, err := h.reqHandler.ConferenceV1ConferenceTranscribeStop(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not stop the conference transcribe. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
