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

// transcribeGet validates the transcribe's ownership and returns the transcribe info.
func (h *serviceHandler) transcribeGet(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "transcribeGet",
			"customer_id":   u.ID,
			"transcribe_id": transcribeID,
		},
	)

	// send request
	res, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get the transcribe info. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// TranscribeGet sends a request to transcribe-manager
// to getting the transcribe.
func (h *serviceHandler) TranscribeGet(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGet",
		"customer_id":   u.ID,
		"username":      u.Username,
		"transcribe_id": transcribeID,
	})

	tmp, err := h.transcribeGet(ctx, u, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TranscribeGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
func (h *serviceHandler) TranscribeGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TranscribeGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	tmps, err := h.reqHandler.TranscribeV1TranscribeGets(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return nil, err
	}

	res := []*tmtranscribe.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// TranscribeStart sends a request to transcribe-manager
// to start a transcribe.
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeStart(ctx context.Context, u *cscustomer.Customer, referenceType tmtranscribe.ReferenceType, referenceID uuid.UUID, language string, direction tmtranscribe.Direction) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
	})

	// check the ownership
	var err error
	switch referenceType {
	case tmtranscribe.ReferenceTypeCall:
		_, err = h.callGet(ctx, u, referenceID)

	case tmtranscribe.ReferenceTypeRecording:
		_, err = h.recordingGet(ctx, u, referenceID)

	default:
		err = fmt.Errorf("unsupported reference type")
	}
	if err != nil {
		log.Errorf("Could not pass the reference validation. err: %v", err)
		return nil, fmt.Errorf("could not pass the reference validation. err: %v", err)
	}

	tmp, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, u.ID, referenceType, referenceID, language, direction)
	if err != nil {
		log.Errorf("Could not start the transcribe. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TranscribeStop sends a request to transcribe-manager
// to stop a transcribe.
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeStop(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":   u.ID,
		"transcribe_id": transcribeID,
	})

	// check the transcribe info
	_, err := h.transcribeGet(ctx, u, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TranscribeDelete sends a request to tramscribe-manager
// to delete the transcribe.
// it returns transcribe info if it succeed.
func (h *serviceHandler) TranscribeDelete(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     transcribeID,
	})

	_, err := h.transcribeGet(ctx, u, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.TranscribeV1TranscribeDelete(ctx, transcribeID)
	if err != nil {
		// no call info found
		log.Infof("Could not delete the transcribe info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}
