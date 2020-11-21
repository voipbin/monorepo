package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/record"
)

func TestRecordCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		record       *record.Record
		expectRecord *record.Record
	}

	tests := []test{
		{
			"normal",
			&record.Record{
				ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
				UserID:      1,
				Type:        record.TypeCall,
				ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:      record.StatusRecording,
				Format:      "wav",

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
			},
			&record.Record{
				ID:          "b075f22a-2b59-11eb-aeee-eb56de01c1b1",
				UserID:      1,
				Type:        record.TypeCall,
				ReferenceID: uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:      record.StatusRecording,
				Format:      "wav",

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "b10c2e84-2b59-11eb-b963-db658ca2c824",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().RecordSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.RecordCreate(context.Background(), tt.record); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RecordGet(gomock.Any(), tt.record.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RecordSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.RecordGet(context.Background(), tt.record.ID)
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
