package analysishandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
)

func Test_Start_activeflow_lookup_error_masked(t *testing.T) {
	h, reqMock, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	// a lookup failure must look the same as a foreign activeflow: no db access.
	reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(nil, errors.New("upstream failure"))

	res, err := h.Start(context.Background(), cust, afID, false)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %+v", res)
	}
}

func Test_Start_db_errors(t *testing.T) {
	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		expect func(dbMock *analysisdbhandler.MockAnalysisDBHandler)
	}{
		{
			name: "existing lookup fails unexpectedly",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, injected)
			},
		},
		{
			name: "progressing count fails",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, analysisdbhandler.ErrNotFound)
				dbMock.EXPECT().AnalysisCountProgressing(gomock.Any(), cust).Return(int64(0), injected)
			},
		},
		{
			name: "create fails unexpectedly",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, analysisdbhandler.ErrNotFound)
				dbMock.EXPECT().AnalysisCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
				dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).Return(injected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, dbMock, mc := newStartTestHandler(t)
			defer mc.Finish()

			reqMock.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(endedActiveflow(afID, cust), nil)
			tt.expect(dbMock)

			res, err := h.Start(context.Background(), cust, afID, false)
			if !errors.Is(err, injected) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", injected, err)
			}
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrConcurrencyLimit) {
				t.Errorf("Wrong match. expect: unclassified error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %+v", res)
			}
		})
	}
}

func Test_createProgressing(t *testing.T) {
	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	injected := errors.New("injected failure")

	t.Run("created", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()

		var stored *analysis.Analysis
		dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, a *analysis.Analysis) error {
				stored = a
				return nil
			})

		res, err := h.createProgressing(context.Background(), cust, afID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !res.justCreated {
			t.Errorf("Wrong match. expect: justCreated true, got: false")
		}
		if res.row != stored {
			t.Errorf("Wrong match. expect: the stored row to be returned")
		}
		if stored.CustomerID != cust || stored.ActiveflowID != afID || stored.Status != analysis.StatusProgressing {
			t.Errorf("Wrong match. expect: progressing row for %s/%s, got: %+v", cust, afID, stored)
		}
		if stored.ID == uuid.Nil {
			t.Errorf("Wrong match. expect: a generated id, got: nil uuid")
		}
	})

	t.Run("duplicate then re-read returns the in-flight row", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()

		inflight := &analysis.Analysis{Status: analysis.StatusProgressing}
		inflight.Identity = commonidentity.Identity{ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"), CustomerID: cust}
		inflight.ActiveflowID = afID

		dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).Return(analysisdbhandler.ErrDuplicate)
		dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(inflight, nil)

		res, err := h.createProgressing(context.Background(), cust, afID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		// the concurrent row is returned and this caller must not start a second chain.
		if res.row != inflight || res.justCreated {
			t.Errorf("Wrong match. expect: in-flight row with justCreated false, got: %+v", res)
		}
	})

	t.Run("duplicate then re-read fails", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()

		dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).Return(analysisdbhandler.ErrDuplicate)
		dbMock.EXPECT().AnalysisGetByActiveflowID(gomock.Any(), cust, afID).Return(nil, injected)

		res, err := h.createProgressing(context.Background(), cust, afID)
		if !errors.Is(err, injected) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", injected, err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %+v", res)
		}
	})

	t.Run("create fails unexpectedly", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()

		dbMock.EXPECT().AnalysisCreate(gomock.Any(), gomock.Any()).Return(injected)

		res, err := h.createProgressing(context.Background(), cust, afID)
		if !errors.Is(err, injected) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", injected, err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %+v", res)
		}
	})
}

func Test_applyExistingPolicy_errors(t *testing.T) {
	cust := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		expect func(dbMock *analysisdbhandler.MockAnalysisDBHandler)
	}{
		{
			name: "reset fails",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisReset(gomock.Any(), id).Return(int64(0), injected)
			},
		},
		{
			name: "re-read fails after a lost reset",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisReset(gomock.Any(), id).Return(int64(0), nil)
				dbMock.EXPECT().AnalysisGet(gomock.Any(), cust, id).Return(nil, injected)
			},
		},
		{
			name: "re-read fails after a won reset",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler) {
				dbMock.EXPECT().AnalysisReset(gomock.Any(), id).Return(int64(1), nil)
				dbMock.EXPECT().AnalysisGet(gomock.Any(), cust, id).Return(nil, injected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, dbMock, mc := newStartTestHandler(t)
			defer mc.Finish()

			// TMUpdate is nil, so the cooldown gate (and the clock) is not used.
			existing := &analysis.Analysis{Status: analysis.StatusFailed}
			existing.Identity = commonidentity.Identity{ID: id, CustomerID: cust}
			tt.expect(dbMock)

			res, handled, err := h.applyExistingPolicy(context.Background(), existing, true)
			if !errors.Is(err, injected) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", injected, err)
			}
			if !handled {
				t.Errorf("Wrong match. expect: handled true (no chain start), got: false")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %+v", res)
			}
		})
	}
}

func Test_applyExistingPolicy_unknown_status(t *testing.T) {
	h, _, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	// an unrecognized status is treated as in-flight: returned as is, no db call, no chain.
	existing := &analysis.Analysis{Status: analysis.Status("unrecognized")}
	existing.Identity = commonidentity.Identity{ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")}

	res, handled, err := h.applyExistingPolicy(context.Background(), existing, true)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !handled || res != existing {
		t.Errorf("Wrong match. expect: existing row handled, got: handled=%v res=%+v", handled, res)
	}
}
