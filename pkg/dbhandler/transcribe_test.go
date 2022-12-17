package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
)

func Test_TranscribeCreate(t *testing.T) {

	type test struct {
		name string

		transcribe *transcribe.Transcribe

		responseCurTime string
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
				CustomerID:  uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				Type:        transcribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:      uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:    "en-US",
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
				CustomerID:  uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				Type:        transcribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:      uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:    "en-US",
				Transcripts: []transcript.Transcript{},
				TMCreate:    "2021-01-01 00:00:00.000",
				TMUpdate:    DefaultTimeStamp,
				TMDelete:    DefaultTimeStamp,
			},
		},
		{
			"has transcripts",
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				CustomerID:  uuid.FromStringOrNil("ec059f08-7ffd-11ec-8bb6-db2f62788edb"),
				Type:        transcribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("820df186-0edd-11ec-b4f8-df7e8fbe9569"),
				HostID:      uuid.FromStringOrNil("822d79e8-0edd-11ec-a46a-0b4de8e393bb"),
				Language:    "en-US",
				Transcripts: []transcript.Transcript{
					{
						Direction: common.DirectionIn,
						Message:   "Hello, this is test.",
						TMCreate:  "00:00:00.000",
					},
				},
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				CustomerID:  uuid.FromStringOrNil("ec059f08-7ffd-11ec-8bb6-db2f62788edb"),
				Type:        transcribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("820df186-0edd-11ec-b4f8-df7e8fbe9569"),
				HostID:      uuid.FromStringOrNil("822d79e8-0edd-11ec-a46a-0b4de8e393bb"),
				Language:    "en-US",
				Transcripts: []transcript.Transcript{
					{
						Direction: common.DirectionIn,
						Message:   "Hello, this is test.",
						TMCreate:  "00:00:00.000",
					},
				},
				TMCreate: "2021-01-01 00:00:00.000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any())
			if err := h.TranscribeCreate(context.Background(), tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscribeGet(gomock.Any(), tt.transcribe.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any())
			res, err := h.TranscribeGet(context.Background(), tt.transcribe.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
