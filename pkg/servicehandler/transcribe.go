package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// TranscribeCreate sends a request to transcribe-manager
// to generate a recording-transcribe
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeCreate(u *cscustomer.Customer, recordingID uuid.UUID, language string) (*tmtranscribe.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"reference_id": recordingID,
		"language":     language,
	})

	// get recording
	rec, err := h.reqHandler.CallV1RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not")
		return nil, err
	}

	// check the recording ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != rec.CustomerID {
		log.Error("The user has no permission for this recording.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create tanscribe
	tmp, err := h.reqHandler.TranscribeV1RecordingCreate(ctx, u.ID, recordingID, language)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
