package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// TranscribeCreate sends a request to transcribe-manager
// to generate a recording-transcribe
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeCreate(u *user.User, recordingID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":         u,
		"reference_id": recordingID,
		"language":     language,
	})

	// get recording
	rec, err := h.reqHandler.CMRecordingGet(recordingID)
	if err != nil {
		log.Errorf("Could not")
		return nil, err
	}

	// check the recording ownership
	if u.HasPermission(user.PermissionAdmin) != true && u.ID != rec.UserID {
		log.Error("The user has no permission for this recording.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create tanscribe
	tmp, err := h.reqHandler.TMRecordingPost(recordingID, language)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := &transcribe.Transcribe{
		ID:            tmp.ID,
		Type:          transcribe.Type(tmp.Type),
		ReferenceID:   tmp.ReferenceID,
		Language:      tmp.Language,
		WebhookURI:    tmp.WebhookURI,
		WebhookMethod: tmp.WebhookMethod,
	}

	// transcripts
	for _, t := range tmp.Transcripts {
		tmp := transcribe.Transcript{
			Direction: transcribe.TranscriptDirection(t.Direction),
			Message:   t.Message,
			TMCreate:  t.TMCreate,
		}

		res.Transcripts = append(res.Transcripts, tmp)
	}

	return res, nil
}
