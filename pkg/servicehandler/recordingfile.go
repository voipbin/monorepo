package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(u *user.User, id uuid.UUID) (string, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"user":      u,
			"recording": id,
		},
	)

	// get recording info from call-manager
	recording, err := h.reqHandler.CMRecordingGet(id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return "", err
	}

	// check the recording ownership
	if u.HasPermission(user.PermissionAdmin) != true && u.ID != recording.UserID {
		log.Error("The user has no permission for this recording.")
		return "", fmt.Errorf("user has no permission")
	}

	// get download url from storage-manager
	log.Debugf("Getting recording file. recording: %s", recording.Filename)
	url, err := h.reqHandler.SMRecordingGet(recording.Filename)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return url, nil
}
