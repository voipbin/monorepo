package servicehandler

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// RecordingfileGet returns downloadable url for recording
func (h *serviceHandler) RecordingfileGet(u *user.User, id string) (string, error) {

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

	filename := fmt.Sprintf("%s.wav", id)
	log.Debugf("Getting recording file. recording: %s", filename)

	// get download url from storage-manager
	url, err := h.reqHandler.STRecordingGet(filename)
	if err != nil {
		log.Errorf("Could not get download url. err: %v", err)
		return "", err
	}

	return url, nil
}
