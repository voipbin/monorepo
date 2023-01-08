package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "RecordingfileGet",
			"customer_id": u.ID,
			"recording":   id,
		},
	)

	// get r info from call-manager
	r, err := h.recordingGet(ctx, u, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return "", err
	}
	log.WithField("recording", r).Debugf("Found recording info. recording_id: %s", r.ID)

	// get download url from storage-manager
	log.Debugf("Getting recording file. recording: %s", id)
	res, err := h.reqHandler.StorageV1RecordingGet(ctx, id, 300000)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return res.DownloadURI, nil
}
