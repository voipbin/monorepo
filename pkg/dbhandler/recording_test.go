package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/recording"
)

func TestRecordingCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		record       *recording.Recording
		expectRecord *recording.Recording
	}

	tests := []test{
		{
			"normal",
			&recording.Recording{
				ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:      recording.StatusRecording,
				Format:      "wav",
				Filename:    "call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000.wav",

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
			},
			&recording.Recording{
				ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:      recording.StatusRecording,
				Format:      "wav",
				Filename:    "call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000.wav",

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().RecordSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.RecordingCreate(context.Background(), tt.record); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RecordingGet(gomock.Any(), tt.record.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RecordSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.RecordingGet(context.Background(), tt.record.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRecord, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRecord, res)
			}
		})
	}
}

func TestRecordingGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		userID  uint64
		records []*recording.Recording
		// expectRecord *recording.Recording
	}

	tests := []test{
		{
			"normal",
			1,
			[]*recording.Recording{
				&recording.Recording{
					ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
					UserID:      1,
					Type:        recording.TypeCall,
					ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
					Status:      recording.StatusRecording,
					Format:      "wav",

					AsteriskID: "3e:50:6b:43:bb:30",
					ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
				},
				&recording.Recording{
					ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
					UserID:      1,
					Type:        recording.TypeCall,
					ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
					Status:      recording.StatusRecording,
					Format:      "wav",

					AsteriskID: "3e:50:6b:43:bb:30",
					ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
				},
			},
		},
		{
			"empty",
			2,
			[]*recording.Recording{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			res, err := h.RecordingGets(context.Background(), tt.userID, 10, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for i, recording := range res {
				if recording.ID != tt.records[i].ID ||
					recording.AsteriskID != tt.records[i].AsteriskID ||
					recording.ChannelID != tt.records[i].ChannelID ||
					recording.Format != tt.records[i].Format ||
					recording.ReferenceID != tt.records[i].ReferenceID ||
					recording.Status != tt.records[i].Status ||
					recording.Type != tt.records[i].Type ||
					recording.UserID != tt.records[i].UserID {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.records[i], recording)
				}
			}
		})
	}
}
