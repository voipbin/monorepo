package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(u *user.User, id uuid.UUID) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"user":      u,
			"recording": id,
		},
	)

	// get recording info from call-manager
	recording, err := h.reqHandler.CMV1RecordingGet(ctx, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return "", err
	}

	// check the recording ownership
	if !u.HasPermission(user.PermissionAdmin) && u.ID != recording.UserID {
		log.Error("The user has no permission for this recording.")
		return "", fmt.Errorf("user has no permission")
	}

	// get download url from storage-manager
	log.Debugf("Getting recording file. recording: %s", id)
	res, err := h.reqHandler.SMV1RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return res.DownloadURI, nil
}
