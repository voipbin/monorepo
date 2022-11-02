package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": u.ID,
			"recording":   id,
		},
	)

	// get recording info from call-manager
	recording, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return "", err
	}

	// check the recording ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != recording.CustomerID {
		log.Error("The user has no permission for this recording.")
		return "", fmt.Errorf("user has no permission")
	}

	// get download url from storage-manager
	log.Debugf("Getting recording file. recording: %s", id)
	res, err := h.reqHandler.StorageV1RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return res.DownloadURI, nil
}
