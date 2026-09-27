package aicallhandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
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

// insightSessionStart returns the current Insight session boundary recorded on
// the AIcall's Metadata (VOIP-1484), in UTC.
//
// Absent returns false, which every caller treats as "no boundary" -- the
// pre-VOIP-1484 behaviour, and the correct reading for a freshly created AIcall
// whose rows are all newer than it is. An unparsable value is a corrupted
// write, never something to guess at: it is logged and treated as absent, so a
// bad string degrades to full replay rather than dropping the entire history.
func insightSessionStart(c *aicall.AIcall) (time.Time, bool) {
	if c == nil || c.Metadata == nil {
		return time.Time{}, false
	}

	raw, ok := c.Metadata[aicall.MetaKeyInsightSessionStart].(string)
	if !ok || raw == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":      "insightSessionStart",
			"aicall_id": c.ID,
		}).Warnf("Could not parse the insight session start, treating it as absent. value: %s, err: %v", raw, err)
		return time.Time{}, false
	}

	return parsed.UTC(), true
}

// cutBeforeSessionStart drops the message rows that belong to an EARLIER
// Insight session, and is the one place that rule is defined for both history
// builders (getPipecatcallMessages and buildListenTurnMessages).
//
// The cut is STRICT: a row is dropped only when its TMCreate is strictly before
// the boundary. The boundary row itself -- the first system row the refresh
// wrote, whose TMCreate IS the boundary -- must survive, otherwise the new
// session opens without the platform's own Insight guardrails. A nil TMCreate
// is kept for the same reason: it carries no evidence of being old.
//
// No boundary means no cut, so this is a no-op for every non-Insight AIcall.
func cutBeforeSessionStart(rows []*message.Message, c *aicall.AIcall) []*message.Message {
	boundary, ok := insightSessionStart(c)
	if !ok {
		return rows
	}

	res := make([]*message.Message, 0, len(rows))
	for _, m := range rows {
		if m.TMCreate != nil && m.TMCreate.UTC().Before(boundary) {
			continue
		}
		res = append(res, m)
	}

	return res
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

// resolveActiveAIForMcp resolves the AI whose MCP whitelist governs this AIcall.
//
// It exists because the MCP paths need the CURRENT team member's whitelist, while
// resolveAI resolves the START member and resolveActiveAIIDFromAIcall has no
// fallback at all (it returns uuid.Nil and only warns). Neither is usable here:
// using the start member calls the wrong member's MCP servers, and propagating a
// uuid.Nil would make the MCP paths fail for a team whose current member is
// momentarily unresolvable -- strictly worse than today.
//
// The fallback is therefore unconditional: on ANY failure (member absent from the
// team, member's AI unfetchable, team unfetchable) it degrades to the START
// member's AI, which is the session's defined state whenever CurrentMemberID
// cannot be resolved (the send path persists exactly that repair). It returns nil
// only when the start member is unusable too; callers must then keep their
// existing behaviour rather than failing the call.
//
// Deliberately separate from resolveActiveAIIDFromAIcall: that helper has six
// other callers driving message attribution, plus a twin in messagehandler, and
// giving it a fallback would change all of them.
func (h *aicallHandler) resolveActiveAIForMcp(ctx context.Context, c *aicall.AIcall) *ai.AI {
	log := logrus.WithFields(logrus.Fields{
		"func":      "resolveActiveAIForMcp",
		"aicall_id": c.ID,
	})

	switch c.AssistanceType {
	case aicall.AssistanceTypeAI:
		a, err := h.aiHandler.Get(ctx, c.AssistanceID)
		if err != nil {
			log.Warnf("Could not get the ai. ai_id: %s, err: %v", c.AssistanceID, err)
			return nil
		}
		return a

	case aicall.AssistanceTypeTeam:
		// handled below

	default:
		// Mirror resolveAI's default arm: an unrecognised assistance type must not
		// be treated as an AI id. Returning nil keeps the MCP gates fail-closed
		// instead of authorising against whatever row AssistanceID happens to hit.
		log.Warnf("Unsupported assistance type for mcp resolution. assistance_type: %s", c.AssistanceType)
		return nil
	}

	t, err := h.teamHandler.Get(ctx, c.AssistanceID)
	if err != nil {
		// no team, no start member to fall back to.
		log.Warnf("Could not get the team. team_id: %s, err: %v", c.AssistanceID, err)
		return nil
	}

	// resolveTeamMemberAI falls back to the start member only when the requested
	// member is absent; it errors out when the member is present but its AI is
	// unfetchable. Retry explicitly for the start member to cover that mode too.
	a, resolvedMemberID, err := h.resolveTeamMemberAI(ctx, t, c.CurrentMemberID)
	if err == nil {
		log.Debugf("Resolved the team member AI for mcp. member_id: %s, ai_id: %s", resolvedMemberID, a.ID)
		return a
	}
	log.Warnf("Could not resolve the current team member AI, falling back to the start member. member_id: %s, err: %v", c.CurrentMemberID, err)

	// resolveTeamMemberAI's own fallback loop already tried the start member whenever
	// CurrentMemberID was absent from the roster, so retrying then would re-issue the
	// byte-identical failing fetch. Retry ONLY for the mode its fallback does not
	// cover: the current member IS on the roster but its own AI fetch failed.
	if !teamHasMember(t, c.CurrentMemberID) || c.CurrentMemberID == t.StartMemberID {
		return nil
	}

	a, resolvedMemberID, err = h.resolveTeamMemberAI(ctx, t, t.StartMemberID)
	if err != nil {
		log.Warnf("Could not resolve the start member AI either. start_member_id: %s, err: %v", t.StartMemberID, err)
		return nil
	}
	log.Debugf("Resolved the start member AI as fallback. member_id: %s", resolvedMemberID)

	return a
}

// teamHasMember reports whether the given member id is on the team's roster.
func teamHasMember(t *team.Team, memberID uuid.UUID) bool {
	for _, m := range t.Members {
		if m.ID == memberID {
			return true
		}
	}
	return false
}
