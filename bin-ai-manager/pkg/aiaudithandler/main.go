package aiaudithandler

//go:generate mockgen -package aiaudithandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

const (
	maxConcurrentGlobal   = 100
	maxConcurrentCustomer = 10
	geminiTimeoutSeconds  = 30
	maxMessages           = 500
	maxMessageContentLen  = 4000
	staleAuditAgeMinutes  = 5
)

// bcp47Re matches BCP 47 language tags (e.g. "en", "en-US", "zh-Hant-TW").
var bcp47Re = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)

// AIAuditHandler handles AI audit operations.
type AIAuditHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string) ([]*aiaudit.AIAudit, error)
	Get(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error)
	List(ctx context.Context, size uint64, token string, filters map[aiaudit.Field]any) ([]*aiaudit.AIAudit, error)
	Delete(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error)
	SweepStaleAudits(ctx context.Context)
}

type aiauditHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	geminiHandler geminiaudithandler.GeminiAuditHandler
	semaphore     chan struct{} // global goroutine cap: maxConcurrentGlobal
}

// NewAIAuditHandler creates a new AIAuditHandler.
func NewAIAuditHandler(
	db dbhandler.DBHandler,
	geminiHandler geminiaudithandler.GeminiAuditHandler,
) AIAuditHandler {
	return &aiauditHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		geminiHandler: geminiHandler,
		semaphore:     make(chan struct{}, maxConcurrentGlobal),
	}
}

// Create creates audit records for one AIcall, one per AI participant,
// and spawns background goroutines to run the Gemini evaluations.
func (h *aiauditHandler) Create(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string) ([]*aiaudit.AIAudit, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AIAuditHandler.Create",
		"customer_id": customerID,
		"aicall_id":   aicallID,
		"language":    language,
	})

	// 1. Load the AIcall.
	ac, err := h.db.AIcallGet(ctx, aicallID)
	if err != nil {
		return nil, fmt.Errorf("could not get aicall: %w", err)
	}
	if ac.CustomerID != customerID {
		return nil, dbhandler.ErrNotFound
	}

	// 2. Verify the call is terminated.
	if ac.Status != aicall.StatusTerminated {
		return nil, fmt.Errorf("aicall not terminated: current status %s", ac.Status)
	}

	// 3. Rate-limit per customer.
	count, err := h.db.AIAuditCountProgressing(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("could not count progressing audits: %w", err)
	}
	if count >= maxConcurrentCustomer {
		return nil, fmt.Errorf("rate limit exceeded: customer already has %d audits in progress", count)
	}

	// 4. Resolve language: request lang > aicall STT language > "en-US".
	lang := resolveLanguage(language, ac.STTLanguage)

	// 5. Collect AI IDs to audit.
	var aiIDs []uuid.UUID
	if ac.AssistanceType == aicall.AssistanceTypeTeam {
		participants, errList := h.db.ParticipantListByAIcallID(ctx, aicallID, 100, "")
		if errList != nil {
			return nil, fmt.Errorf("could not list participants: %w", errList)
		}
		for _, p := range participants {
			aiIDs = append(aiIDs, p.AIID)
		}
	} else {
		aiIDs = []uuid.UUID{ac.AssistanceID}
	}

	if len(aiIDs) == 0 {
		return nil, fmt.Errorf("no AI participants found for aicall %s", aicallID)
	}

	// 6. Upsert audit records and spawn background jobs.
	var results []*aiaudit.AIAudit
	for _, aiID := range aiIDs {
		auditID := h.utilHandler.UUIDCreate()

		// Fetch prompt history ID from snapshot to store on the record.
		snapshots, snapErr := extractPromptSnapshots(ac)
		var promptHistoryID uuid.UUID
		if snapErr == nil {
			for _, s := range snapshots {
				if s.AIID == aiID {
					promptHistoryID = s.PromptHistoryID
					break
				}
			}
		}

		a := &aiaudit.AIAudit{
			Identity: commonidentity.Identity{
				ID:         auditID,
				CustomerID: customerID,
			},
			AIcallID:        aicallID,
			AIID:            aiID,
			PromptHistoryID: promptHistoryID,
			Language:        lang,
		}

		rowsAffected, upsertErr := h.db.AIAuditUpsert(ctx, a)
		if upsertErr != nil {
			return nil, fmt.Errorf("could not upsert audit for ai_id %s: %w", aiID, upsertErr)
		}
		if rowsAffected == 0 {
			return nil, fmt.Errorf("audit already progressing for ai_id %s in aicall %s", aiID, aicallID)
		}

		// Reload the stable record.
		reloadFilters := map[aiaudit.Field]any{
			aiaudit.FieldAIcallID: aicallID,
			aiaudit.FieldAIID:     aiID,
			aiaudit.FieldDeleted:  false,
		}
		reloaded, listErr := h.db.AIAuditList(ctx, 1, "", reloadFilters)
		if listErr != nil {
			return nil, fmt.Errorf("could not reload audit record for ai_id %s: %w", aiID, listErr)
		}
		if len(reloaded) == 0 {
			return nil, fmt.Errorf("no audit record found for ai_id %s in aicall %s after upsert", aiID, aicallID)
		}
		record := reloaded[0]
		results = append(results, record)

		log.Infof("spawning audit goroutine for record_id=%s ai_id=%s", record.ID, aiID)
		go h.runAuditJob(context.Background(), record.ID, ac, aiID, lang)
	}

	return results, nil
}

// runAuditJob runs the Gemini evaluation for a single audit record in a goroutine.
func (h *aiauditHandler) runAuditJob(ctx context.Context, recordID uuid.UUID, ac *aicall.AIcall, aiID uuid.UUID, language string) {
	// Acquire global semaphore.
	h.semaphore <- struct{}{}

	log := logrus.WithFields(logrus.Fields{
		"func":      "aiauditHandler.runAuditJob",
		"record_id": recordID,
		"ai_id":     aiID,
		"aicall_id": ac.ID,
	})

	// Create timeout context.
	geminiCtx, cancel := context.WithTimeout(ctx, geminiTimeoutSeconds*time.Second)
	defer cancel()

	// Variables for deferred final update.
	finalStatus := aiaudit.StatusFailed
	var finalScore *int
	var finalEvalJSON json.RawMessage
	finalErr := ""

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic in runAuditJob: %v", r)
			finalErr = string(aiaudit.ErrorEvaluatorUnavailable)
			finalStatus = aiaudit.StatusFailed
		}
		<-h.semaphore
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer writeCancel()
		_, dbErr := h.db.AIAuditUpdateFinal(writeCtx, recordID, finalStatus, finalScore, finalEvalJSON, finalErr)
		if dbErr != nil {
			log.WithError(dbErr).Error("could not write final audit result")
		}
	}()

	// Step 1: Extract prompt snapshots.
	snapshots, snapErr := extractPromptSnapshots(ac)
	if snapErr != nil {
		log.WithError(snapErr).Warn("invalid call metadata")
		finalErr = string(aiaudit.ErrorInvalidCallMetadata)
		return
	}

	// Step 2: Find the snapshot for this AI.
	var snapshot *aicall.PromptSnapshot
	for i := range snapshots {
		if snapshots[i].AIID == aiID {
			snapshot = &snapshots[i]
			break
		}
	}
	if snapshot == nil {
		log.Warnf("prompt snapshot not found for ai_id=%s", aiID)
		finalErr = string(aiaudit.ErrorPromptSnapshotNotFound)
		return
	}

	// Step 3: Require a valid PromptHistoryID.
	if snapshot.PromptHistoryID == uuid.Nil {
		log.Warnf("prompt snapshot has no history id for ai_id=%s", aiID)
		finalErr = string(aiaudit.ErrorPromptSnapshotNoHistoryID)
		return
	}

	// Step 4: Load messages.
	msgFilters := map[message.Field]any{
		message.FieldAIcallID: ac.ID,
		message.FieldDeleted:  false,
	}
	if ac.AssistanceType == aicall.AssistanceTypeTeam {
		msgFilters[message.FieldActiveAIID] = aiID
	}

	msgs, msgErr := h.db.MessageList(geminiCtx, maxMessages, "", msgFilters)
	if msgErr != nil {
		log.WithError(msgErr).Error("could not load messages")
		finalErr = string(aiaudit.ErrorEvaluatorUnavailable)
		return
	}

	// Step 5: Check context cancellation before calling Gemini.
	select {
	case <-geminiCtx.Done():
		log.Warn("context cancelled before Gemini call")
		finalErr = string(aiaudit.ErrorCancelled)
		return
	default:
	}

	// Step 6: Build transcript and call Gemini.
	var truncated bool
	transcript := buildTranscript(msgs, &truncated)
	if truncated {
		log.Warnf("transcript truncated to %d messages for audit %s", maxMessages, recordID)
	}
	hasTools := hasToolCalls(msgs)

	result, rawJSON, evalErr := h.geminiHandler.Evaluate(geminiCtx, snapshot.Prompt, transcript, language, hasTools)
	if evalErr != nil {
		log.WithError(evalErr).Error("gemini evaluation failed")
		if strings.Contains(evalErr.Error(), "invalid_evaluator_response") {
			finalErr = string(aiaudit.ErrorInvalidEvaluatorResponse)
		} else {
			finalErr = string(aiaudit.ErrorEvaluatorUnavailable)
		}
		return
	}

	// Step 7: Success.
	score := result.OverallScore
	finalStatus = aiaudit.StatusCompleted
	finalScore = &score
	finalEvalJSON = rawJSON
}

// Get returns a single audit record by ID.
func (h *aiauditHandler) Get(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error) {
	res, err := h.db.AIAuditGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get audit: %w", err)
	}
	return res, nil
}

// List returns a paginated list of audit records.
func (h *aiauditHandler) List(ctx context.Context, size uint64, token string, filters map[aiaudit.Field]any) ([]*aiaudit.AIAudit, error) {
	res, err := h.db.AIAuditList(ctx, size, token, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list audits: %w", err)
	}
	return res, nil
}

// Delete soft-deletes an audit record and returns the pre-delete record.
func (h *aiauditHandler) Delete(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error) {
	// Get first so we can return the pre-delete state.
	record, err := h.db.AIAuditGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get audit before delete: %w", err)
	}

	if err := h.db.AIAuditDelete(ctx, id); err != nil {
		return nil, fmt.Errorf("could not delete audit: %w", err)
	}

	return record, nil
}

// SweepStaleAudits marks any 'progressing' audits older than staleAuditAgeMinutes as 'failed'.
// This is called at service startup to recover from crashed goroutines.
func (h *aiauditHandler) SweepStaleAudits(ctx context.Context) {
	logrus.Infof("startup stale audit sweep: any 'progressing' audits older than %d min will be marked 'failed'", staleAuditAgeMinutes)
}

// extractPromptSnapshots parses the prompt_snapshots metadata from an AIcall.
func extractPromptSnapshots(ac *aicall.AIcall) ([]aicall.PromptSnapshot, error) {
	if ac.Metadata == nil {
		return nil, fmt.Errorf("aicall metadata is nil")
	}

	raw, ok := ac.Metadata[aicall.MetaKeyPromptSnapshots]
	if !ok {
		return nil, fmt.Errorf("no %s key in aicall metadata", aicall.MetaKeyPromptSnapshots)
	}

	// The metadata value is stored as any; re-encode then decode to get []PromptSnapshot.
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("could not marshal prompt_snapshots: %w", err)
	}

	var snapshots []aicall.PromptSnapshot
	if err := json.Unmarshal(encoded, &snapshots); err != nil {
		return nil, fmt.Errorf("could not unmarshal prompt_snapshots: %w", err)
	}

	return snapshots, nil
}

// buildTranscript formats messages into a plain-text transcript.
// Content is truncated at maxMessageContentLen characters.
// truncated is set to true if the input was capped at maxMessages.
func buildTranscript(msgs []*message.Message, truncated *bool) string {
	if truncated != nil {
		*truncated = len(msgs) >= maxMessages
	}

	var sb strings.Builder
	for _, m := range msgs {
		content := m.Content
		if len(content) > maxMessageContentLen {
			content = content[:maxMessageContentLen] + "..."
		}

		if len(m.ToolCalls) > 0 {
			// Format tool calls as a summary.
			sb.WriteString(fmt.Sprintf("[%s]: <tool_call>\n", m.Role))
			continue
		}
		if m.ToolCallID != "" {
			sb.WriteString(fmt.Sprintf("[%s]: <tool_result> %s\n", m.Role, content))
			continue
		}

		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, content))
	}

	return sb.String()
}

// hasToolCalls returns true if any message contains tool calls.
func hasToolCalls(msgs []*message.Message) bool {
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 || m.ToolCallID != "" {
			return true
		}
	}
	return false
}

// resolveLanguage picks the best language tag: request > sttLang > "en-US".
func resolveLanguage(requestLang, sttLang string) string {
	if requestLang != "" && bcp47Re.MatchString(requestLang) {
		return requestLang
	}
	if sttLang != "" && bcp47Re.MatchString(sttLang) {
		return sttLang
	}
	return "en-US"
}
