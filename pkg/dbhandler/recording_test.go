package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
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
				ID:          uuid.FromStringOrNil("b075f22a-2b59-11eb-aeee-eb56de01c1b1"),
				CustomerID:  uuid.FromStringOrNil("de299b2e-7f43-11ec-b9c5-67885bdabb39"),
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:      recording.StatusRecording,
				Format:      "wav",
				Filename:    "call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000.wav",

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
			},
			&recording.Recording{
				ID:          uuid.FromStringOrNil("b075f22a-2b59-11eb-aeee-eb56de01c1b1"),
				CustomerID:  uuid.FromStringOrNil("de299b2e-7f43-11ec-b9c5-67885bdabb39"),
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

			mockCache.EXPECT().RecordingSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.RecordingCreate(context.Background(), tt.record); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RecordingGet(gomock.Any(), tt.record.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RecordingSet(gomock.Any(), gomock.Any()).Return(nil)
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
	h := NewHandler(dbTest, mockCache)

	type test struct {
		name string

		customerID uuid.UUID
		recordings []*recording.Recording
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
			[]*recording.Recording{
				{
					ID:          uuid.FromStringOrNil("72ccda84-878d-11eb-ba5a-973cd51aa68a"),
					CustomerID:  uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					Type:        recording.TypeCall,
					ReferenceID: uuid.FromStringOrNil("77a43886-878d-11eb-b4d3-a373acdc4de4"),
					Status:      recording.StatusRecording,
					Format:      "wav",

					AsteriskID: "3e:50:6b:43:bb:30",
					ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
				},
				{
					ID:          uuid.FromStringOrNil("c9b4cb8a-878e-11eb-9855-7b5ad1e3392c"),
					CustomerID:  uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					Type:        recording.TypeCall,
					ReferenceID: uuid.FromStringOrNil("ccca16ea-878e-11eb-98ff-f3dd532a2331"),
					Status:      recording.StatusRecording,
					Format:      "wav",

					AsteriskID: "3e:50:6b:43:bb:30",
					ChannelID:  "d09176ba-878e-11eb-a3f1-8743bd4202ae",
				},
			},
		},
		{
			"empty",
			uuid.FromStringOrNil("08cb92b0-7f44-11ec-8753-6f51eae532cc"),
			[]*recording.Recording{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, recording := range tt.recordings {
				mockCache.EXPECT().RecordingSet(gomock.Any(), gomock.Any()).Return(nil)
				_ = h.RecordingCreate(ctx, recording)
			}

			res, err := h.RecordingGets(context.Background(), tt.customerID, 10, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, recording := range res {
				recording.TMCreate = ""
				recording.TMUpdate = ""
				recording.TMDelete = ""
			}

			for i, j := len(res)-1, 0; i >= 0; i, j = i-1, j+1 {
				recording := tt.recordings[i]
				if reflect.DeepEqual(recording, res[j]) != true {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", recording, res[j])
				}
			}
		})
	}
}
