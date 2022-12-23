package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// TranscriptGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
func (h *serviceHandler) TranscriptGets(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGets",
		"customer_id":   u.ID,
		"username":      u.Username,
		"transcribe_id": transcribeID,
	})

	_, err := h.transcribeGet(ctx, u, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcripts from the transcribe-manager. err: %v", err)
		return nil, err
	}

	res := []*tmtranscript.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// TranscribeStart sends a request to transcribe-
