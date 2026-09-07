package aicallhandler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

// The three outcomes of one refresh evaluation, and the label set of
// aicall_insight_session_refresh_total. EVERY evaluation reports exactly one of
// them, so `kept` is the denominator the other two are read against.
const (
	insightSessionRefreshKept      = "kept"
	insightSessionRefreshRefreshed = "refreshed"
	insightSessionRefreshFailed    = "failed"
)

// insightSessionVariableMarker is the flow-variable opening token. Only a
// prompt that actually contains it can need an activeflow, which is what lets
// the refresh fail closed on substitution without breaking the (common) case of
// a reused AIcall whose original activeflow is long gone.
const insightSessionVariableMarker = "${"

// refreshInsightSessionIfIdle starts a NEW Insight assistant session on an
// AIcall that is being reused after an idle gap (VOIP-1484).
//
// One AIcall lives per Case, so an agent reopening a Case days later lands in
// the same thread and the model replays every old turn, including tool outputs
// that no longer describe anything real. This gives that thread a session
// boundary: it rewrites the system rows from the AI's CURRENT prompt and
// records the first new row's timestamp, after which the history builders
// replay only rows at or after it.
//
// It NEVER fails the panel open. Every failure path returns the AIcall exactly
// as it was found, leaving the previous session intact, and reports "failed" for
// the metric.
func (h *aicallHandler) refreshInsightSessionIfIdle(ctx context.Context, existing *aicall.AIcall) (*aicall.AIcall, string) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "refreshInsightSessionIfIdle",
		"aicall_id": existing.ID,
	})

	res, result := h.runInsightSessionRefresh(ctx, log, existing)
	promAIcallInsightSessionRefreshTotal.WithLabelValues(result).Inc()

	return res, result
}

// runInsightSessionRefresh is refreshInsightSessionIfIdle's body, split out so
// the metric is incremented on exactly one line for all three outcomes.
//
// The gate order is deliberate and cheapest-first: assistance type, then the
// config switch, then the idle test (one indexed message read), and only then
// the AI lookup (an RPC). Anything that can decide "kept" without a network
// call decides before one is made.
func (h *aicallHandler) runInsightSessionRefresh(ctx context.Context, log *logrus.Entry, existing *aicall.AIcall) (*aicall.AIcall, string) {
	// Step 1: cheap gates.
	if existing.AssistanceType != aicall.AssistanceTypeAI {
		// Teams carry one prompt snapshot per member and one member's AI is not
		// the whole thread's prompt; refreshing them is out of scope.
		return existing, insightSessionRefreshKept
	}

	minutes := config.Get().AIcallInsightSessionIdleMinutes
	if minutes <= 0 {
		// Disabled. The Go zero value must never be read as "refresh on every
		// open" -- that would rewrite the system rows on every panel open.
		return existing, insightSessionRefreshKept
	}
	threshold := time.Duration(minutes) * time.Minute

	// Step 2: is the thread actually idle?
	lastActivity, found, err := h.insightSessionLastActivity(ctx, existing)
	if err != nil {
		log.Errorf("Could not determine the insight session last activity. err: %v", err)
		return existing, insightSessionRefreshFailed
	}
	if !found {
		log.Warnf("No timestamp available to age the insight session, keeping the current session.")
		return existing, insightSessionRefreshKept
	}
	if time.Since(lastActivity) < threshold {
		return existing, insightSessionRefreshKept
	}

	// Step 3: the AI. AssistanceID is the AI id directly (step 1 gated on
	// AssistanceTypeAI), so the team-walking resolveActiveAIIDFromAIcall is
	// deliberately not used here.
	aiID := existing.AssistanceID
	if aiID == uuid.Nil {
		log.Warnf("The reused aicall has no assistance id, keeping the current session.")
		return existing, insightSessionRefreshKept
	}
	a, err := h.aiHandler.Get(ctx, aiID)
	if err != nil {
		log.Errorf("Could not get the ai for the insight session refresh. ai_id: %s, err: %v", aiID, err)
		return existing, insightSessionRefreshFailed
	}
	if a.Type != ai.TypeInsight {
		return existing, insightSessionRefreshKept
	}

	// Step 4: the prompt, resolved strictly. Fail closed rather than persist a
	// prompt still carrying a raw ${...}.
	initPrompt, paramJSON, err := h.refreshPrompt(ctx, existing, a)
	if err != nil {
		log.Errorf("Could not resolve the prompt for the insight session refresh, keeping the current session. err: %v", err)
		return existing, insightSessionRefreshFailed
	}

	// Step 5: write. Rows first, boundary second: the first row's own TMCreate
	// IS the boundary, so it cannot be known before the write.
	prompts := []string{InsightSystemPrompt}
	if initPrompt != "" {
		prompts = append(prompts, initPrompt)
	}
	if paramJSON != "" {
		prompts = append(prompts, paramJSON)
	}

	// These Creates publish aimessage_created webhooks mid-session; that is safe because both Insight panels render only user/assistant/tool rows (square-admin CaseInsightAssistantPanel.js:106-111), so a system row is never displayed.
	rows, err := h.writeSystemRows(ctx, a, existing, prompts)
	if err != nil {
		log.Errorf("Could not write the refreshed system rows, keeping the current session. created_message_ids: %v, err: %v", messageIDs(rows), err)
		return existing, insightSessionRefreshFailed
	}
	if len(rows) == 0 || rows[0].TMCreate == nil {
		log.Errorf("The refreshed system rows carry no usable create timestamp, keeping the current session. rows: %d", len(rows))
		return existing, insightSessionRefreshFailed
	}
	boundary := *rows[0].TMCreate

	if errWrite := h.writeInsightSessionMetadata(ctx, existing, a, boundary, initPrompt); errWrite != nil {
		log.Errorf("Could not write the insight session metadata. created_message_ids: %v, err: %v", messageIDs(rows), errWrite)
		return existing, insightSessionRefreshFailed
	}

	res, err := h.Get(ctx, existing.ID)
	if err != nil {
		log.Errorf("Could not re-read the aicall after the insight session refresh. err: %v", err)
		return existing, insightSessionRefreshFailed
	}

	log.WithField("aicall", res).Infof("Started a new insight session. aicall_id: %s, boundary: %s", existing.ID, boundary.UTC().Format(time.RFC3339Nano))
	return res, insightSessionRefreshRefreshed
}

// insightSessionLastActivity returns the newest evidence that this Insight
// thread was in use, as (timestamp, found, error).
//
// The terms are the current session boundary, the newest agent question, and
// the AIcall's own creation time. TMUpdate is DELIBERATELY not among them: it
// is bumped by every status write, so a thread nobody has touched in a week can
// still look fresh.
func (h *aicallHandler) insightSessionLastActivity(ctx context.Context, c *aicall.AIcall) (time.Time, bool, error) {
	last := time.Time{}
	found := false
	consider := func(t time.Time) {
		if !found || t.After(last) {
			last = t
			found = true
		}
	}

	if boundary, ok := insightSessionStart(c); ok {
		consider(boundary)
	}

	rows, err := h.messageHandler.List(ctx, 1, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldRole:     message.RoleUser,
		message.FieldDeleted:  false,
	})
	if err != nil {
		return time.Time{}, false, errors.Wrapf(err, "could not get the newest user message. aicall_id: %s", c.ID)
	}
	if len(rows) > 0 && rows[0].TMCreate != nil {
		consider(rows[0].TMCreate.UTC())
	}

	if c.TMCreate != nil {
		consider(c.TMCreate.UTC())
	}

	return last, found, nil
}

// writeInsightSessionMetadata records the new boundary and the refreshed prompt
// snapshot in one metadata write.
//
// The row is RE-READ first and its metadata copied key by key, so a listen
// start that wrote its own pointer between the panel open and this write is
// merged rather than clobbered. The write is the NoTouch variant on purpose:
// bumping TMUpdate here would make the next idle evaluation of any TMUpdate
// based rule read this bookkeeping write as agent activity.
func (h *aicallHandler) writeInsightSessionMetadata(ctx context.Context, existing *aicall.AIcall, a *ai.AI, boundary time.Time, initPrompt string) error {
	cur, err := h.db.AIcallGet(ctx, existing.ID)
	if err != nil {
		return errors.Wrapf(err, "could not re-read the aicall before writing the insight session metadata. aicall_id: %s", existing.ID)
	}

	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyInsightSessionStart] = boundary.UTC().Format(time.RFC3339Nano)
	metadata[aicall.MetaKeyPromptSnapshots] = []aicall.PromptSnapshot{
		{
			AIID:            a.ID,
			PromptHistoryID: a.CurrentPromptHistoryID,
			Prompt:          initPrompt,
		},
	}

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, existing.ID, map[aicall.Field]any{
		aicall.FieldMetadata: metadata,
	}); errUpdate != nil {
		return errors.Wrapf(errUpdate, "could not write the insight session metadata. aicall_id: %s", existing.ID)
	}

	return nil
}

// refreshPrompt resolves the system-row prompts for a new Insight session:
// the AI's current init prompt and, when there is one, the AIcall's parameter
// block as JSON.
//
// It is STRICT where getInitPrompt / getDataAsJSON are lenient. Those two fall
// back to the raw text on a substitution failure, which is right for a live
// turn and wrong here: a persisted system row still carrying ${...} would keep
// misinstructing the model for the whole session. So any substitution failure
// is returned, and the caller keeps the previous session.
//
// The "${" pre-check keeps that strictness from punishing the common case: a
// reused AIcall's activeflow is the one from its original panel open and may be
// long ended, but a prompt with no variables never needs it.
func (h *aicallHandler) refreshPrompt(ctx context.Context, existing *aicall.AIcall, a *ai.AI) (string, string, error) {
	initPrompt := a.InitPrompt
	if strings.Contains(initPrompt, insightSessionVariableMarker) {
		if existing.ActiveflowID == uuid.Nil {
			return "", "", errors.Errorf("the init prompt carries flow variables but the aicall has no activeflow. aicall_id: %s", existing.ID)
		}

		substituted, err := h.substituteText(ctx, existing.ActiveflowID, initPrompt)
		if err != nil {
			return "", "", errors.Wrapf(err, "could not substitute the init prompt. aicall_id: %s", existing.ID)
		}
		initPrompt = substituted
	}

	paramJSON, err := h.refreshParameter(ctx, existing)
	if err != nil {
		return "", "", err
	}

	return initPrompt, paramJSON, nil
}

// refreshParameter renders the AIcall's parameter block for a refreshed session,
// returning "" when there is nothing worth writing as a system row.
func (h *aicallHandler) refreshParameter(ctx context.Context, existing *aicall.AIcall) (string, error) {
	if len(existing.Parameter) == 0 {
		return "", nil
	}

	raw, err := json.Marshal(existing.Parameter)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal the parameter. aicall_id: %s", existing.ID)
	}

	if !strings.Contains(string(raw), insightSessionVariableMarker) {
		// No variables anywhere in the block: no RPC, and no need for a live
		// activeflow.
		return skipEmptyJSONObject(string(raw)), nil
	}

	if existing.ActiveflowID == uuid.Nil {
		return "", errors.Errorf("the parameter carries flow variables but the aicall has no activeflow. aicall_id: %s", existing.ID)
	}

	substituted, err := h.substituteValue(ctx, existing.ActiveflowID, existing.Parameter)
	if err != nil {
		return "", errors.Wrapf(err, "could not substitute the parameter. aicall_id: %s", existing.ID)
	}

	out, err := json.Marshal(substituted)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal the substituted parameter. aicall_id: %s", existing.ID)
	}

	return skipEmptyJSONObject(string(out)), nil
}

// skipEmptyJSONObject maps the empty JSON object to "", so an empty parameter
// block is skipped rather than written as a useless "{}" system row -- the same
// rule startInitMessages applies.
func skipEmptyJSONObject(s string) string {
	if s == "{}" {
		return ""
	}
	return s
}

// messageIDs is for failure logs: the rows a partially completed refresh
// already created sit after the OLD boundary and are replayed until the next
// successful refresh, so their ids belong in the log that reports the failure.
func messageIDs(messages []*message.Message) []uuid.UUID {
	res := make([]uuid.UUID, 0, len(messages))
	for _, m := range messages {
		res = append(res, m.ID)
	}
	return res
}
