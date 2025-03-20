package servicehandler

import (
	"context"
	"fmt"
	"net/http"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmrecording "monorepo/bin-call-manager/models/recording"

	cfconference "monorepo/bin-conference-manager/models/conference"

	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// conferenceGet vaildates the customer's ownership and returns the conference info.
func (h *serviceHandler) conferenceGet(ctx context.Context, id uuid.UUID) (*cfconference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "conferenceGet",
		"conference_id": id,
	})

	// send request
	res, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", res).Debug("Received result.")

	return res, nil
}

// ConferenceGet gets the conference.
// It returns conference info if it succeed.
func (h *serviceHandler) ConferenceGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConferenceGet",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"conference_id": id,
	})
	log.Debugf("Get conference. conference: %s", id)

	// get conference
	tmp, err := h.conferenceGet(ctx, id)
	if err != nil {
		log.Infof("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceGets gets the list of conference.
// It returns list of calls if it succeed.
func (h *serviceHandler) ConferenceGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConferenceGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
		"type":        string(cfconference.TypeConference),
	}

	// get conferences
	tmps, err := h.reqHandler.ConferenceV1ConferenceGets(ctx, token, size, filters)
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
	a *amagent.Agent,
	confType cfconference.Type,
	name string,
	detail string,
	timeout int,
	data map[string]interface{},
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ConferenceCreate",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"type":         confType,
		"name":         name,
		"detail":       detail,
		"timeout":      timeout,
		"data":         data,
		"pre_actions":  preActions,
		"post_actions": postActions,
	})
	log.Debugf("Creating a conference.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ConferenceV1ConferenceCreate(ctx, a.CustomerID, confType, name, detail, timeout, data, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceDelete is a service handler for conference creating.
func (h *serviceHandler) ConferenceDelete(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConferenceDelete",
		"agent":         a,
		"conference_id": conferenceID,
	})
	log.Debug("Destroying conference.")

	// get conference for ownership check
	c, err := h.conferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// destroy
	tmp, err := h.reqHandler.ConferenceV1ConferenceDelete(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceUpdate is a service handler for conference updating.
func (h *serviceHandler) ConferenceUpdate(
	ctx context.Context,
	a *amagent.Agent,
	cfID uuid.UUID,
	name string,
	detail string,
	timeout int,
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConferenceUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"conference":  cfID,
	})

	// get conference for ownership check
	c, err := h.conferenceGet(ctx, cfID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ConferenceRecordingStart(
	ctx context.Context,
	a *amagent.Agent,
	conferenceID uuid.UUID,
	format cmrecording.Format,
	duration int,
	onEndFlowID uuid.UUID,
) (*cfconference.WebhookMessage, error) {
	// get conference for ownership check
	c, err := h.conferenceGet(ctx, conferenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get conference info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	// recording
	tmp, err := h.reqHandler.ConferenceV1ConferenceRecordingStart(ctx, conferenceID, format, duration, onEndFlowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the conference recording")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceRecordingStop is a service handler for conference recording stop.
func (h *serviceHandler) ConferenceRecordingStop(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error) {

	// get conference for ownership check
	c, err := h.conferenceGet(ctx, confID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get conference info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	// recording
	tmp, err := h.reqHandler.ConferenceV1ConferenceRecordingStop(ctx, confID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the conference recording")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferenceTranscribeStart is a service handler for conference transcribe start.
func (h *serviceHandler) ConferenceTranscribeStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConferenceTranscribeStart",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"conference_id": conferenceID,
	})

	// get conference for ownership check
	c, err := h.conferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ConferenceTranscribeStop(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConferenceTranscribeStop",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"conference_id": conferenceID,
	})

	// get conference for ownership check
	c, err := h.conferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
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

// ConferenceMediaStreamStart starts a media streaming of the conference
// it returns error if it failed.
func (h *serviceHandler) ConferenceMediaStreamStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConferenceMediaStreamStart",
		"agent":         a,
		"conference_id": conferenceID,
		"encapsulation": encapsulation,
	})

	c, err := h.conferenceGet(ctx, conferenceID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	if errRun := h.websockHandler.RunMediaStream(ctx, w, r, cmexternalmedia.ReferenceTypeConfbridge, c.ConfbridgeID, encapsulation); errRun != nil {
		log.Errorf("Could not run the meida stream handler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
