package analysishandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
)

func getTestIDs() (uuid.UUID, uuid.UUID) {
	return uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"), uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
}

// getCall is one of the three read/delete methods that mask a db not-found.
type getCall struct {
	name   string
	expect func(dbMock *analysisdbhandler.MockAnalysisDBHandler, ctx context.Context, custID, id uuid.UUID) *gomock.Call
	call   func(h *analysisHandler, ctx context.Context, custID, id uuid.UUID) (*analysis.Analysis, error)
}

func getCalls() []getCall {
	return []getCall{
		{
			name: "Get",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler, ctx context.Context, custID, id uuid.UUID) *gomock.Call {
				return dbMock.EXPECT().AnalysisGet(ctx, custID, id)
			},
			call: func(h *analysisHandler, ctx context.Context, custID, id uuid.UUID) (*analysis.Analysis, error) {
				return h.Get(ctx, custID, id)
			},
		},
		{
			name: "GetByActiveflowID",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler, ctx context.Context, custID, id uuid.UUID) *gomock.Call {
				return dbMock.EXPECT().AnalysisGetByActiveflowID(ctx, custID, id)
			},
			call: func(h *analysisHandler, ctx context.Context, custID, id uuid.UUID) (*analysis.Analysis, error) {
				return h.GetByActiveflowID(ctx, custID, id)
			},
		},
		{
			name: "Delete",
			expect: func(dbMock *analysisdbhandler.MockAnalysisDBHandler, ctx context.Context, custID, id uuid.UUID) *gomock.Call {
				return dbMock.EXPECT().AnalysisDelete(ctx, custID, id)
			},
			call: func(h *analysisHandler, ctx context.Context, custID, id uuid.UUID) (*analysis.Analysis, error) {
				return h.Delete(ctx, custID, id)
			},
		},
	}
}

func Test_analysis_get_delete(t *testing.T) {
	for _, gc := range getCalls() {
		t.Run(gc.name+"/success returns the db record", func(t *testing.T) {
			h, _, dbMock, mc := newStartTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			custID, id := getTestIDs()

			rec := &analysis.Analysis{Status: analysis.StatusCompleted}
			gc.expect(dbMock, ctx, custID, id).Return(rec, nil)

			got, err := gc.call(h, ctx, custID, id)
			if err != nil || got != rec {
				t.Errorf("Wrong match. expect: the db record, got: %v err=%v", got, err)
			}
		})

		t.Run(gc.name+"/db not-found is masked", func(t *testing.T) {
			h, _, dbMock, mc := newStartTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			custID, id := getTestIDs()

			gc.expect(dbMock, ctx, custID, id).Return(nil, analysisdbhandler.ErrNotFound)

			got, err := gc.call(h, ctx, custID, id)
			if err != ErrNotFound {
				t.Errorf("Wrong match. expect: the masked ErrNotFound, got: %v", err)
			}
			if errors.Is(err, analysisdbhandler.ErrNotFound) || strings.Contains(err.Error(), "record") {
				t.Errorf("Wrong match. the db error must not be exposed, got: %v", err)
			}
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})

		t.Run(gc.name+"/other errors are wrapped", func(t *testing.T) {
			h, _, dbMock, mc := newStartTestHandler(t)
			defer mc.Finish()
			ctx := context.Background()
			custID, id := getTestIDs()
			boom := errors.New("db failure")

			gc.expect(dbMock, ctx, custID, id).Return(nil, boom)

			got, err := gc.call(h, ctx, custID, id)
			if !errors.Is(err, boom) || err == ErrNotFound || !strings.Contains(err.Error(), gc.name+": could not") {
				t.Errorf("Wrong match. expect: wrapped db failure, got: %v", err)
			}
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})
	}
}

func Test_analysis_List(t *testing.T) {
	custID, _ := getTestIDs()
	filters := map[analysis.Field]any{analysis.FieldStatus: analysis.StatusCompleted}

	t.Run("passes arguments in the db order", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()
		ctx := context.Background()

		rows := []*analysis.Analysis{{Status: analysis.StatusCompleted}}
		dbMock.EXPECT().AnalysisList(ctx, custID, uint64(7), "tok-A", filters).Return(rows, nil)

		got, err := h.List(ctx, custID, "tok-A", 7, filters)
		if err != nil || !reflect.DeepEqual(got, rows) {
			t.Errorf("Wrong match. expect: the db rows, got: %v err=%v", got, err)
		}
	})

	t.Run("a db not-found is wrapped, not masked", func(t *testing.T) {
		h, _, dbMock, mc := newStartTestHandler(t)
		defer mc.Finish()
		ctx := context.Background()

		dbMock.EXPECT().AnalysisList(ctx, custID, uint64(7), "tok-A", filters).Return(nil, analysisdbhandler.ErrNotFound)

		got, err := h.List(ctx, custID, "tok-A", 7, filters)
		if err == ErrNotFound || !errors.Is(err, analysisdbhandler.ErrNotFound) || !strings.Contains(err.Error(), "List: could not list analyses") {
			t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
		}
		if got != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", got)
		}
	})
}
