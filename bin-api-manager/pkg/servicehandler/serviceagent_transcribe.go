package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentTranscribeList sends a request to transcribe-manager
// to get a list of transcribes for the service agent's customer.
// it returns list of transcribe info if it succeed.
// If referenceType/referenceID are both provided, results are additionally
// filtered to transcribes originating from that specific resource (e.g. a
// call). Note a single reference can have multiple transcribes (different
// languages, or multiple start/stop sessions), so this can return more than
// one item even when scoped to a single reference.
// The caller (server/service_agents_transcribes.go) is expected to reject a
// partial pair (only one of referenceType/referenceID non-zero) before
// calling this; this function does not itself validate pairing.
func (h *serviceHandler) ServiceAgentTranscribeList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, referenceType string, referenceID uuid.UUID) ([]*tmtranscribe.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceAgentTranscribeList",
		"customer_id":    a.CustomerID,
		"username":       a.DisplayName(),
		"size":           size,
		"token":          token,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}
	if referenceType != "" {
		filters["reference_type"] = referenceType
	}
	if referenceID != uuid.Nil {
		filters["reference_id"] = referenceID.String()
	}

	typedFilters, err := h.convertTranscribeFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscribeList(ctx, token, size, typedFilters)
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

// ServiceAgentTranscribeStart sends a request to transcribe-manager
// to start a transcribe for the service agent's customer.
// it returns transcribe if it succeed.
func (h *serviceHandler) ServiceAgentTranscribeStart(
	ctx context.Context,
	a *auth.AuthIdentity,
	referenceType string,
	referenceID uuid.UUID,
	language string,
	direction tmtranscribe.Direction,
	onEndFlowID uuid.UUID,
	provider tmtranscribe.Provider,
) (*tmtranscribe.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceAgentTranscribeStart",
		"customer_id":    a.CustomerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
		"on_end_flow_id": onEndFlowID,
		"provider":       provider,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// get transcribe resource info. this validates ownership of the
	// reference resource against the agent's own customer.
	tmpReferenceType, tmpReferenceID, err := h.transcribeGetResourceInfoForAgent(ctx, a, referenceType, referenceID)
	if err != nil {
		log.Errorf("Could not get transcribe resource info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		a.CustomerID,
		uuid.Nil,
		onEndFlowID,
		tmpReferenceType,
		tmpReferenceID,
		language,
		direction,
		provider,
		60000,
	)
	if err != nil {
		log.Errorf("Could not start the transcribe. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// transcribeGetResourceInfoForAgent returns corresponding transcribe resource
// info of the given reference, scoped to Agent-level permission (customer
// tenant boundary only, no Admin/Manager requirement). This mirrors
// transcribeGetResourceInfo (used by the Admin/Manager-gated TranscribeStart)
// but checks amagent.PermissionAll instead of
// PermissionCustomerAdmin|PermissionCustomerManager, matching the
// service_agents/* API surface's Agent-level access model.
// Returns error if the reference is not transcribe-able or the agent has no
// permission on the resolved resource's customer.
func (h *serviceHandler) transcribeGetResourceInfoForAgent(ctx context.Context, a *auth.AuthIdentity, referenceType string, referenceID uuid.UUID) (tmtranscribe.ReferenceType, uuid.UUID, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "transcribeGetResourceInfoForAgent",
		"agent":          a,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	var err error
	var tmpCustomerID uuid.UUID
	var resReferenceType tmtranscribe.ReferenceType
	var resReferenceID uuid.UUID

	switch referenceType {
	case "call":
		tmpResource, tmpErr := h.callGet(ctx, referenceID)
		if tmpErr != nil {
			err = tmpErr
			break
		}
		tmpCustomerID = tmpResource.CustomerID
		resReferenceType = tmtranscribe.ReferenceTypeCall
		resReferenceID = tmpResource.ID

	case "conference":
		tmpResource, tmpErr := h.conferenceGet(ctx, referenceID)
		if tmpErr != nil {
			err = tmpErr
		}
		tmpCustomerID = tmpResource.CustomerID
		resReferenceType = tmtranscribe.ReferenceTypeConfbridge
		resReferenceID = tmpResource.ConfbridgeID

	case "recording":
		tmpResource, tmpErr := h.recordingGet(ctx, referenceID)
		if tmpErr != nil {
			err = tmpErr
		}
		tmpCustomerID = tmpResource.CustomerID
		resReferenceType = tmtranscribe.ReferenceTypeRecording
		resReferenceID = tmpResource.ID

	default:
		err = serviceerrors.ErrInvalidArgument
	}
	if err != nil {
		log.Errorf("Could not pass the reference validation. err: %v", err)
		return "", uuid.Nil, err
	}

	if !h.hasPermission(ctx, a, tmpCustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return "", uuid.Nil, serviceerrors.ErrPermissionDenied
	}

	return resReferenceType, resReferenceID, nil
}
