package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// recordingGet validates the recording's ownership and returns the recording info.
func (h *serviceHandler) recordingGet(ctx context.Context, a *amagent.Agent, recordingID uuid.UUID) (*cmrecording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "recordingGet",
		"customer_id":   a.CustomerID,
		"transcribe_id": recordingID,
	})

	// send request
	res, err := h.reqHandler.CallV1RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return nil, err
	}
	log.WithField("recording", res).Debug("Received result.")

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted recording. recording_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// RecordingGet returns downloadable url for recording
func (h *serviceHandler) RecordingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGet",
		"customer_id": a.CustomerID,
		"recording":   id,
	})

	// get recording info from call-manager
	rec, err := h.recordingGet(ctx, a, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get recording info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, rec.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := rec.ConvertWebhookMessage()
	return res, nil
}

// RecordingGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) RecordingGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmp, err := h.reqHandler.CallV1RecordingGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := []*cmrecording.WebhookMessage{}
	for _, tmpRecord := range tmp {
		record := tmpRecord.ConvertWebhookMessage()
		res = append(res, record)
	}

	return res, nil
}

// RecordingDelete sends a request to call-manager
// to deleting a recording.
// it returns deleted recording info if it succeed.
func (h *serviceHandler) RecordingDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RecordingDelete",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"recording_id": id,
	})

	r, err := h.recordingGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}
	log.WithField("recording", r).Debugf("Validated recording info. recording_id: %s", r.ID)

	if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.CallV1RecordingDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the recording. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
