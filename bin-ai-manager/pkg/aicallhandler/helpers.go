package aicallhandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
)

// resolveActiveAIIDFromAIcall returns the active AI UUID for the given AIcall.
// For AssistanceTypeAI it returns ac.AssistanceID.
// For AssistanceTypeTeam it walks the team members to find CurrentMemberID's AIID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *aicallHandler) resolveActiveAIIDFromAIcall(ctx context.Context, ac *aicall.AIcall) uuid.UUID {
	switch ac.AssistanceType {
	case aicall.AssistanceTypeAI:
		return ac.AssistanceID
	case aicall.AssistanceTypeTeam:
		t, err := h.teamHandler.Get(ctx, ac.AssistanceID)
		if err != nil {
			logrus.Warnf("resolveActiveAIIDFromAIcall: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
			return uuid.Nil
		}
		for _, m := range t.Members {
			if m.ID == ac.CurrentMemberID {
				return m.AIID
			}
		}
		logrus.Warnf("resolveActiveAIIDFromAIcall: CurrentMemberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, ac.CurrentMemberID)
		return uuid.Nil
	default:
		logrus.Warnf("resolveActiveAIIDFromAIcall: unknown AssistanceType. type: %s", ac.AssistanceType)
		return uuid.Nil
	}
}

// isAIcallIdleExpired returns true if the AIcall has been idle longer than
// the configured conversation idle timeout. Returns false when c is nil or
// TMUpdate is nil (treated as freshly created).
func (h *aicallHandler) isAIcallIdleExpired(c *aicall.AIcall) bool {
	if c == nil || c.TMUpdate == nil {
		return false
	}
	threshold := time.Duration(config.Get().AIcallConversationIdleTimeoutHours) * time.Hour
	return time.Since(*c.TMUpdate) > threshold
}

// isAIcallReusable returns true if the AIcall is suitable to be reused for
// the next inbound message in the same conversation: it must exist, be in a
// non-terminal status, and not be idle-expired.
func (h *aicallHandler) isAIcallReusable(c *aicall.AIcall) bool {
	if c == nil {
		return false
	}
	if c.Status == aicall.StatusTerminated || c.Status == aicall.StatusTerminating {
		return false
	}
	if h.isAIcallIdleExpired(c) {
		return false
	}
	return true
}

// interruptPreviousPipecatcall attempts a synchronous, ping-gated termination
// of the previous pipecat session. Best-effort: errors are logged at DEBUG and
// swallowed. Correctness is provided by the response guard at delivery time
// (in messagehandler EventPMMessageBotLLM, added by a later slice).
//
// The Get call is bounded by a 1.5s context to avoid blocking the user-facing
// path on a degraded shared queue. The ping is bounded by 1.1s (inside
// pingPipecatHost). Total worst case: ~4.1s (1.5s Get + 1.1s ping + 1.5s terminate).
func (h *aicallHandler) interruptPreviousPipecatcall(ctx context.Context, pcID uuid.UUID) {
	if pcID == uuid.Nil {
		return
	}
	log := logrus.WithFields(logrus.Fields{
		"func":           "interruptPreviousPipecatcall",
		"pipecatcall_id": pcID,
	})

	gctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	pc, errGet := h.reqHandler.PipecatV1PipecatcallGet(gctx, pcID)
	if errGet != nil {
		log.Debugf("Could not get previous pipecatcall — assuming gone. err: %v", errGet)
		promAIcallInterruptAttemptedTotal.WithLabelValues("gone").Inc()
		return
	}
	if !h.pingPipecatHost(ctx, pc.HostID) {
		log.Debugf("Previous pipecatcall pod unreachable — skipping terminate. host_id: %s", pc.HostID)
		promAIcallInterruptAttemptedTotal.WithLabelValues("dead").Inc()
		return
	}
	tctx, tcancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer tcancel()
	if _, errTerm := h.reqHandler.PipecatV1PipecatcallTerminate(tctx, pc.HostID, pc.ID); errTerm != nil {
		log.Debugf("Previous pipecatcall terminate failed — response guard will handle. err: %v", errTerm)
		promAIcallInterruptAttemptedTotal.WithLabelValues("error").Inc()
		return
	}
	promAIcallInterruptAttemptedTotal.WithLabelValues("alive").Inc()
}
