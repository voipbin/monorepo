package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

// covSQLCase describes one sqlmock-driven statement of a MySQL JSON function
// site. SQLite (the unit-test DB) implements none of these functions, so the
// executed SQL text and the bound arguments are asserted instead.
type covSQLCase struct {
	name string
	do   func(h *handler) error

	// wantSQL is the literal statement the handler must execute.
	wantSQL  string
	wantArgs []driver.Value

	// cacheSelect is the regexp of the follow-up SELECT that refreshes the cache.
	cacheSelect string
}

// runCovSQLCases runs the given cases twice with the newTXTestHandler pattern:
// once with a succeeding statement (the cache refresh must follow) and once with a
// failing statement (an error must be returned and no cache refresh may follow).
func runCovSQLCases(t *testing.T, cases []covSQLCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name+"_ok", func(t *testing.T) {
			h, mock, _ := newTXTestHandler(t)

			mock.ExpectExec("^" + regexp.QuoteMeta(tt.wantSQL) + "$").
				WithArgs(tt.wantArgs...).
				WillReturnResult(driver.RowsAffected(1))
			// the cache refresh reads the row back; a failing read is tolerated by the caller.
			mock.ExpectQuery(tt.cacheSelect).WillReturnError(errors.New("simulated read failure"))

			if err := tt.do(h); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})

		t.Run(tt.name+"_exec_error", func(t *testing.T) {
			h, mock, _ := newTXTestHandler(t)

			mock.ExpectExec("^" + regexp.QuoteMeta(tt.wantSQL) + "$").
				WithArgs(tt.wantArgs...).
				WillReturnError(errors.New("simulated exec failure"))

			if err := tt.do(h); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_CallJSONFunctions_cov(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("b1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01")
	otherID := uuid.FromStringOrNil("b1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02")
	sel := `SELECT .* FROM call_calls WHERE id = \?`

	// newTXTestHandler's clock returns a nil *time.Time, bound as NULL.
	runCovSQLCases(t, []covSQLCase{
		{
			name: "CallAddChainedCallID",
			do:   func(h *handler) error { return h.CallAddChainedCallID(ctx, id, otherID) },
			wantSQL: "UPDATE call_calls SET chained_call_ids = json_array_append(chained_call_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "CallRemoveChainedCallID",
			do:   func(h *handler) error { return h.CallRemoveChainedCallID(ctx, id, otherID) },
			wantSQL: "UPDATE call_calls SET chained_call_ids = json_remove(chained_call_ids, " +
				"replace(json_search(chained_call_ids, 'one', ?), '\"', '')), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "CallAddExternalMediaID",
			do:   func(h *handler) error { return h.CallAddExternalMediaID(ctx, id, otherID) },
			wantSQL: "UPDATE call_calls SET external_media_ids = json_array_append(external_media_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "CallRemoveExternalMediaID",
			do:   func(h *handler) error { return h.CallRemoveExternalMediaID(ctx, id, otherID) },
			wantSQL: "UPDATE call_calls SET external_media_ids = IF(json_search(external_media_ids, 'one', ?) IS NOT NULL, " +
				"json_remove(external_media_ids, replace(json_search(external_media_ids, 'one', ?), '\"', '')), " +
				"external_media_ids), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
		{
			name: "CallAddRecordingIDs",
			do:   func(h *handler) error { return h.CallAddRecordingIDs(ctx, id, otherID) },
			wantSQL: "UPDATE call_calls SET recording_ids = json_array_append(recording_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{otherID.String(), nil, id.Bytes()},
			cacheSelect: sel,
		},
	})
}

func Test_CallSetStatusRinging_cov(t *testing.T) {
	tm := testhelper.TimePtr("2020-04-18T03:22:17.995000Z")
	c := &call.Call{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("c1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01")},
		ChannelID: "cov-channel-ringing",
		Type:      call.TypeFlow,
		Status:    call.StatusDialing,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(tm)
	mockCache.EXPECT().CallSet(ctx, gomock.Cond(func(x *call.Call) bool { return x != nil && x.ID == c.ID }))
	if err := h.CallCreate(ctx, c); err != nil {
		t.Fatalf("could not create the call. err: %v", err)
	}

	later := testhelper.TimePtr("2020-04-18T03:22:19.995000Z")
	mockUtil.EXPECT().TimeNow().Return(later)
	mockCache.EXPECT().CallSet(ctx, gomock.Cond(func(x *call.Call) bool { return x != nil && x.ID == c.ID }))
	if err := h.CallSetStatusRinging(ctx, c.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.callGetFromDB(ctx, c.ID)
	if err != nil {
		t.Fatalf("could not get the call. err: %v", err)
	}
	if res.Status != call.StatusRinging {
		t.Errorf("Wrong status. expect: %v, got: %v", call.StatusRinging, res.Status)
	}
	if res.TMRinging == nil || !res.TMRinging.Equal(*later) || res.TMUpdate == nil || !res.TMUpdate.Equal(*later) {
		t.Errorf("Wrong timestamps. tm_ringing: %v, tm_update: %v", res.TMRinging, res.TMUpdate)
	}
	if res.TMProgressing != nil {
		t.Errorf("tm_progressing must stay unset, got: %v", res.TMProgressing)
	}
}

func Test_CallSetStatusProgressing_cov(t *testing.T) {
	tm := testhelper.TimePtr("2020-04-18T03:22:17.995000Z")
	c := &call.Call{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("c1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02")},
		ChannelID: "cov-channel-progressing",
		Type:      call.TypeFlow,
		Status:    call.StatusRinging,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(tm)
	mockCache.EXPECT().CallSet(ctx, gomock.Cond(func(x *call.Call) bool { return x != nil && x.ID == c.ID }))
	if err := h.CallCreate(ctx, c); err != nil {
		t.Fatalf("could not create the call. err: %v", err)
	}

	later := testhelper.TimePtr("2020-04-18T03:22:19.995000Z")
	mockUtil.EXPECT().TimeNow().Return(later)
	mockCache.EXPECT().CallSet(ctx, gomock.Cond(func(x *call.Call) bool { return x != nil && x.ID == c.ID }))
	if err := h.CallSetStatusProgressing(ctx, c.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.callGetFromDB(ctx, c.ID)
	if err != nil {
		t.Fatalf("could not get the call. err: %v", err)
	}
	if res.Status != call.StatusProgressing {
		t.Errorf("Wrong status. expect: %v, got: %v", call.StatusProgressing, res.Status)
	}
	if res.TMProgressing == nil || !res.TMProgressing.Equal(*later) || res.TMUpdate == nil || !res.TMUpdate.Equal(*later) {
		t.Errorf("Wrong timestamps. tm_progressing: %v, tm_update: %v", res.TMProgressing, res.TMUpdate)
	}
	if res.TMRinging != nil {
		t.Errorf("tm_ringing must stay unset, got: %v", res.TMRinging)
	}
}
