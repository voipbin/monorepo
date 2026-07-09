package aicallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// resolveTeamMemberAI looks up a team member by memberID, falling back to
// the team's StartMemberID when the requested member is not found.
// Returns the member's AI config and the matched member ID.
func (h *aicallHandler) resolveTeamMemberAI(ctx context.Context, t *team.Team, memberID uuid.UUID) (*ai.AI, uuid.UUID, error) {
	// try to find the requested member first
	for _, m := range t.Members {
		if m.ID == memberID {
			a, err := h.aiHandler.Get(ctx, m.AIID)
			if err != nil {
				return nil, uuid.Nil, errors.Wrapf(err, "could not get ai info for team member. ai_id: %s, member_id: %s", m.AIID, m.ID)
			}
			return a, m.ID, nil
		}
	}

	// fallback: find the start member
	for _, m := range t.Members {
		if m.ID == t.StartMemberID {
			a, err := h.aiHandler.Get(ctx, m.AIID)
			if err != nil {
				return nil, uuid.Nil, errors.Wrapf(err, "could not get ai info for team start member. ai_id: %s, member_id: %s", m.AIID, m.ID)
			}
			return a, m.ID, nil
		}
	}

	return nil, uuid.Nil, fmt.Errorf("could not find member or start member in team. team_id: %s, member_id: %s, start_member_id: %s", t.ID, memberID, t.StartMemberID)
}

// resolveAI resolves the AI config based on the assistance type.
// For AssistanceTypeAI, it fetches the AI directly.
// For AssistanceTypeTeam, it fetches the team, finds the start member, and fetches that member's AI config.
// Returns the AI config, the team parameter (nil for non-team types), and any error.
func (h *aicallHandler) resolveAI(ctx context.Context, assistanceType aicall.AssistanceType, assistanceID uuid.UUID) (*ai.AI, map[string]any, uuid.UUID, error) {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		c, err := h.aiHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, uuid.Nil, errors.Wrapf(err, "could not get ai info. ai_id: %s", assistanceID)
		}
		return c, nil, uuid.Nil, nil

	case aicall.AssistanceTypeTeam:
		t, err := h.teamHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, uuid.Nil, errors.Wrapf(err, "could not get team info. team_id: %s", assistanceID)
		}

		a, memberID, err := h.resolveTeamMemberAI(ctx, t, t.StartMemberID)
		if err != nil {
			return nil, nil, uuid.Nil, err
		}
		return a, t.Parameter, memberID, nil

	default:
		return nil, nil, uuid.Nil, fmt.Errorf("unsupported assistance type: %s", assistanceType)
	}
}

// resolveAIForTeam fetches all team members' AI configs, keyed by member UUID.
// Partial-failure: if individual member AI fetches fail, logs a warning and returns the partial map.
// Only a teamHandler.Get failure is fatal.
func (h *aicallHandler) resolveAIForTeam(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]*ai.AI, error) {
	t, err := h.teamHandler.Get(ctx, teamID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get team for resolveAIForTeam. team_id: %s", teamID)
	}

	type memberResult struct {
		memberID uuid.UUID
		ai       *ai.AI
		err      error
	}

	// Decouple from the caller's deadline: member AI fetches are best-effort for
	// snapshot observability and must not all fail when the outer RPC times out.
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer fetchCancel()

	ch := make(chan memberResult, len(t.Members))
	for _, m := range t.Members {
		go func(m team.Member) {
			a, errGet := h.aiHandler.Get(fetchCtx, m.AIID)
			ch <- memberResult{memberID: m.ID, ai: a, err: errGet}
		}(m)
	}

	res := make(map[uuid.UUID]*ai.AI, len(t.Members))
	for range t.Members {
		r := <-ch
		if r.err != nil {
			logrus.WithField("func", "resolveAIForTeam").
				Warnf("Could not get AI for team member — skipping. member_id: %s, err: %v", r.memberID, r.err)
			continue
		}
		res[r.memberID] = r.ai
	}

	return res, nil
}

// buildPromptSnapshots constructs the []PromptSnapshot to store in AIcall.Metadata at call start.
// For AssistanceTypeAI: one snapshot for the single AI config.
// For AssistanceTypeTeam: one snapshot per team member (partial-failure-tolerant via resolveAIForTeam).
// The returned bool is the auto-audit flag: true if any participating AI has AutoAICallAuditEnabled set.
func (h *aicallHandler) buildPromptSnapshots(ctx context.Context, a *ai.AI, assistanceType aicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID) ([]aicall.PromptSnapshot, bool) {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		substituted := h.getInitPrompt(ctx, a, activeflowID)
		return []aicall.PromptSnapshot{
			{
				AIID:            a.ID,
				PromptHistoryID: a.CurrentPromptHistoryID,
				Prompt:          substituted,
			},
		}, a.AutoAICallAuditEnabled

	case aicall.AssistanceTypeTeam:
		memberAIs, err := h.resolveAIForTeam(ctx, assistanceID)
		if err != nil {
			logrus.WithField("func", "buildPromptSnapshots").
				Errorf("Could not resolve team AIs — storing empty snapshots. err: %v", err)
			return []aicall.PromptSnapshot{}, false
		}
		snapshots := make([]aicall.PromptSnapshot, 0, len(memberAIs))
		autoAudit := false
		for memberID, memberAI := range memberAIs {
			if memberAI.AutoAICallAuditEnabled {
				autoAudit = true
			}
			substituted := h.getInitPrompt(ctx, memberAI, activeflowID)
			snapshots = append(snapshots, aicall.PromptSnapshot{
				AIID:            memberAI.ID,
				PromptHistoryID: memberAI.CurrentPromptHistoryID,
				Prompt:          substituted,
				MemberID:        memberID,
			})
		}
		return snapshots, autoAudit

	default:
		return []aicall.PromptSnapshot{}, false
	}
}

func (h *aicallHandler) Start(
	ctx context.Context,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
) (*aicall.AIcall, error) {

	// resolve AI config based on assistance type
	c, teamParameter, currentMemberID, err := h.resolveAI(ctx, assistanceType, assistanceID)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve ai config")
	}

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, c, assistanceType, assistanceID, activeflowID, referenceID, teamParameter, currentMemberID)

	case aicall.ReferenceTypeConversation:
		return h.startReferenceTypeConversation(ctx, c, assistanceType, assistanceID, activeflowID, referenceID, teamParameter, currentMemberID)

	case aicall.ReferenceTypeContactCase:
		return h.startReferenceTypeContactCase(ctx, c, assistanceType, assistanceID, activeflowID, referenceID, teamParameter, currentMemberID)

	case aicall.ReferenceTypeNone:
		return h.startReferenceTypeNone(ctx, c, assistanceType, assistanceID, activeflowID, teamParameter, currentMemberID)

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// startReferenceTypeCall starts a new aicall with reference type call
func (h *aicallHandler) startReferenceTypeCall(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeCall",
		"ai":            a,
		"activeflow_id": activeflowID,
	})
	log.Debugf("Starting a new aicall")

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, activeflowID, cmconfbridge.ReferenceTypeAI, a.ID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// start ai call
	res, err := h.startAIcallByRealtime(ctx, a, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, false, teamParameter, currentMemberID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	// start pipecatcall
	tmpPipecatcall, err := h.startPipecatcall(ctx, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmpPipecatcall).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	return res, nil
}

// startReferenceTypeConversation starts a new aicall with reference type conversation
func (h *aicallHandler) startReferenceTypeConversation(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConversation",
		"ai":            a,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// get conversation message
	vars, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the activeflow variables. activeflow_id: %s", activeflowID)
	}

	messageText, ok := vars.Variables["voipbin.conversation_message.text"]
	if !ok {
		return nil, errors.New("could not get the conversation message text from the activeflow variables")
	}

	// get existing aicall info — decide reuse or create fresh
	res, err := h.GetByReferenceID(ctx, referenceID)
	reusable := err == nil && h.isAIcallReusable(res)

	if !reusable {
		// mark idle-expired AIcalls as Terminated for hygiene before recreating
		if err == nil && res.Status != aicall.StatusTerminated && res.Status != aicall.StatusTerminating && h.isAIcallIdleExpired(res) {
			log.Infof("Existing AIcall idle-expired — terminating and starting fresh. aicall_id: %s", res.ID)
			promAIcallIdleExpiredTotal.Inc()
			if _, errEnd := h.UpdateStatus(ctx, res.ID, aicall.StatusTerminated); errEnd != nil {
				log.Warnf("Could not terminate idle AIcall: %v", errEnd)
			}
		}
		res, err = h.startAIcallByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeConversation, referenceID, false, teamParameter, currentMemberID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
		}
		if updated, errUpdate := h.UpdateStatus(ctx, res.ID, aicall.StatusProgressing); errUpdate != nil {
			log.Warnf("Could not update status to Progressing — continuing anyway (status field is observability only). aicall_id: %s, err: %v", res.ID, errUpdate)
		} else {
			res = updated
		}
	} else {
		// reuse: interrupt previous pipecat session (best-effort), then atomically
		// update both PipecatcallID and ActiveflowID so concurrent readers cannot
		// observe a half-applied state.
		log.WithFields(logrus.Fields{
			"aicall_id":          res.ID,
			"old_pipecatcall_id": res.PipecatcallID,
			"new_activeflow_id":  activeflowID,
		}).Debugf("Reusing existing conversation AIcall.")
		h.interruptPreviousPipecatcall(ctx, res.PipecatcallID)
		newPipecatcallID := h.utilHandler.UUIDCreate()
		tmp, errUpdate := h.UpdatePipecatcallIDAndActiveflowID(ctx, res.ID, newPipecatcallID, activeflowID)
		if errUpdate != nil {
			return nil, errors.Wrapf(errUpdate, "could not update pipecatcall_id+activeflow_id for existing aicall. aicall_id: %s", res.ID)
		}
		res = tmp

		// For team-typed AIcalls, refresh the in-memory AIEngineModel so the new
		// pipecat session uses the current member's engine. Falls back to StartMemberID
		// if CurrentMemberID is stale (e.g., team config changed). Symmetric with the
		// Send path (see resolveTeamMemberForSend invocation in send.go).
		if res.AssistanceType == aicall.AssistanceTypeTeam {
			if errResolve := h.resolveTeamMemberForSend(ctx, res); errResolve != nil {
				log.Warnf("Could not resolve team member AI on reuse — using snapshot. err: %v", errResolve)
			}
		}
	}
	log.WithField("aicall", res).Debugf("AIcall ready. aicall_id: %s", res.ID)

	// note: after create a new aicall, we need to create a new message for the conversation message
	// TODO: for AssistanceTypeTeam this calls teamHandler.Get via resolveTeamMemberForSend above,
	// and resolveActiveAIIDFromAIcall below calls it again. Same fix needed as send.go: refactor
	// resolveTeamMemberForSend to accept an optionally pre-fetched *team.Team.
	convUserActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, res)
	tmp, err := h.messageHandler.Create(ctx, uuid.Nil, res.CustomerID, res.ID, res.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
		messagehandler.WithActiveAIID(convUserActiveAIID))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", res.ID)
	}
	log.WithField("message", tmp).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)

	// NOTE: Tool whitelist for conversation-typed AIcalls is deferred to v2 — see
	// docs/plans/2026-04-27-conversation-ai-talk-design.md §13 and the Slice 0
	// decision in 2026-04-27-conversation-ai-talk-plan.md. The LLM may invoke
	// connect_call / stop_media / stop_flow in a chat context; each fails at
	// execute time and is observable via ai_manager_aicall_tool_execute_total.
	pc, err := h.startPipecatcall(ctx, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	if errTerminate := h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultAITaskTimeout); errTerminate != nil {
		log.Errorf("Could not send the pipecatcall terminate request correctly. err: %v", errTerminate)
	}

	return res, nil
}

// maxContactCaseCreateRetries bounds the create/reconcile loop in
// startReferenceTypeContactCase. A bound is required because a pathological
// sequence of concurrent creators+terminators could in principle keep colliding
// forever; 3 is generous relative to the spike-measured contention (20 concurrent
// INSERTs -> 1 success/19 duplicate-key 1062/0 deadlocks, see VOIP-1234 design).
const maxContactCaseCreateRetries = 3

// startReferenceTypeContactCase starts (or reuses) an aicall for reference type
// contact_case (VOIP-1234). Concurrency guard: the DB enforces "at most one active
// AIcall per (customer_id, reference_type, reference_id)" via the
// active_reference_key generated-column UNIQUE index (bin-dbscheme-manager
// a5a40c93d3e6). This function relies on that DB-level constraint with an
// optimistic-INSERT + reconcile-on-1062 pattern rather than a pre-check SELECT or
// in-process lock — a real spike (20 concurrent INSERTs against MySQL 8.0) proved
// the optimistic path alone gives full consistency for this workload (1
// success/19 duplicate-key 1062/0 deadlocks), unlike the heavier
// SELECT...FOR UPDATE + retry pattern bin-contact-manager uses for contact_cases.
//
// Flow:
//  1. Attempt to create a new AIcall.
//  2. On success, return it.
//  3. On a duplicate-key (1062) conflict, re-fetch the existing AIcall by
//     reference_id.
//     - If it is still active (not Terminated/Terminating), reuse it.
//     - If it has since terminated, its active_reference_key is NULL (no longer
//     occupying the unique slot). Before retrying the create, a rate-limit
//     guard is checked (see below) — if it trips, an error is returned
//     immediately instead of retrying.
//  4. Any other error propagates immediately.
//
// Recreate rate limit (VOIP-1234 design doc §2-1):
// Unlike Call/Conversation references, a Contact Case has no natural upper
// bound on lifetime — there is no "call ended" or "conversation closed" event
// that stops new questions from arriving. An agent can repeatedly reopen a
// closed Case and ask questions; each question after the existing AIcall has
// idle-expired (or otherwise become Terminated/Terminating) would otherwise
// trigger an unbounded sequence of AIcall recreations, and therefore unbounded
// LLM spend. To bound this, once an existing AIcall for the reference_id is
// observed to be Terminated/Terminating, recreation is blocked for
// config.Get().AIcallContactCaseRecreateRateLimitMinutes minutes measured from
// that AIcall's end time (TMEnd, falling back to TMUpdate when TMEnd is unset).
// If neither timestamp is available there is no basis to compute an age, so
// the guard fails open and the retry proceeds.
func (h *aicallHandler) startReferenceTypeContactCase(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeContactCase",
		"ai":            a,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	var lastErr error
	for attempt := 0; attempt < maxContactCaseCreateRetries; attempt++ {
		res, err := h.startAIcallByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeContactCase, referenceID, false, teamParameter, currentMemberID)
		if err == nil {
			log.WithField("aicall", res).Debugf("Created aicall for contact_case. aicall_id: %s", res.ID)
			return res, nil
		}

		if !dbhandler.IsErrDuplicate(err) {
			return nil, errors.Wrapf(err, "could not create aicall for contact_case. reference_id: %s", referenceID)
		}

		// duplicate key: another creator won the race for this
		// (customer_id, reference_type, reference_id) tuple. Re-fetch and decide
		// whether to reuse it or retry the create.
		existing, errGet := h.GetByReferenceID(ctx, referenceID)
		if errGet != nil {
			return nil, errors.Wrapf(errGet, "could not get existing aicall after duplicate key conflict. reference_id: %s", referenceID)
		}

		if existing.Status == aicall.StatusTerminated || existing.Status == aicall.StatusTerminating {
			// The row that won the race has since terminated (or was already
			// terminating). Its active_reference_key computes to NULL, so it no
			// longer occupies the unique slot — but before retrying the create,
			// check the recreate rate limit (VOIP-1234 design doc §2-1).
			if blocked, blockedErr := h.checkContactCaseRecreateRateLimit(referenceID, existing); blocked {
				return nil, blockedErr
			}

			log.Infof("Existing AIcall for contact_case is terminated/terminating — retrying create. aicall_id: %s, attempt: %d", existing.ID, attempt+1)
			lastErr = err
			continue
		}

		log.WithField("aicall", existing).Debugf("Reusing existing active aicall for contact_case. aicall_id: %s", existing.ID)
		return existing, nil
	}

	return nil, errors.Wrapf(lastErr, "could not create aicall for contact_case after %d retries. reference_id: %s", maxContactCaseCreateRetries, referenceID)
}

// checkContactCaseRecreateRateLimit implements the VOIP-1234 §2-1 recreate rate
// limit for contact_case AIcalls. It returns (true, err) when recreation must be
// blocked because the given terminated/terminating AIcall ended within the
// configured rate-limit window; the returned err is ready to be surfaced to the
// caller as-is. It returns (false, nil) when the retry may proceed — either
// because the window has elapsed, or because neither TMEnd nor TMUpdate is set
// on the existing AIcall (fail-open: no timestamp means no basis to block).
func (h *aicallHandler) checkContactCaseRecreateRateLimit(referenceID uuid.UUID, existing *aicall.AIcall) (bool, error) {
	terminatedAt := existing.TMEnd
	if terminatedAt == nil {
		terminatedAt = existing.TMUpdate
	}
	if terminatedAt == nil {
		// No timestamp to reason about — fail open and allow the retry.
		return false, nil
	}

	limitMinutes := config.Get().AIcallContactCaseRecreateRateLimitMinutes
	elapsed := time.Since(*terminatedAt)
	limitDuration := time.Duration(limitMinutes) * time.Minute
	if elapsed >= limitDuration {
		return false, nil
	}

	remaining := limitDuration - elapsed
	remainingMinutes := int(remaining.Minutes())
	if remaining%time.Minute != 0 {
		remainingMinutes++
	}

	promAIcallContactCaseRecreateRateLimitedTotal.Inc()

	return true, fmt.Errorf(
		"rate limit exceeded: aicall for contact_case reference_id %s was terminated %s ago, recreate blocked for %d more minutes",
		referenceID, elapsed.Round(time.Second), remainingMinutes,
	)
}

// startReferenceTypeNone starts a new aicall with no reference
func (h *aicallHandler) startReferenceTypeNone(
	ctx context.Context,
	c *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeNone",
		"ai":            c,
		"activeflow_id": activeflowID,
	})

	// start ai call
	tmp, err := h.startAIcallByMessaging(ctx, c, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeNone, uuid.Nil, false, teamParameter, currentMemberID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall with no reference")
	}
	log.WithField("aicall", tmp).Debugf("Created aicall. aicall_id: %s", tmp.ID)

	res, err := h.UpdateStatus(ctx, tmp.ID, aicall.StatusProgressing)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}

func (h *aicallHandler) getPipecatcallMessages(ctx context.Context, c *aicall.AIcall) ([]map[string]any, error) {

	// retrieve previous messages
	tmpMessages, err := h.messageHandler.List(ctx, 100, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get messages")
	}

	res := []map[string]any{}
	if len(tmpMessages) > 0 {
		// reverse the messages to have the correct order
		for i, j := 0, len(tmpMessages)-1; i < j; i, j = i+1, j-1 {
			tmpMessages[i], tmpMessages[j] = tmpMessages[j], tmpMessages[i]
		}

		for _, m := range tmpMessages {
			// skip non-LLM roles (e.g. notification) that would cause API errors
			if m.Role == message.RoleNotification {
				continue
			}

			tmp := map[string]any{
				"role":    string(m.Role),
				"content": string(m.Content),
			}

			if len(m.ToolCalls) > 0 {
				tmp["tool_calls"] = m.ToolCalls
			}

			if len(m.ToolCallID) > 0 {
				tmp["tool_call_id"] = m.ToolCallID
			}

			res = append(res, tmp)
		}
	}

	return res, nil
}

func (h *aicallHandler) getPipecatcallSTTType(c *aicall.AIcall) pmpipecatcall.STTType {
	if c.AISTTType != ai.STTTypeNone {
		return pmpipecatcall.STTType(c.AISTTType)
	}

	return defaultPipecatcallSTTType
}

func (h *aicallHandler) getPipecatcallTTSInfo(a *aicall.AIcall) (pmpipecatcall.TTSType, string) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "getPipecatcallTTSInfo",
		"aicall_id": a.ID,
	})

	// get tts type
	ttsType := defaultPipecatcallTTSType
	if a.AITTSType != ai.TTSTypeNone {
		ttsType = pmpipecatcall.TTSType(a.AITTSType)
	}

	// get voiceID
	ttsVoiceID, ok := mapDefaultTTSVoiceIDByTTSType[ai.TTSType(ttsType)]
	if !ok {
		log.Warnf("No default TTS voice ID found for TTSType: %v", ttsType)
		ttsVoiceID = ""
	}

	if a.AITTSVoiceID != "" {
		ttsVoiceID = a.AITTSVoiceID
	}

	return ttsType, ttsVoiceID
}

func (h *aicallHandler) startPipecatcall(ctx context.Context, c *aicall.AIcall) (*pmpipecatcall.Pipecatcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startPipecatcall",
		"aicall_id": c.ID,
	})

	// get llmMessages for pipecatcall
	llmType := pmpipecatcall.LLMType(c.AIEngineModel)
	llmMessages, err := h.getPipecatcallMessages(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages for pipecatcall")
	}
	log.Debugf("Got %d messages for pipecatcall", len(llmMessages))

	// determine stt and tts info
	ttsType := pmpipecatcall.TTSTypeNone
	ttsVoiceID := ""
	sttType := pmpipecatcall.STTTypeNone
	if c.ReferenceType == aicall.ReferenceTypeCall {
		log.Debugf("The aicall reference type is call. Getting tts and stt types for pipecatcall")
		ttsType, ttsVoiceID = h.getPipecatcallTTSInfo(c)
		sttType = h.getPipecatcallSTTType(c)
	}
	log.Debugf("Determined variables. sttType: %s, ttsType: %s, ttsVoiceID: %s for pipecatcall", sttType, ttsType, ttsVoiceID)

	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		c.PipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		llmType,
		llmMessages,
		sttType,
		c.STTLanguage,
		ttsType,
		"",
		ttsVoiceID,
	)
	if err != nil {
		log.Errorf("Could not start pipecatcall. err: %v", err)
		return nil, errors.Wrap(err, "could not start pipecatcall")
	}
	log.WithField("pipecatcall", res).Debugf("Started pipecatcall. pipecatcall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startPipecatcallTask(ctx context.Context, c *aicall.AIcall) (*pmpipecatcall.Pipecatcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startPipecatcallTask",
		"aicall_id": c.ID,
	})

	// get llmMessages for pipecatcall
	llmType := pmpipecatcall.LLMType(c.AIEngineModel)
	llmMessages, err := h.getPipecatcallMessages(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages for pipecatcall")
	}
	log.Debugf("Got %d messages for pipecatcall", len(llmMessages))

	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		c.PipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		llmType,
		llmMessages,
		pmpipecatcall.STTTypeNone,
		"",
		pmpipecatcall.TTSTypeNone,
		"",
		"",
	)
	if err != nil {
		log.Errorf("Could not start pipecatcall. err: %v", err)
		return nil, errors.Wrap(err, "could not start pipecatcall")
	}
	log.WithField("pipecatcall", res).Debugf("Started pipecatcall. pipecatcall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startInitMessages(ctx context.Context, a *ai.AI, c *aicall.AIcall, isTask bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startInitMessages",
		"aicall_id": c.ID,
	})

	messages := []string{}
	switch {
	case a.Type == ai.TypeInsight:
		messages = append(messages, InsightSystemPrompt)
	case isTask:
		messages = append(messages, defaultCommonAItaskSystemPrompt)
	default:
		messages = append(messages, defaultCommonAIcallSystemPrompt)
	}

	// parse init prompt
	if msg := h.getInitPrompt(ctx, a, c.ActiveflowID); msg != "" {
		messages = append(messages, msg)
	}
	log.Debugf("Parsed init prompt. aicall_id: %s", c.ID)

	// parse parameter (merged ai + team parameter)
	if msg := h.getDataAsJSON(ctx, c.Parameter, c.ActiveflowID); msg != "{}" {
		messages = append(messages, msg)
	}
	log.Debugf("Parsed parameter. aicall_id: %s", c.ID)

	for _, msg := range messages {
		tmp, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "",
			messagehandler.WithActiveAIID(a.ID))
		if err != nil {
			return errors.Wrapf(err, "could not create the init message to the ai. aicall_id: %s", c.ID)
		}
		log.WithField("message", tmp).Debugf("Created the init message to the ai. aicall_id: %s", c.ID)
	}

	return nil
}

// mergeParameters merges AI and team parameters, with team overriding AI on key collision.
// Returns nil if both are empty.
func mergeParameters(aiParam, teamParam map[string]any) map[string]any {
	merged := map[string]any{}
	for k, v := range aiParam {
		merged[k] = v
	}
	for k, v := range teamParam {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func (h *aicallHandler) startAIcallByRealtime(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	isTask bool,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startAIcallByRealtime",
		"ai_id":         a.ID,
		"activeflow_id": activeflowID,
	})

	parameter := mergeParameters(a.Parameter, teamParameter)

	// create ai call
	pipecatcallID := h.utilHandler.UUIDCreate()
	snapshots, autoAudit := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
	metadata := map[string]any{
		aicall.MetaKeyPromptSnapshots:  snapshots,
		aicall.MetaKeyAutoAuditEnabled: autoAudit,
	}
	res, err := h.Create(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
		confbridgeID, pipecatcallID, currentMemberID, parameter, metadata)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	if h.participantHandler != nil {
		if err := h.participantHandler.Create(ctx, res.ID, a.ID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v", res.ID, a.ID, err)
		}
	}

	// set activeflow variables
	if errSet := h.setActiveflowVariables(ctx, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the activeflow variables for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Set activeflow variables for aicall. aicall_id: %s", res.ID)

	// start initial messages
	if errInitMessages := h.startInitMessages(ctx, a, res, isTask); errInitMessages != nil {
		return nil, errors.Wrapf(errInitMessages, "could not start initial messages for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Initialized messages for aicall. aicall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startAIcallByMessaging(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	isTask bool,
	teamParameter map[string]any,
	currentMemberID uuid.UUID,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startAIcallByMessaging",
		"ai_id":         a.ID,
		"activeflow_id": activeflowID,
	})

	parameter := mergeParameters(a.Parameter, teamParameter)

	// create ai call
	pipecatcallID := h.utilHandler.UUIDCreate()
	snapshots, autoAudit := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
	metadata := map[string]any{
		aicall.MetaKeyPromptSnapshots:  snapshots,
		aicall.MetaKeyAutoAuditEnabled: autoAudit,
	}
	res, err := h.CreateByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
		pipecatcallID, currentMemberID, parameter, metadata)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	if h.participantHandler != nil {
		if err := h.participantHandler.Create(ctx, res.ID, a.ID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v", res.ID, a.ID, err)
		}
	}

	// set activeflow variables
	if errSet := h.setActiveflowVariables(ctx, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the activeflow variables for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Set activeflow variables for aicall. aicall_id: %s", res.ID)

	// start initial messages
	if errInitMessages := h.startInitMessages(ctx, a, res, isTask); errInitMessages != nil {
		return nil, errors.Wrapf(errInitMessages, "could not start initial messages for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Initialized messages for aicall. aicall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) StartTask(ctx context.Context, assistanceType aicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "StartTask",
		"assistance_type": assistanceType,
		"assistance_id":   assistanceID,
		"activeflow_id":   activeflowID,
	})
	log.Debugf("Starting a new aicall task")

	// resolve AI config based on assistance type
	c, teamParameter, currentMemberID, err := h.resolveAI(ctx, assistanceType, assistanceID)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve ai config")
	}

	res, err := h.startAIcallByMessaging(ctx, c, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeTask, uuid.Nil, true, teamParameter, currentMemberID)
	if err != nil {
		return nil, errors.Wrap(err, "could not start AIcall")
	}

	// start pipecatcall
	pc, err := h.startPipecatcallTask(ctx, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	if errTerminate := h.reqHandler.AIV1AIcallTerminateWithDelay(ctx, res.ID, defaultAITaskTimeout); errTerminate != nil {
		// note: the delayed termination request has failed, but we just log it and continue
		log.Errorf("Could not send the aicall terminate request correctly. err: %v", errTerminate)
	}

	return res, nil
}
