package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TranscriptList sends a request to transcribe-manager to get a page of
// transcript lines for one transcribe session (newest first, `size` rows
// older than `token`).
// The admin-surface counterpart of ServiceAgentTranscriptList: same shape,
// admin/manager permission on the fetched transcribe's customer.
func (h *serviceHandler) TranscriptList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscriptList",
		"customer_id":   a.CustomerID,
		"username":      a.DisplayName(),
		"transcribe_id": transcribeID,
		"size":          size,
		"token":         token,
	})

	// An empty token means "from now", as TranscribeList and
	// ServiceAgentTranscriptList do. Set here (not left to
	// transcribe-manager) so the token this handler sends is fully
	// determined by its inputs (VOIP-1480).
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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

	tmps, err := h.reqHandler.TranscribeV1TranscriptList(ctx, token, size, typedFilters)
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
