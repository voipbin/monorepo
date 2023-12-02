package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// transcribeGet validates the transcribe's ownership and returns the transcribe info.
func (h *serviceHandler) transcribeGet(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "transcribeGet",
		"customer_id":   a.CustomerID,
		"transcribe_id": transcribeID,
	})

	// send request
	res, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get the transcribe info. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debug("Received result.")

	return res, nil
}

// TranscribeGet sends a request to transcribe-manager
// to getting the transcribe.
func (h *serviceHandler) TranscribeGet(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGet",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"transcribe_id": transcribeID,
	})

	tmp, err := h.transcribeGet(ctx, a, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TranscribeGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
func (h *serviceHandler) TranscribeGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TranscribeGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmps, err := h.reqHandler.TranscribeV1TranscribeGets(ctx, a.CustomerID, token, size)
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
func (h *serviceHandler) TranscribeStart(ctx context.Context, a *amagent.Agent, referenceType tmtranscribe.ReferenceType, referenceID uuid.UUID, language string, direction tmtranscribe.Direction) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "TranscribeStart",
		"customer_id":    a.CustomerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// check the ownership
	var err error
	var customerID uuid.UUID
	switch referenceType {
	case tmtranscribe.ReferenceTypeCall:
		tmpResource, tmpErr := h.callGet(ctx, a, referenceID)
		if tmpErr != nil {
			err = tmpErr
			break
		} else {
			customerID = tmpResource.CustomerID
		}

	case tmtranscribe.ReferenceTypeRecording:
		tmpResource, tmpErr := h.recordingGet(ctx, a, referenceID)
		if tmpErr != nil {
			err = tmpErr
			break
		} else {
			customerID = tmpResource.CustomerID
		}

	default:
		err = fmt.Errorf("unsupported reference type")
	}
	if err != nil {
		log.Errorf("Could not pass the reference validation. err: %v", err)
		return nil, fmt.Errorf("could not pass the reference validation. err: %v", err)
	}

	if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, a.CustomerID, referenceType, referenceID, language, direction)
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
func (h *serviceHandler) TranscribeStop(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeStop",
		"customer_id":   a.CustomerID,
		"transcribe_id": transcribeID,
	})

	// check the transcribe info
	t, err := h.transcribeGet(ctx, a, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) TranscribeDelete(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TranscribeDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     transcribeID,
	})

	t, err := h.transcribeGet(ctx, a, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
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
