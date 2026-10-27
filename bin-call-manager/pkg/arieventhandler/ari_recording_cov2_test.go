package arieventhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func Test_EventHandlerRecording_cov2(t *testing.T) {
	recID := uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")
	rec := &recording.Recording{Identity: commonidentity.Identity{ID: recID}}

	tests := []struct {
		name string

		started  bool
		recName  string
		getErr   error
		stateErr error

		expectGet   bool
		expectState bool
	}{
		{name: "started out ignored", started: true, recName: "rec_out"},
		{name: "started get error", started: true, recName: "rec_in", getErr: errors.New("get"), expectGet: true},
		{name: "started state error", started: true, recName: "rec_in", stateErr: errors.New("st"), expectGet: true, expectState: true},
		{name: "started success", started: true, recName: "rec_in", expectGet: true, expectState: true},
		{name: "finished out ignored", recName: "rec_out"},
		{name: "finished get error", recName: "rec_in", getErr: errors.New("get"), expectGet: true},
		{name: "finished state error", recName: "rec_in", stateErr: errors.New("st"), expectGet: true, expectState: true},
		{name: "finished success", recName: "rec_in", expectGet: true, expectState: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRec := recordinghandler.NewMockRecordingHandler(mc)
			h := eventHandler{recordingHandler: mockRec}

			if tt.expectGet {
				if tt.getErr != nil {
					mockRec.EXPECT().GetByRecordingName(gomock.Any(), "rec").Return(nil, tt.getErr)
				} else {
					mockRec.EXPECT().GetByRecordingName(gomock.Any(), "rec").Return(rec, nil)
				}
			}
			if tt.expectState {
				if tt.started {
					mockRec.EXPECT().Started(gomock.Any(), recID).Return(rec, tt.stateErr)
				} else {
					mockRec.EXPECT().Stopped(gomock.Any(), recID).Return(rec, tt.stateErr)
				}
			}

			var err error
			if tt.started {
				err = h.EventHandlerRecordingStarted(context.Background(), &ari.RecordingStarted{Recording: ari.RecordingLive{Name: tt.recName}})
			} else {
				err = h.EventHandlerRecordingFinished(context.Background(), &ari.RecordingFinished{Recording: ari.RecordingLive{Name: tt.recName}})
			}

			expect := tt.getErr
			if expect == nil {
				expect = tt.stateErr
			}
			if err != expect {
				t.Errorf("Wrong match. expect: %v, got: %v", expect, err)
			}
		})
	}
}
