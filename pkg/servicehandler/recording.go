package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// RecordingGet returns downloadable url for recording
func (h *serviceHandler) RecordingGet(u *user.User, id uuid.UUID) (*recording.Recording, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"user":      u,
			"recording": id,
		},
	)

	// get recording info from call-manager
	rec, err := h.reqHandler.CMRecordingGet(id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// check the recording ownership
	if !u.HasPermission(user.PermissionAdmin) && u.ID != rec.UserID {
		log.Error("The user has no permission for this recording.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := recording.Convert(rec)

	return res, nil
}

// RecordingGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) RecordingGets(u *user.User, size uint64, token string) ([]*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	tmp, err := h.reqHandler.CMRecordingGets(u.ID, size, token)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := []*recording.Recording{}
	for _, tmpRecord := range tmp {
		record := recording.Convert(&tmpRecord)
		res = append(res, record)
	}

	return res, nil
}
