package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
)

func Test_RecordingStart_cov2a(t *testing.T) {
	recID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000020")

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
			name: "recording start error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.recording.EXPECT().Start(gomock.Any(), c.ActiveflowID, recording.ReferenceTypeCall, c.ID, recording.Format("wav"), 0, "", 0, uuid.Nil).Return(nil, errCov2aBoom)
			},
			wantErr: "could not start the recording",
		},
		{
			name: "recording id update error",
			setup: func(m *cov2aMocks, c *call.Call) {
				rec := &recording.Recording{}
				rec.ID = recID
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.recording.EXPECT().Start(gomock.Any(), c.ActiveflowID, recording.ReferenceTypeCall, c.ID, recording.Format("wav"), 0, "", 0, uuid.Nil).Return(rec, nil)
				m.db.EXPECT().CallSetRecordingID(gomock.Any(), c.ID, recID).Return(errCov2aBoom)
			},
			wantErr: "could not update recording id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			got, err := h.RecordingStart(context.Background(), c.ID, recording.Format("wav"), 0, "", 0, uuid.Nil)
			cov2aCheckErr(t, err, tt.wantErr)
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})
	}
}

func Test_RecordingStop_cov2a(t *testing.T) {
	recID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000020")

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
			wantErr: "could not get call info",
		},
		{
			name: "call is not progressing",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.RecordingID = recID
				c.Status = call.StatusRinging
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
			},
			wantErr: "the call is not progressing",
		},
		{
			name: "recording stop error",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.RecordingID = recID
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.recording.EXPECT().Stop(gomock.Any(), recID).Return(nil, errCov2aBoom)
			},
			wantErr: "could not stop the recording",
		},
		{
			name: "recording id reset error",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.RecordingID = recID
				rec := &recording.Recording{}
				rec.ID = recID
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.recording.EXPECT().Stop(gomock.Any(), recID).Return(rec, nil)
				m.db.EXPECT().CallSetRecordingID(gomock.Any(), c.ID, uuid.Nil).Return(errCov2aBoom)
			},
			wantErr: "could not update recording id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			got, err := h.RecordingStop(context.Background(), c.ID)
			cov2aCheckErr(t, err, tt.wantErr)
			if got != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", got)
			}
		})
	}
}
