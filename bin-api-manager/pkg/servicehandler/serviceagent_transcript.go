package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentTranscriptList sends a request to transcribe-manager
// to get a list of transcript lines for one transcribe session, scoped to
// the service agent's own customer.
// Ownership is authorized against the target transcribe's own CustomerID
// (fetched by transcribeID), not a re-derived parent resource — this is a
// pure Get/Read path, not a Create/Start path validating a caller-supplied
// reference before the row exists. Mirrors TranscriptList's own existing
// shape (fetch, then hasPermission on the fetched struct's field) — only
// the permission bitmask changes (PermissionAll instead of Admin|Manager).
func (h *serviceHandler) ServiceAgentTranscriptList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "ServiceAgentTranscriptList",
		"customer_id":   a.CustomerID,
		"username":      a.DisplayName(),
		"transcribe_id": transcribeID,
		"size":          size,
		"token":         token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"transcribe_id": transcribeID.String(),
		"deleted":       "false",
	}
	typedFilters, err := h.convertTranscriptFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return nil, err
	}

	res := []*tmtranscript.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}
