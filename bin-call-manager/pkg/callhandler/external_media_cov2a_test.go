package callhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/externalmedia"
)

func Test_ExternalMediaStart_cov2a(t *testing.T) {
	emID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000030")

	expectStart := func(m *cov2aMocks, c *call.Call) *gomock.Call {
		return m.extMedia.EXPECT().Start(gomock.Any(), emID, externalmedia.ReferenceTypeCall, c.ID, "", externalmedia.Encapsulation(""), externalmedia.Transport(""), "", "", "", externalmedia.Direction(""), externalmedia.Direction(""))
	}

	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "maximum number of external medias reached",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.ExternalMediaIDs = make([]uuid.UUID, defaultMaxExternalMediaPerCall)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
			},
			wantErr: "maximum number of external medias",
		},
		{
			name: "external media start error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				expectStart(m, c).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "external media id add error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				expectStart(m, c).Return(&externalmedia.ExternalMedia{ID: emID}, nil)
				m.db.EXPECT().CallAddExternalMediaID(gomock.Any(), c.ID, emID).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			got, err := h.ExternalMediaStart(context.Background(), c.ID, emID, "", "", "", "", "", "", "", "")
			cov2aCheckErr(t, err, tt.wantErr)
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})
	}
}

func Test_ExternalMediaStop_cov2a(t *testing.T) {
	emID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000030")

	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "external media id is not on the call",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
			},
			wantErr: "not associated with this call",
		},
		{
			name: "external media stop error",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.ExternalMediaIDs = []uuid.UUID{emID}
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.extMedia.EXPECT().Stop(gomock.Any(), emID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "refetch of the updated call fails",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.ExternalMediaIDs = []uuid.UUID{emID}
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.extMedia.EXPECT().Stop(gomock.Any(), emID).Return(&externalmedia.ExternalMedia{ID: emID}, nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom),
				)
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			got, err := h.ExternalMediaStop(context.Background(), c.ID, emID)
			cov2aCheckErr(t, err, tt.wantErr)
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})
	}
}

func Test_ExternalMediaStop_cov2a_success(t *testing.T) {
	h, m := cov2aNewHandler(t)
	ctx := context.Background()

	emID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000030")
	c := cov2aNewCall(nil)
	c.ExternalMediaIDs = []uuid.UUID{emID}
	updated := cov2aNewCall(nil)

	gomock.InOrder(
		m.db.EXPECT().CallGet(ctx, c.ID).Return(c, nil),
		m.extMedia.EXPECT().Stop(ctx, emID).Return(&externalmedia.ExternalMedia{ID: emID}, nil),
		m.db.EXPECT().CallGet(ctx, c.ID).Return(updated, nil),
	)

	got, err := h.ExternalMediaStop(ctx, c.ID, emID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, updated) {
		t.Errorf("Wrong match. expect: %v, got: %v", updated, got)
	}
}
