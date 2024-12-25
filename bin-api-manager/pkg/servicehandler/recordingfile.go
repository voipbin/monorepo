package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingfileGet",
		"customer_id": a.CustomerID,
		"recording":   id,
	})

	// get r info from call-manager
	r, err := h.recordingGet(ctx, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return "", err
	}
	log.WithField("recording", r).Debugf("Found recording info. recording_id: %s", r.ID)

	if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return "", fmt.Errorf("agent has no permission")
	}

	// get download url from storage-manager
	log.Debugf("Getting recording file. recording: %s", id)
	referenceIDs := []uuid.UUID{
		id,
	}
	res, err := h.reqHandler.StorageV1CompressfileCreate(ctx, referenceIDs, []uuid.UUID{}, 300000)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return res.DownloadURI, nil
}
