package servicehandler

import (
	"context"
	"fmt"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TranscriptGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
func (h *serviceHandler) TranscriptList(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGets",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"transcribe_id": transcribeID,
	})

	t, err := h.transcribeGet(ctx, transcribeID)
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

	// Convert string filters to typed filters
	typedFilters, err := h.convertTranscriptFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 100, typedFilters)
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

// convertTranscriptFilters converts map[string]string to map[tmtranscript.Field]any
func (h *serviceHandler) convertTranscriptFilters(filters map[string]string) (map[tmtranscript.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, tmtranscript.Transcript{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[tmtranscript.Field]any, len(typed))
	for k, v := range typed {
		result[tmtranscript.Field(k)] = v
	}

	return result, nil
}
