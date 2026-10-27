package dbhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/outboundconfig"
)

func Test_OutboundConfig_cov2_getFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2e00001-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&outboundconfig.OutboundConfig{})

	tests := []struct {
		name string
		qErr error
		want func(err error) bool
	}{
		{
			name: "query_error",
			qErr: errCov2Query,
			want: func(err error) bool { return errors.Is(err, errCov2Query) },
		},
		{
			name: "row_iteration_error",
			want: func(err error) bool { return errors.Is(err, errCov2Row) },
		},
		{
			name: "scan_error",
			want: func(err error) bool {
				return err != nil && !errors.Is(err, errCov2Query) && !errors.Is(err, errCov2Row)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCov2Env(t)

			exp := e.mock.ExpectQuery(cov2SelectRe(outboundConfigTable))
			switch tt.name {
			case "query_error":
				exp.WillReturnError(tt.qErr)
			case "row_iteration_error":
				exp.WillReturnRows(cov2RowErrRows(cols))
			default:
				exp.WillReturnRows(cov2ShortRows())
			}

			res, err := e.h.OutboundConfigGetByID(ctx, id)
			if !tt.want(err) {
				t.Errorf("Wrong match. got err: %v", err)
			}
			if res != nil {
				t.Errorf("expected no outbound config together with an error, got: %v", res)
			}
			if err := e.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_OutboundConfig_cov2_listFailures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c2e00002-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&outboundconfig.OutboundConfig{})

	// a non-empty page token adds the tm_create cursor condition.
	runCov2List(t, outboundConfigTable, cols, false, func(h *handler) error {
		res, err := h.OutboundConfigList(ctx, customerID, 10, "2026-01-02 03:04:05.000000")
		if err != nil && res != nil {
			t.Errorf("expected no result together with an error, got: %v", res)
		}
		return err
	})
}

func Test_OutboundConfig_cov2_listPageTokenArgument(t *testing.T) {
	customerID := uuid.FromStringOrNil("c2e00003-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&outboundconfig.OutboundConfig{})
	token := "2026-01-02 03:04:05.000000"

	e := newCov2Env(t)
	e.mock.ExpectQuery(cov2SelectRe(outboundConfigTable)+".*tm_create < \\?").
		WithArgs(customerID.Bytes(), token).
		WillReturnRows(sqlmock.NewRows(cols))

	res, err := e.h.OutboundConfigList(context.Background(), customerID, 10, token)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", res)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_OutboundConfig_cov2_writeFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2e00004-1b2d-11f1-8a5e-0b3f6c1d2e01")
	detail := "detail-cov2"
	codecs := "ulaw,alaw"

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "OutboundConfigCreate",
			stmtRe: cov2InsertRe(outboundConfigTable),
			run: func(h *handler) error {
				c := &outboundconfig.OutboundConfig{}
				c.ID = id
				return h.OutboundConfigCreate(ctx, c)
			},
		},
		{
			name:   "OutboundConfigDelete",
			stmtRe: cov2UpdateRe(outboundConfigTable),
			run: func(h *handler) error {
				return h.OutboundConfigDelete(ctx, id)
			},
		},
		{
			name:   "OutboundConfigUpdate",
			stmtRe: cov2UpdateRe(outboundConfigTable),
			run: func(h *handler) error {
				res, err := h.OutboundConfigUpdate(ctx, id, &outboundconfig.UpdateRequest{Detail: &detail, Codecs: &codecs})
				if res != nil {
					t.Errorf("expected no result together with an error, got: %v", res)
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}
