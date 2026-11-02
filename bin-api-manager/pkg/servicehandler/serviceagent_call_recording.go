package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// serviceAgentRecordingDurationDefault is the recording duration (seconds) used when an agent
	// omits it. It is also the upper bound: call-manager treats 0 as unlimited, so the agent
	// path never forwards 0 and never accepts more than this.
	serviceAgentRecordingDurationDefault = 7200
)

// serviceAgentCallOwned returns the call when it belongs to the agent's customer and is owned
// by the agent. An empty owner never matches (fail-closed).
func (h *serviceHandler) serviceAgentCallOwned(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.Call, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	c, err := h.callGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	if c.CustomerID != a.CustomerID || a.AgentID() == uuid.Nil || c.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return c, nil
}

// ServiceAgentCallRecordingStart starts the recording of a call the agent owns.
//
// Unlike the admin endpoint it does not accept on_end_flow_id (uuid.Nil is always sent), it
// does not answer a ringing call (the call must already be progressing), and it limits the
// duration to 1..7200 seconds (omitted: 7200). A nil argument means the field was omitted.
// The platform does not announce the recording to the other party by design.
func (h *serviceHandler) ServiceAgentCallRecordingStart(
	ctx context.Context,
	a *auth.AuthIdentity,
	callID uuid.UUID,
	format *cmrecording.Format,
	endOfSilence *int,
	endOfKey *string,
	duration *int,
) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentCallRecordingStart",
		"call_id": callID,
	})
	if a != nil {
		log = log.WithField("agent_id", a.AgentID())
	}

	c, err := h.serviceAgentCallOwned(ctx, a, callID)
	if err != nil {
		log.Infof("The recording start is refused. err: %v", err)
		return nil, err
	}

	// call-manager answers a call that is not progressing before recording. Refuse here so that a
	// ringing leg is never answered just because a recording was requested.
	if c.Status != cmcall.StatusProgressing {
		return nil, fmt.Errorf("%w: the call is not progressing. status: %s", serviceerrors.ErrInvalidArgument, c.Status)
	}

	// best-effort check on the call read above. It is not atomic.
	if c.RecordingID != uuid.Nil {
		return nil, fmt.Errorf("%w: the call is already being recorded", serviceerrors.ErrStateInvalid)
	}

	f := cmrecording.FormatWAV
	if format != nil {
		f = *format
	}
	if f != cmrecording.FormatWAV {
		return nil, fmt.Errorf("%w: unsupported format: %s", serviceerrors.ErrInvalidArgument, f)
	}

	d := serviceAgentRecordingDurationDefault
	if duration != nil {
		d = *duration
	}
	if d < 1 || d > serviceAgentRecordingDurationDefault {
		return nil, fmt.Errorf("%w: duration must be between 1 and %d seconds", serviceerrors.ErrInvalidArgument, serviceAgentRecordingDurationDefault)
	}

	silence := 0
	if endOfSilence != nil {
		silence = *endOfSilence
	}
	if silence < 0 {
		return nil, fmt.Errorf("%w: end_of_silence must not be negative", serviceerrors.ErrInvalidArgument)
	}

	key := ""
	if endOfKey != nil {
		key = *endOfKey
	}

	// audit log: who did what. No party information is logged.
	log.Info("Audit: the agent starts the call recording.")

	tmp, err := h.reqHandler.CallV1CallRecordingStart(ctx, callID, f, silence, key, d, uuid.Nil)
	if err != nil {
		log.Errorf("Could not start the recording. err: %v", err)
		return nil, err
	}
	if tmp == nil {
		log.Errorf("The call-manager returned an empty call.")
		return nil, fmt.Errorf("%w: empty response from call-manager", serviceerrors.ErrInternal)
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ServiceAgentCallRecordingStop stops the recording of a call the agent owns, whoever started it.
func (h *serviceHandler) ServiceAgentCallRecordingStop(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentCallRecordingStop",
		"call_id": callID,
	})
	if a != nil {
		log = log.WithField("agent_id", a.AgentID())
	}

	c, err := h.serviceAgentCallOwned(ctx, a, callID)
	if err != nil {
		log.Infof("The recording stop is refused. err: %v", err)
		return nil, err
	}

	if c.RecordingID == uuid.Nil {
		return nil, fmt.Errorf("%w: the call has no ongoing recording", serviceerrors.ErrStateInvalid)
	}
	if c.Status != cmcall.StatusProgressing {
		return nil, fmt.Errorf("%w: the call is not progressing. status: %s", serviceerrors.ErrStateInvalid, c.Status)
	}

	// audit log: who did what. No party information is logged.
	log.Info("Audit: the agent stops the call recording.")

	tmp, err := h.reqHandler.CallV1CallRecordingStop(ctx, callID)
	if err != nil {
		log.Errorf("Could not stop the recording. err: %v", err)
		return nil, err
	}
	if tmp == nil {
		log.Errorf("The call-manager returned an empty call.")
		return nil, fmt.Errorf("%w: empty response from call-manager", serviceerrors.ErrInternal)
	}

	return tmp.ConvertWebhookMessage(), nil
}
