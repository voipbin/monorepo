package dbhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func newProposalCoverageHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	return &handler{utilHandler: mockUtil, db: dbTest}, mockUtil
}

// seedReadableProposal inserts a proposal row like insertTestProposal but with
// an explicit 16 byte applied_prompt_history_id. The sqlite stand-in schema
// defaults that column to an integer literal, which the row scanner cannot
// read back as a uuid; production rows always carry 16 bytes.
func seedReadableProposal(t *testing.T, id, customerID, aiID, basisHistID uuid.UUID, status aipromptproposal.Status) {
	t.Helper()
	if _, err := dbTest.Exec(
		`INSERT INTO ai_ai_prompt_proposals
		 (id, customer_id, ai_id, audit_ids, basis_prompt_history_id, status, error, applied_prompt_history_id, tm_create)
		 VALUES (?, ?, ?, ?, ?, ?, '', ?, ?)`,
		id.Bytes(), customerID.Bytes(), aiID.Bytes(),
		`[]`,
		basisHistID.Bytes(),
		string(status),
		uuid.Nil.Bytes(),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05.000000"),
	); err != nil {
		t.Fatalf("seedReadableProposal: %v", err)
	}
}

// proposalStatusOf reads a proposal status straight from the table.
func proposalStatusOf(t *testing.T, id uuid.UUID) string {
	t.Helper()

	var status string
	if err := dbTest.QueryRow(`SELECT status FROM ai_ai_prompt_proposals WHERE id = ?`, id.Bytes()).Scan(&status); err != nil {
		t.Fatalf("could not read the stored status. err: %v", err)
	}
	return status
}

func Test_AIPromptProposalCreate_Get(t *testing.T) {
	resetTestDB(t)

	h, mockUtil := newProposalCoverageHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	basisID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")
	auditID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	// Create forces the lifecycle fields regardless of what the caller passed.
	mockUtil.EXPECT().TimeNow().Return(&now)
	in := &aipromptproposal.AIPromptProposal{
		Identity:             identity.Identity{ID: id, CustomerID: customerID},
		AIID:                 aiID,
		AuditIDs:             []uuid.UUID{auditID},
		BasisPromptHistoryID: basisID,
		OriginalPrompt:       "original",
		Status:               aipromptproposal.StatusAccepted,
		Error:                "stale",
		TMUpdate:             &now,
		TMDelete:             &now,
	}
	if err := h.AIPromptProposalCreate(ctx, in); err != nil {
		t.Fatalf("create failed. err: %v", err)
	}
	if in.Status != aipromptproposal.StatusProgressing || in.Error != "" || in.TMCreate == nil || !in.TMCreate.Equal(now) {
		t.Errorf("create must reset lifecycle fields on the input. got: %+v", in)
	}

	got, err := h.AIPromptProposalGet(ctx, id)
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if got.TMUpdate != nil || got.TMDelete != nil {
		t.Errorf("create must clear tm_update and tm_delete. got: %v, %v", got.TMUpdate, got.TMDelete)
	}
	if got.Status != aipromptproposal.StatusProgressing || got.AIID != aiID || got.OriginalPrompt != "original" {
		t.Errorf("row mismatch. got: %+v", got)
	}
	if len(got.AuditIDs) != 1 || got.AuditIDs[0] != auditID {
		t.Errorf("audit ids mismatch. got: %v", got.AuditIDs)
	}

	// a second create with the same id violates the primary key.
	mockUtil.EXPECT().TimeNow().Return(&now)
	if err := h.AIPromptProposalCreate(ctx, in); err == nil {
		t.Errorf("expected a duplicate id to be rejected")
	}

	// unknown id.
	_, err = h.AIPromptProposalGet(ctx, uuid.FromStringOrNil("e1000000-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func Test_AIPromptProposalUpdateFinal(t *testing.T) {
	resetTestDB(t)

	h, mockUtil := newProposalCoverageHandler(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	basisID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")
	progressingID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001")
	completedID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000002")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	seedReadableProposal(t, progressingID, customerID, aiID, basisID, aipromptproposal.StatusProgressing)
	seedReadableProposal(t, completedID, customerID, aiID, basisID, aipromptproposal.StatusCompleted)

	// a progressing proposal of another customer must not be written either.
	bystanderID := uuid.FromStringOrNil("e1000000-0000-0000-0000-0000000000b1")
	seedReadableProposal(t, bystanderID, uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000002"), aiID, basisID, aipromptproposal.StatusProgressing)

	// only a progressing row is written.
	mockUtil.EXPECT().TimeNow().Return(&now)
	n, err := h.AIPromptProposalUpdateFinal(ctx, progressingID, aipromptproposal.StatusCompleted, "new prompt", "because", "")
	if err != nil || n != 1 {
		t.Fatalf("expected one row to be written. n: %d, err: %v", n, err)
	}
	got, err := h.AIPromptProposalGet(ctx, progressingID)
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if got.Status != aipromptproposal.StatusCompleted || got.ProposedPrompt != "new prompt" || got.Rationale != "because" || got.TMUpdate == nil {
		t.Errorf("final result mismatch. got: %+v", got)
	}

	if status := proposalStatusOf(t, bystanderID); status != string(aipromptproposal.StatusProgressing) {
		t.Errorf("another proposal was written by the final update. status: %s", status)
	}

	// a second final write must not overwrite the completed row.
	mockUtil.EXPECT().TimeNow().Return(&now)
	n, err = h.AIPromptProposalUpdateFinal(ctx, progressingID, aipromptproposal.StatusFailed, "", "", "evaluator_unavailable")
	if err != nil || n != 0 {
		t.Errorf("a finished row must not be rewritten. n: %d, err: %v", n, err)
	}

	// a failure writes the error and no prompt.
	failedID := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000003")
	seedReadableProposal(t, failedID, customerID, aiID, basisID, aipromptproposal.StatusProgressing)
	mockUtil.EXPECT().TimeNow().Return(&now)
	n, err = h.AIPromptProposalUpdateFinal(ctx, failedID, aipromptproposal.StatusFailed, "", "", "evaluator_unavailable")
	if err != nil || n != 1 {
		t.Fatalf("expected the failure to be written. n: %d, err: %v", n, err)
	}
	got, err = h.AIPromptProposalGet(ctx, failedID)
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if got.Status != aipromptproposal.StatusFailed || got.Error != "evaluator_unavailable" || got.ProposedPrompt != "" {
		t.Errorf("failure mismatch. got: %+v", got)
	}

	// empty prompt and rationale are stored as NULL, not as empty strings.
	var promptIsNull, rationaleIsNull bool
	if errQuery := dbTest.QueryRow(
		`SELECT proposed_prompt IS NULL, rationale IS NULL FROM ai_ai_prompt_proposals WHERE id = ?`, failedID.Bytes(),
	).Scan(&promptIsNull, &rationaleIsNull); errQuery != nil {
		t.Fatalf("could not read the stored columns. err: %v", errQuery)
	}
	if !promptIsNull || !rationaleIsNull {
		t.Errorf("expected NULL columns for empty values. prompt null: %v, rationale null: %v", promptIsNull, rationaleIsNull)
	}
}

func Test_AIPromptProposalUpdateExpired_UpdateRejected(t *testing.T) {
	resetTestDB(t)

	h, mockUtil := newProposalCoverageHandler(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	basisID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name         string
		seedStatus   aipromptproposal.Status
		call         func(h *handler, id uuid.UUID) (int64, error)
		expectRows   int64
		expectStatus aipromptproposal.Status
		expectError  string
	}{
		{
			name:       "expire a completed proposal",
			seedStatus: aipromptproposal.StatusCompleted,
			call: func(h *handler, id uuid.UUID) (int64, error) {
				return h.AIPromptProposalUpdateExpired(ctx, id, "prompt_version_drifted")
			},
			expectRows:   1,
			expectStatus: aipromptproposal.StatusExpired,
			expectError:  "prompt_version_drifted",
		},
		{
			name:       "expire is refused for a progressing proposal",
			seedStatus: aipromptproposal.StatusProgressing,
			call: func(h *handler, id uuid.UUID) (int64, error) {
				return h.AIPromptProposalUpdateExpired(ctx, id, "prompt_version_drifted")
			},
			expectRows:   0,
			expectStatus: aipromptproposal.StatusProgressing,
		},
		{
			name:         "reject a completed proposal",
			seedStatus:   aipromptproposal.StatusCompleted,
			call:         func(h *handler, id uuid.UUID) (int64, error) { return h.AIPromptProposalUpdateRejected(ctx, id) },
			expectRows:   1,
			expectStatus: aipromptproposal.StatusRejected,
		},
		{
			name:         "reject is refused for an accepted proposal",
			seedStatus:   aipromptproposal.StatusAccepted,
			call:         func(h *handler, id uuid.UUID) (int64, error) { return h.AIPromptProposalUpdateRejected(ctx, id) },
			expectRows:   0,
			expectStatus: aipromptproposal.StatusAccepted,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.FromStringOrNil(fmt.Sprintf("e2000000-0000-0000-0000-%012d", i+1))
			seedReadableProposal(t, id, customerID, aiID, basisID, tt.seedStatus)

			// a proposal of another customer in the same state must not change.
			bystanderID := uuid.FromStringOrNil(fmt.Sprintf("e2100000-0000-0000-0000-%012d", i+1))
			seedReadableProposal(t, bystanderID, uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000002"), aiID, basisID, tt.seedStatus)

			mockUtil.EXPECT().TimeNow().Return(&now)
			n, err := tt.call(h, id)
			if err != nil {
				t.Fatalf("unexpected error. err: %v", err)
			}
			if n != tt.expectRows {
				t.Errorf("rows affected mismatch. expected: %d, got: %d", tt.expectRows, n)
			}

			if status := proposalStatusOf(t, bystanderID); status != string(tt.seedStatus) {
				t.Errorf("another proposal was written. expected: %s, got: %s", tt.seedStatus, status)
			}

			got, err := h.AIPromptProposalGet(ctx, id)
			if err != nil {
				t.Fatalf("get failed. err: %v", err)
			}
			if got.Status != tt.expectStatus || got.Error != tt.expectError {
				t.Errorf("state mismatch. expected: %s/%q, got: %s/%q", tt.expectStatus, tt.expectError, got.Status, got.Error)
			}
		})
	}
}

func Test_AIPromptProposalDelete_CountProgressing_List(t *testing.T) {
	resetTestDB(t)

	h, mockUtil := newProposalCoverageHandler(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	otherCustomerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000002")
	aiID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	basisID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")
	p1 := uuid.FromStringOrNil("e3000000-0000-0000-0000-000000000001")
	p2 := uuid.FromStringOrNil("e3000000-0000-0000-0000-000000000002")
	p3 := uuid.FromStringOrNil("e3000000-0000-0000-0000-000000000003")
	p4 := uuid.FromStringOrNil("e3000000-0000-0000-0000-000000000004")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	seedReadableProposal(t, p1, customerID, aiID, basisID, aipromptproposal.StatusProgressing)
	seedReadableProposal(t, p2, customerID, aiID, basisID, aipromptproposal.StatusProgressing)
	seedReadableProposal(t, p3, customerID, aiID, basisID, aipromptproposal.StatusCompleted)
	seedReadableProposal(t, p4, otherCustomerID, aiID, basisID, aipromptproposal.StatusProgressing)

	// seedReadableProposal stamps every row with the same fixed tm_create;
	// give p3 a later one so list ordering is observable below.
	if _, errExec := dbTest.Exec(`UPDATE ai_ai_prompt_proposals SET tm_create = ? WHERE id = ?`,
		"2024-01-02 00:00:00.000000", p3.Bytes()); errExec != nil {
		t.Fatalf("could not set p3's tm_create. err: %v", errExec)
	}

	count, err := h.AIPromptProposalCountProgressing(ctx, customerID)
	if err != nil || count != 2 {
		t.Fatalf("expected two in-flight proposals for the customer. count: %d, err: %v", count, err)
	}

	// soft delete removes the row from the count and from reads, and stamps
	// both timestamps.
	deleteTime := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&deleteTime)
	if errDelete := h.AIPromptProposalDelete(ctx, p1); errDelete != nil {
		t.Fatalf("delete failed. err: %v", errDelete)
	}
	var tmUpdateIsNull bool
	if errQuery := dbTest.QueryRow(`SELECT tm_update IS NULL FROM ai_ai_prompt_proposals WHERE id = ?`, p1.Bytes()).Scan(&tmUpdateIsNull); errQuery != nil {
		t.Fatalf("could not read the stored tm_update. err: %v", errQuery)
	}
	if tmUpdateIsNull {
		t.Errorf("delete must stamp tm_update along with tm_delete")
	}
	count, err = h.AIPromptProposalCountProgressing(ctx, customerID)
	if err != nil || count != 1 {
		t.Errorf("a deleted proposal must not count. count: %d, err: %v", count, err)
	}
	if _, errGet := h.AIPromptProposalGet(ctx, p1); !errors.Is(errGet, ErrNotFound) {
		t.Errorf("a deleted proposal must read as not found, got: %v", errGet)
	}

	// deleting it again, or deleting an unknown id, reports not found.
	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	if errDelete := h.AIPromptProposalDelete(ctx, p1); !errors.Is(errDelete, ErrNotFound) {
		t.Errorf("expected ErrNotFound for a second delete, got: %v", errDelete)
	}
	if errDelete := h.AIPromptProposalDelete(ctx, uuid.FromStringOrNil("e3000000-0000-0000-0000-0000000000ff")); !errors.Is(errDelete, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an unknown id, got: %v", errDelete)
	}

	// the list is scoped by the filter and by the paging token.
	list, err := h.AIPromptProposalList(ctx, 10, "2099-01-01 00:00:00.000000", map[aipromptproposal.Field]any{
		aipromptproposal.FieldCustomerID: customerID,
		aipromptproposal.FieldDeleted:    false,
	})
	if err != nil {
		t.Fatalf("list failed. err: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected the two live proposals of the customer, got: %d", len(list))
	}
	for _, p := range list {
		if p.CustomerID != customerID || p.ID == p1 {
			t.Errorf("list leaked a foreign or deleted row: %+v", p)
		}
	}
	// the list must return the newest-first order the query declares
	// (ORDER BY tm_create desc); p3 was given the later tm_create above.
	if list[0].ID != p3 || list[1].ID != p2 {
		t.Errorf("list must be ordered newest tm_create first. got: [%s, %s]", list[0].ID, list[1].ID)
	}

	// size bounds the page.
	list, err = h.AIPromptProposalList(ctx, 1, "2099-01-01 00:00:00.000000", map[aipromptproposal.Field]any{aipromptproposal.FieldCustomerID: customerID})
	if err != nil || len(list) != 1 {
		t.Errorf("expected a page of one. len: %d, err: %v", len(list), err)
	}

	// the token is an exclusive upper bound on tm_create (p2 was created at
	// 2024-01-01 00:00:00, p3 was moved to 2024-01-02 above and p1 was
	// deleted): a token equal to p2's time excludes it, one microsecond
	// later includes it.
	for _, tc := range []struct {
		token  string
		expect int
	}{
		{"2024-01-01 00:00:00.000000", 0},
		{"2023-12-31 23:59:59.999999", 0},
		{"2024-01-01 00:00:00.000001", 1},
	} {
		list, err = h.AIPromptProposalList(ctx, 10, tc.token, map[aipromptproposal.Field]any{
			aipromptproposal.FieldCustomerID: customerID,
			aipromptproposal.FieldDeleted:    false,
		})
		if err != nil {
			t.Fatalf("list with token %s failed. err: %v", tc.token, err)
		}
		if len(list) != tc.expect {
			t.Errorf("token %s: expected %d rows, got: %d", tc.token, tc.expect, len(list))
		}
	}

	// an empty token falls back to the current time supplied by the util handler.
	mockUtil.EXPECT().TimeGetCurTime().Return("2099-01-01 00:00:00.000000")
	list, err = h.AIPromptProposalList(ctx, 10, "", map[aipromptproposal.Field]any{aipromptproposal.FieldCustomerID: otherCustomerID})
	if err != nil || len(list) != 1 || list[0].ID != p4 {
		t.Errorf("expected the other customer's single row. list: %v, err: %v", list, err)
	}
}

func Test_AIAuditDelete(t *testing.T) {
	resetTestDB(t)

	h, mockUtil := newProposalCoverageHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("e4000000-0000-0000-0000-000000000001")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	insertTestAudit(t, &aiaudit.AIAudit{
		Identity: identity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")},
	}, aiaudit.StatusCompleted, nil, &now)

	mockUtil.EXPECT().TimeNow().Return(&now)
	if err := h.AIAuditDelete(ctx, id); err != nil {
		t.Fatalf("delete failed. err: %v", err)
	}
	if _, err := h.AIAuditGet(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("a deleted audit must read as not found, got: %v", err)
	}

	// already deleted, and unknown ids, both report not found.
	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	if err := h.AIAuditDelete(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for a second delete, got: %v", err)
	}
	if err := h.AIAuditDelete(ctx, uuid.FromStringOrNil("e4000000-0000-0000-0000-0000000000ff")); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an unknown id, got: %v", err)
	}
}

// Test_AIPromptProposalUpdates_SkipSoftDeleted pins that none of the guarded
// status writes can touch a proposal that has been soft deleted.
func Test_AIPromptProposalUpdates_SkipSoftDeleted(t *testing.T) {
	resetTestDB(t)

	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	basisID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	ctx := context.Background()

	tests := []struct {
		name       string
		seedStatus aipromptproposal.Status
		call       func(h *handler, id uuid.UUID) (int64, error)
	}{
		{
			name:       "final write",
			seedStatus: aipromptproposal.StatusProgressing,
			call: func(h *handler, id uuid.UUID) (int64, error) {
				return h.AIPromptProposalUpdateFinal(ctx, id, aipromptproposal.StatusCompleted, "p", "r", "")
			},
		},
		{
			name:       "expire",
			seedStatus: aipromptproposal.StatusCompleted,
			call: func(h *handler, id uuid.UUID) (int64, error) {
				return h.AIPromptProposalUpdateExpired(ctx, id, "prompt_version_drifted")
			},
		},
		{
			name:       "reject",
			seedStatus: aipromptproposal.StatusCompleted,
			call:       func(h *handler, id uuid.UUID) (int64, error) { return h.AIPromptProposalUpdateRejected(ctx, id) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTestDB(t)

			h, mockUtil := newProposalCoverageHandler(t)
			id := uuid.FromStringOrNil("e6000000-0000-0000-0000-000000000001")
			seedReadableProposal(t, id, customerID, aiID, basisID, tt.seedStatus)

			mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
			if err := h.AIPromptProposalDelete(ctx, id); err != nil {
				t.Fatalf("delete failed. err: %v", err)
			}

			n, err := tt.call(h, id)
			if err != nil {
				t.Fatalf("unexpected error. err: %v", err)
			}
			if n != 0 {
				t.Errorf("a soft deleted proposal must not be written, rows affected: %d", n)
			}

			var status string
			if errQuery := dbTest.QueryRow(`SELECT status FROM ai_ai_prompt_proposals WHERE id = ?`, id.Bytes()).Scan(&status); errQuery != nil {
				t.Fatalf("could not read the stored status. err: %v", errQuery)
			}
			if status != string(tt.seedStatus) {
				t.Errorf("status must be unchanged. expected: %s, got: %s", tt.seedStatus, status)
			}
		})
	}
}
