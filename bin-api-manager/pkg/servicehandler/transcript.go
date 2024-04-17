package servicehandler

import (
	"context"
	"fmt"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TranscriptGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
func (h *serviceHandler) TranscriptGets(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGets",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"transcribe_id": transcribeID,
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

	filters := map[string]string{
		"transcribe_id": transcribeID.String(),
		"deleted":       "false",
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, "", 100, filters)
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
