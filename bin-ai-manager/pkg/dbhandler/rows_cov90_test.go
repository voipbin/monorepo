package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

var (
	cov90ID      = uuid.FromStringOrNil("3b7e1a20-8c44-11f1-a0d3-5b2e9c4f7a10")
	cov90Other   = uuid.FromStringOrNil("3b7e1a21-8c44-11f1-a0d3-5b2e9c4f7a10")
	cov90Created = "2025-06-01 00:00:00.000000"
)

func cov90Exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("could not run the fixture statement. query: %s, err: %v", query, err)
	}
}

// cov90Insert inserts one row into the table with the given column values. Every
// other NOT NULL column gets a neutral filler of its declared type, so a test
// names only the columns that matter to it.
func cov90Insert(t *testing.T, db *sql.DB, table string, values map[string]any) {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("could not read the table definition. err: %v", err)
	}
	type column struct {
		name    string
		kind    string
		notNull bool
	}
	columns := []column{}
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, kind       string
			dflt             sql.NullString
		)
		if errScan := rows.Scan(&cid, &name, &kind, &notNull, &dflt, &pk); errScan != nil {
			_ = rows.Close()
			t.Fatalf("could not scan the table definition. err: %v", errScan)
		}
		columns = append(columns, column{name: name, kind: strings.ToLower(kind), notNull: notNull == 1 || pk > 0})
	}
	errRows := rows.Err()
	// the pool has a single connection, so rows must be closed before any other statement.
	_ = rows.Close()
	if errRows != nil {
		t.Fatalf("could not read the table definition. err: %v", errRows)
	}

	names := []string{}
	marks := []string{}
	args := []any{}
	for _, c := range columns {
		v, ok := values[c.name]
		if !ok {
			if !c.notNull {
				continue
			}
			switch {
			case strings.HasPrefix(c.kind, "binary"):
				v = make([]byte, 16)
			case strings.HasPrefix(c.kind, "datetime"):
				v = cov90Created
			case strings.HasPrefix(c.kind, "int"), strings.HasPrefix(c.kind, "bigint"), strings.HasPrefix(c.kind, "bool"):
				v = 0
			default:
				v = "x"
			}
		}
		names = append(names, c.name)
		marks = append(marks, "?")
		args = append(args, v)
	}

	cov90Exec(t, db, "INSERT INTO "+table+" ("+strings.Join(names, ", ")+") VALUES ("+strings.Join(marks, ", ")+")", args...)
}

// Test_GetFromDB_NotFound verifies that reading a missing row is reported as
// ErrNotFound for every cached resource and for the uncached ones.
func Test_GetFromDB_NotFound(t *testing.T) {
	h, _, mockCache := newCov90Handler(t, newPrivateDB(t))

	errMiss := errors.New("cache miss")
	mockCache.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().AIcallGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().AIcallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().SummaryGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().TeamGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()

	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"AIGet", func() error { _, err := h.AIGet(ctx, cov90ID); return err }},
		{"AIcallGet", func() error { _, err := h.AIcallGet(ctx, cov90ID); return err }},
		{"AIcallGetSkipCache", func() error { _, err := h.AIcallGetSkipCache(ctx, cov90ID); return err }},
		{"AIcallGetByReferenceID", func() error { _, err := h.AIcallGetByReferenceID(ctx, cov90ID); return err }},
		{"AIcallGetByCustomerIDAndReferenceID", func() error {
			_, err := h.AIcallGetByCustomerIDAndReferenceID(ctx, cov90ID, cov90Other)
			return err
		}},
		{"AIAuditGet", func() error { _, err := h.AIAuditGet(ctx, cov90ID); return err }},
		{"AIPromptHistoryGet", func() error { _, err := h.AIPromptHistoryGet(ctx, cov90ID); return err }},
		{"AIPromptProposalGet", func() error { _, err := h.AIPromptProposalGet(ctx, cov90ID); return err }},
		{"McpOAuthStateGet", func() error { _, err := h.McpOAuthStateGet(ctx, "missing-state"); return err }},
		{"McpServerGet", func() error { _, err := h.McpServerGet(ctx, cov90ID); return err }},
		{"MessageGet", func() error { _, err := h.MessageGet(ctx, cov90ID); return err }},
		{"SummaryGet", func() error { _, err := h.SummaryGet(ctx, cov90ID); return err }},
		{"TeamGet", func() error { _, err := h.TeamGet(ctx, cov90ID); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrNotFound) {
				t.Errorf("%s: expected ErrNotFound, got: %v", tt.name, err)
			}
		})
	}
}

// Test_Get_CacheHit verifies that a cached value is returned as is and that the
// database is not consulted (the handler runs on a closed database).
func Test_Get_CacheHit(t *testing.T) {
	h, _, mockCache := newCov90Handler(t, newClosedDB(t))
	ctx := context.Background()

	cachedAI := &ai.AI{Name: "cached-ai"}
	cachedAIcall := &aicall.AIcall{Status: aicall.StatusProgressing}
	cachedMessage := &message.Message{}
	cachedSummary := &summary.Summary{}
	cachedTeam := &team.Team{Name: "cached-team"}

	mockCache.EXPECT().AIGet(ctx, cov90ID).Return(cachedAI, nil)
	mockCache.EXPECT().AIcallGet(ctx, cov90ID).Return(cachedAIcall, nil)
	mockCache.EXPECT().AIcallGetByReferenceID(ctx, cov90Other).Return(cachedAIcall, nil)
	mockCache.EXPECT().MessageGet(ctx, cov90ID).Return(cachedMessage, nil)
	mockCache.EXPECT().SummaryGet(ctx, cov90ID).Return(cachedSummary, nil)
	mockCache.EXPECT().TeamGet(ctx, cov90ID).Return(cachedTeam, nil)

	if res, err := h.AIGet(ctx, cov90ID); err != nil || res != cachedAI {
		t.Errorf("AIGet: expected the cached ai, got: %v, err: %v", res, err)
	}
	if res, err := h.AIcallGet(ctx, cov90ID); err != nil || res != cachedAIcall {
		t.Errorf("AIcallGet: expected the cached aicall, got: %v, err: %v", res, err)
	}
	if res, err := h.AIcallGetByReferenceID(ctx, cov90Other); err != nil || res != cachedAIcall {
		t.Errorf("AIcallGetByReferenceID: expected the cached aicall, got: %v, err: %v", res, err)
	}
	if res, err := h.MessageGet(ctx, cov90ID); err != nil || res != cachedMessage {
		t.Errorf("MessageGet: expected the cached message, got: %v, err: %v", res, err)
	}
	if res, err := h.SummaryGet(ctx, cov90ID); err != nil || res != cachedSummary {
		t.Errorf("SummaryGet: expected the cached summary, got: %v, err: %v", res, err)
	}
	if res, err := h.TeamGet(ctx, cov90ID); err != nil || res != cachedTeam {
		t.Errorf("TeamGet: expected the cached team, got: %v, err: %v", res, err)
	}
}

// Test_UpdateToCache_Errors verifies the two failure points of the cache
// refresh: the row cannot be read (it does not exist) and the cache refuses
// the write.
func Test_UpdateToCache_Errors(t *testing.T) {
	errCache := errors.New("cache write refused")
	ctx := context.Background()

	tests := []struct {
		name   string
		table  string
		update func(h *handler) error
		set    func(h *handler) error
		expect func(mockCache *cachehandler.MockCacheHandler)
	}{
		{
			name:   "ai",
			table:  "ai_ais",
			update: func(h *handler) error { return h.aiUpdateToCache(ctx, cov90ID) },
			set:    func(h *handler) error { return h.aiSetToCache(ctx, &ai.AI{}) },
			expect: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().AISet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			},
		},
		{
			name:   "aicall",
			table:  "ai_aicalls",
			update: func(h *handler) error { return h.aicallUpdateToCache(ctx, cov90ID) },
			set:    func(h *handler) error { return h.aicallSetToCache(ctx, &aicall.AIcall{}) },
			expect: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().AIcallSet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			},
		},
		{
			name:   "message",
			table:  "ai_messages",
			update: func(h *handler) error { return h.messageUpdateToCache(ctx, cov90ID) },
			set:    func(h *handler) error { return h.messageSetToCache(ctx, &message.Message{}) },
			expect: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			},
		},
		{
			name:   "summary",
			table:  "ai_summaries",
			update: func(h *handler) error { return h.summaryUpdateToCache(ctx, cov90ID) },
			set:    func(h *handler) error { return h.summarySetToCache(ctx, &summary.Summary{}) },
			expect: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().SummarySet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			},
		},
		{
			name:   "team",
			table:  "ai_teams",
			update: func(h *handler) error { return h.teamUpdateToCache(ctx, cov90ID) },
			set:    func(h *handler) error { return h.teamSetToCache(ctx, &team.Team{}) },
			expect: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().TeamSet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newPrivateDB(t)
			h, _, mockCache := newCov90Handler(t, db)
			tt.expect(mockCache)

			// the row does not exist yet
			if err := tt.update(h); !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound for a missing row, got: %v", err)
			}

			// the row exists, the cache refuses the write
			cov90Insert(t, db, tt.table, map[string]any{"id": cov90ID.Bytes(), "customer_id": cov90Other.Bytes()})
			if err := tt.update(h); !errors.Is(err, errCache) {
				t.Errorf("expected the cache error from the refresh, got: %v", err)
			}

			if err := tt.set(h); !errors.Is(err, errCache) {
				t.Errorf("expected the cache error from the set, got: %v", err)
			}
		})
	}
}

// Test_Scan_Errors verifies that a row that cannot be decoded (a malformed uuid
// column) is returned as an error by the single reads and the list reads, and
// is neither skipped nor reported as ErrNotFound.
func Test_Scan_Errors(t *testing.T) {
	bad := []byte{0x01, 0x02}
	ctx := context.Background()

	tests := []struct {
		name   string
		table  string
		values map[string]any
		fn     func(h *handler) error
	}{
		{"ai get", "ai_ais", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.aiGetFromDB(ctx, cov90ID); return err }},
		{"ai list", "ai_ais", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIList(ctx, 10, "", nil); return err }},
		{"aicall get", "ai_aicalls", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.aicallGetFromDB(ctx, cov90ID); return err }},
		{"aicall get by reference id", "ai_aicalls", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "reference_id": cov90Other.Bytes(), "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIcallGetByReferenceID(ctx, cov90Other); return err }},
		{"aicall get by customer id and reference id", "ai_aicalls", map[string]any{"id": cov90ID.Bytes(), "customer_id": cov90Other.Bytes(), "reference_id": cov90ID.Bytes(), "activeflow_id": bad, "tm_create": cov90Created},
			func(h *handler) error {
				_, err := h.AIcallGetByCustomerIDAndReferenceID(ctx, cov90Other, cov90ID)
				return err
			}},
		{"aicall list", "ai_aicalls", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIcallList(ctx, 10, "", nil); return err }},
		{"aiaudit get", "ai_ai_audits", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.aiauditGetFromDB(ctx, cov90ID); return err }},
		{"aiaudit list", "ai_ai_audits", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIAuditList(ctx, 10, "", nil); return err }},
		{"aiprompthistory get", "ai_ai_prompt_histories", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.aiPromptHistoryGetFromDB(ctx, cov90ID); return err }},
		{"aiprompthistory list", "ai_ai_prompt_histories", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "ai_id": cov90Other.Bytes(), "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIPromptHistoryGetsByAIID(ctx, cov90Other, 10, ""); return err }},
		{"aipromptproposal get", "ai_ai_prompt_proposals", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.aipromptproposalGetFromDB(ctx, cov90ID); return err }},
		{"aipromptproposal list", "ai_ai_prompt_proposals", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.AIPromptProposalList(ctx, 10, "", nil); return err }},
		{"mcpoauthstate get", "ai_mcp_oauth_states", map[string]any{"state": "scan-state", "customer_id": bad},
			func(h *handler) error { _, err := h.McpOAuthStateGet(ctx, "scan-state"); return err }},
		{"mcpserver get", "ai_mcp_servers", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.mcpserverGetFromDB(ctx, cov90ID); return err }},
		{"mcpserver list", "ai_mcp_servers", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.McpServerList(ctx, 10, "", nil); return err }},
		{"message get", "ai_messages", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.messageGetFromDB(ctx, cov90ID); return err }},
		{"message list", "ai_messages", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.MessageList(ctx, 10, "", nil); return err }},
		{"participant list by aicall id", "ai_aicall_participants", map[string]any{"ai_id": bad, "aicall_id": cov90ID.Bytes(), "tm_create": cov90Created},
			func(h *handler) error { _, err := h.ParticipantListByAIcallID(ctx, cov90ID, 10, ""); return err }},
		{"participant list by ai id", "ai_aicall_participants", map[string]any{"ai_id": cov90ID.Bytes(), "aicall_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.ParticipantListByAIID(ctx, cov90ID, 10, ""); return err }},
		{"summary get", "ai_summaries", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.summaryGetFromDB(ctx, cov90ID); return err }},
		{"summary list", "ai_summaries", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.SummaryList(ctx, 10, "", nil); return err }},
		{"team get", "ai_teams", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.teamGetFromDB(ctx, cov90ID); return err }},
		{"team list", "ai_teams", map[string]any{"id": cov90ID.Bytes(), "customer_id": bad, "tm_create": cov90Created},
			func(h *handler) error { _, err := h.TeamList(ctx, 10, "", nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newPrivateDB(t)
			h, _, mockCache := newCov90Handler(t, db)
			mockCache.EXPECT().AIcallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()
			cov90Insert(t, db, tt.table, tt.values)

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected a scan error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a scan failure must not be reported as ErrNotFound. err: %v", err)
			}
		})
	}
}

// Test_McpServerUpdateOAuthTokensIfCurrent_RequiresSpentToken verifies that the
// compare-and-swap update refuses to run without the spent refresh token.
func Test_McpServerUpdateOAuthTokensIfCurrent_RequiresSpentToken(t *testing.T) {
	h, _, _ := newCov90Handler(t, newPrivateDB(t))

	fields := map[mcpserver.Field]any{mcpserver.FieldName: "renamed"}
	err := h.McpServerUpdateOAuthTokensIfCurrent(context.Background(), cov90ID, nil, fields)
	if err == nil || !strings.Contains(err.Error(), "spent refresh token is required") {
		t.Fatalf("expected the missing spent token to be refused, got: %v", err)
	}
}

// Test_AIAuditUpdateFinal_WithEvaluation verifies that a completed audit with an
// evaluation document is written to the row.
func Test_AIAuditUpdateFinal_WithEvaluation(t *testing.T) {
	db := newPrivateDB(t)
	h, _, _ := newCov90Handler(t, db)
	cov90Insert(t, db, "ai_ai_audits", map[string]any{"id": cov90ID.Bytes(), "customer_id": cov90Other.Bytes(), "status": string(aiaudit.StatusProgressing)})

	score := 7
	evaluation := json.RawMessage(`{"summary":"ok"}`)
	n, err := h.AIAuditUpdateFinal(context.Background(), cov90ID, aiaudit.StatusCompleted, &score, evaluation, "", []uuid.UUID{cov90Other})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected one updated row, got: %d", n)
	}

	var gotEvaluation, gotStatus string
	if errScan := db.QueryRow("SELECT evaluation, status FROM ai_ai_audits WHERE id = ?", cov90ID.Bytes()).Scan(&gotEvaluation, &gotStatus); errScan != nil {
		t.Fatalf("could not read the audit row: %v", errScan)
	}
	if gotEvaluation != string(evaluation) || gotStatus != string(aiaudit.StatusCompleted) {
		t.Errorf("unexpected stored row. evaluation: %s, status: %s", gotEvaluation, gotStatus)
	}
}

// Test_AIActivateInsight_MalformedCustomerID verifies that an insight AI whose
// stored customer id is not a uuid is refused before any lock is taken.
func Test_AIActivateInsight_MalformedCustomerID(t *testing.T) {
	db := newPrivateDB(t)
	h, _, _ := newCov90Handler(t, db)
	cov90Insert(t, db, "ai_ais", map[string]any{"id": cov90ID.Bytes(), "customer_id": []byte{0x01, 0x02}, "type": string(ai.TypeInsight)})

	res, previous, err := h.AIActivateInsight(context.Background(), cov90ID)
	if err == nil {
		t.Fatalf("expected an error for a malformed customer id, got nil")
	}
	if res != nil || previous != nil {
		t.Errorf("expected no ai to be returned, got: %v, %v", res, previous)
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrNotInsightAI) {
		t.Errorf("a malformed customer id must not be reported as a missing or wrong-type ai. err: %v", err)
	}
}

// Test_McpServerList_ReturnsRows verifies that listed servers carry HasSecret
// derived from the stored ciphertext columns.
func Test_McpServerList_ReturnsRows(t *testing.T) {
	db := newPrivateDB(t)
	h, _, _ := newCov90Handler(t, db)

	withSecret := uuid.FromStringOrNil("3b7e1a30-8c44-11f1-a0d3-5b2e9c4f7a10")
	withoutSecret := uuid.FromStringOrNil("3b7e1a31-8c44-11f1-a0d3-5b2e9c4f7a10")
	cov90Insert(t, db, "ai_mcp_servers", map[string]any{"id": withSecret.Bytes(), "customer_id": cov90Other.Bytes(), "secret_ciphertext": []byte("cipher"), "tm_create": "2025-06-02 00:00:00.000000"})
	cov90Insert(t, db, "ai_mcp_servers", map[string]any{"id": withoutSecret.Bytes(), "customer_id": cov90Other.Bytes(), "tm_create": "2025-06-01 00:00:00.000000"})

	res, err := h.McpServerList(context.Background(), 10, "", map[mcpserver.Field]any{mcpserver.FieldCustomerID: cov90Other})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 servers, got: %d", len(res))
	}
	// newest first
	if res[0].ID != withSecret || !res[0].HasSecret {
		t.Errorf("expected the server with a secret first and HasSecret set, got: %v %v", res[0].ID, res[0].HasSecret)
	}
	if res[1].ID != withoutSecret || res[1].HasSecret {
		t.Errorf("expected the server without a secret second and HasSecret unset, got: %v %v", res[1].ID, res[1].HasSecret)
	}
}
