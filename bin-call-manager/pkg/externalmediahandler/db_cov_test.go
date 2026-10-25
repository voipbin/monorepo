package externalmediahandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	stderrors "errors"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/errors"
)

func Test_Create_generatesID_cov(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	genID := uuid.FromStringOrNil("44444444-0001-11f0-8000-000000000001")

	m.util.EXPECT().UUIDCreate().Return(genID)
	m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)

	res, err := h.Create(ctx, uuid.Nil, "ast-1", "ch-1", "br-1", "", externalmedia.ReferenceTypeCall, uuid.Nil, "", 0, "ext:1", defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat, externalmedia.DirectionBoth, externalmedia.DirectionBoth, "")
	if err != nil || res.ID != genID {
		t.Errorf("Wrong match. got: (%v, %v)", res, err)
	}
}

func Test_Get_errors_cov(t *testing.T) {
	id := uuid.FromStringOrNil("44444444-0002-11f0-8000-000000000002")

	tests := []struct {
		name      string
		dbErr     error
		expectNF  bool
		expectErr bool
	}{
		{name: "normal"},
		{name: "not found is typed", dbErr: dbhandler.ErrNotFound, expectNF: true, expectErr: true},
		{name: "other error", dbErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			if tt.dbErr != nil {
				m.db.EXPECT().ExternalMediaGet(ctx, id).Return(nil, tt.dbErr)
			} else {
				m.db.EXPECT().ExternalMediaGet(ctx, id).Return(&externalmedia.ExternalMedia{ID: id}, nil)
			}

			res, err := h.Get(ctx, id)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res.ID != id {
				t.Errorf("Wrong match. got: %v", res)
			}
			if tt.expectErr {
				var ve *errors.VoipbinError
				if got := stderrors.As(err, &ve); got != tt.expectNF {
					t.Errorf("Wrong typed error. expect: %v, got: %v", tt.expectNF, got)
				}
			}
		})
	}
}

func Test_List_cov(t *testing.T) {
	refID := uuid.FromStringOrNil("44444444-0003-11f0-8000-000000000003")

	tests := []struct {
		name    string
		filters map[externalmedia.Field]any
		dbErr   error
		callDB  bool
		expect  int
	}{
		{name: "no filter", filters: map[externalmedia.Field]any{}, expect: 0},
		{name: "nil reference id", filters: map[externalmedia.Field]any{externalmedia.FieldReferenceID: nil}, expect: 0},
		{name: "uuid reference id", filters: map[externalmedia.Field]any{externalmedia.FieldReferenceID: refID}, callDB: true, expect: 1},
		{name: "string reference id", filters: map[externalmedia.Field]any{externalmedia.FieldReferenceID: refID.String()}, callDB: true, expect: 1},
		{name: "not found gives an empty list", filters: map[externalmedia.Field]any{externalmedia.FieldReferenceID: refID}, callDB: true, dbErr: errCov, expect: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			if tt.callDB {
				if tt.dbErr != nil {
					m.db.EXPECT().ExternalMediaGetByReferenceID(ctx, refID).Return(nil, tt.dbErr)
				} else {
					m.db.EXPECT().ExternalMediaGetByReferenceID(ctx, refID).Return(&externalmedia.ExternalMedia{ID: refID}, nil)
				}
			}

			res, err := h.List(ctx, 10, "", tt.filters)
			if err != nil || len(res) != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: (%v, %v)", tt.expect, res, err)
			}
		})
	}
}

func Test_UpdateLocalAddress_UpdateStatus_cov(t *testing.T) {
	id := uuid.FromStringOrNil("44444444-0004-11f0-8000-000000000004")

	// failAt: 0 = none, 1 = first get, 2 = set, 3 = second get
	steps := []struct {
		name   string
		failAt int
	}{
		{name: "normal", failAt: 0},
		{name: "first get error", failAt: 1},
		{name: "set error", failAt: 2},
		{name: "second get error", failAt: 3},
	}

	calls := []struct {
		name string
		call func(h *externalMediaHandler, ctx context.Context) (*externalmedia.ExternalMedia, error)
	}{
		{"UpdateLocalAddress", func(h *externalMediaHandler, ctx context.Context) (*externalmedia.ExternalMedia, error) {
			return h.UpdateLocalAddress(ctx, id, "10.0.0.1", 5000)
		}},
		{"UpdateStatus", func(h *externalMediaHandler, ctx context.Context) (*externalmedia.ExternalMedia, error) {
			return h.UpdateStatus(ctx, id, externalmedia.StatusTerminating)
		}},
	}

	for _, c := range calls {
		for _, s := range steps {
			t.Run(c.name+"/"+s.name, func(t *testing.T) {
				h, m := newCovHandler(t)
				ctx := context.Background()

				if s.failAt == 1 {
					m.db.EXPECT().ExternalMediaGet(ctx, id).Return(nil, errCov)
				} else {
					m.db.EXPECT().ExternalMediaGet(ctx, id).Return(&externalmedia.ExternalMedia{ID: id}, nil)
					if s.failAt == 2 {
						m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(errCov)
					} else {
						m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
						if s.failAt == 3 {
							m.db.EXPECT().ExternalMediaGet(ctx, id).Return(nil, errCov)
						} else {
							m.db.EXPECT().ExternalMediaGet(ctx, id).Return(&externalmedia.ExternalMedia{ID: id}, nil)
						}
					}
				}

				res, err := c.call(h, ctx)
				if (err != nil) != (s.failAt != 0) {
					t.Fatalf("Wrong match. failAt: %d, got: (%v, %v)", s.failAt, res, err)
				}
			})
		}
	}
}
