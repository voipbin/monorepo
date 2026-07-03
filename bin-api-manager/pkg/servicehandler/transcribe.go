package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// transcribeGet validates the transcribe's ownership and returns the transcribe info.
func (h *serviceHandler) transcribeGet(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	res, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, transcribeID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TranscribeGet sends a request to transcribe-manager
// to getting the transcribe.
func (h *serviceHandler) TranscribeGet(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeGet",
		"customer_id":   a.CustomerID,
		"username":      a.DisplayName(),
		"transcribe_id": transcribeID,
	})

	tmp, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TranscribeGets sends a request to transcribe-manager
// to getting a list of transcribes.
// it returns list of transcribe info if it succeed.
// If referenceType/referenceID are both provided, results are additionally
// filtered to transcribes originating from that specific resource (e.g. a
// call). Note a single reference can have multiple transcribes (different
// languages, or multiple start/stop sessions), so this can return more than
// one item even when scoped to a single reference.
// The caller (server/transcribes.go) is expected to reject a partial pair
// (only one of referenceType/referenceID non-zero) before calling this;
// this function does not itself validate pairing.
func (h *serviceHandler) TranscribeList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, referenceType string, referenceID uuid.UUID) ([]*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "TranscribeGets",
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

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
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

	// Convert string filters to typed filters
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

// TranscribeStart sends a request to transcribe-manager
// to start a transcribe.
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeStart(
	ctx context.Context,
	a *auth.AuthIdentity,
	referenceType string,
	referenceID uuid.UUID,
	language string,
	direction tmtranscribe.Direction,
	onEndFlowID uuid.UUID,
	provider tmtranscribe.Provider,
) (*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "TranscribeStart",
		"customer_id":    a.CustomerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
		"on_end_flow_id": onEndFlowID,
		"provider":       provider,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// get transcribe resource info
	tmpReferenceType, tmpReferenceID, err := h.transcribeGetResourceInfo(ctx, a, referenceType, referenceID)
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

// transcribeGetResourceInfo returns corresponding transcribe resource info of the given reference.
// returns error if the reference is not transcrib-able or has no perrmission
func (h *serviceHandler) transcribeGetResourceInfo(ctx context.Context, a *auth.AuthIdentity, referenceType string, referenceID uuid.UUID) (tmtranscribe.ReferenceType, uuid.UUID, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "transcribeGetResourceInfo",
		"agent":          a,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	var err error
	var tmpCustomerID uuid.UUID
	var resReferenceType tmtranscribe.ReferenceType
	var resReferenceID uuid.UUID

	// get reference resource
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
		err = fmt.Errorf("%w: unsupported reference type", serviceerrors.ErrInvalidArgument)
	}
	if err != nil {
		log.Errorf("Could not pass the reference validation. err: %v", err)
		return "", uuid.Nil, fmt.Errorf("could not pass the reference validation: %w", err)
	}

	// check the ownership
	if !h.hasPermission(ctx, a, tmpCustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return "", uuid.Nil, serviceerrors.ErrPermissionDenied
	}

	return resReferenceType, resReferenceID, nil

}

// TranscribeStop sends a request to transcribe-manager
// to stop a transcribe.
// it returns transcribe if it succeed.
func (h *serviceHandler) TranscribeStop(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeStop",
		"customer_id":   a.CustomerID,
		"transcribe_id": transcribeID,
	})

	// check the transcribe info
	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) TranscribeDelete(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "TranscribeDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"call_id":     transcribeID,
	})

	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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

// convertTranscribeFilters converts map[string]string to map[tmtranscribe.Field]any
func (h *serviceHandler) convertTranscribeFilters(filters map[string]string) (map[tmtranscribe.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, tmtranscribe.Transcribe{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[tmtranscribe.Field]any, len(typed))
	for k, v := range typed {
		result[tmtranscribe.Field(k)] = v
	}

	return result, nil
}
