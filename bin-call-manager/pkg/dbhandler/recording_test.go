package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_RecordingCreate(t *testing.T) {
	type test struct {
		name string

		recording *recording.Recording

		responseCurTime string

		expectedRes *recording.Recording
	}

	tests := []test{
		{
			name: "have all",

			recording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b075f22a-2b59-11eb-aeee-eb56de01c1b1"),
					CustomerID: uuid.FromStringOrNil("de299b2e-7f43-11ec-b9c5-67885bdabb39"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("a19704ac-2bf9-11ef-9691-7768f2e4877f"),
				},

				ActiveflowID:  uuid.FromStringOrNil("48a3b6e2-12f8-11f0-91fe-5bd3c476e4f7"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:        recording.StatusRecording,
				Format:        "wav",

				OnEndFlowID: uuid.FromStringOrNil("bf507a98-053b-11f0-8d47-5f7eaa3ba62c"),

				RecordingName: "call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000",
				Filenames: []string{
					"call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000_in.wav",
					"call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000_out.wav",
				},

				// AsteriskID: "3e:50:6b:43:bb:30",
				ChannelIDs: []string{
					"b10c2e84-2b59-11eb-b963-db658ca2c824",
					"125a1ea4-8cb9-11ed-b34c-336ac5eeeec4",
				},

				TMStart: "2020-04-18T03:22:18.995000Z",
				TMEnd:   "2020-04-18T03:22:19.995000Z",
			},

			responseCurTime: "2020-04-18T03:22:17.995000Z",

			expectedRes: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b075f22a-2b59-11eb-aeee-eb56de01c1b1"),
					CustomerID: uuid.FromStringOrNil("de299b2e-7f43-11ec-b9c5-67885bdabb39"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("a19704ac-2bf9-11ef-9691-7768f2e4877f"),
				},

				ActiveflowID:  uuid.FromStringOrNil("48a3b6e2-12f8-11f0-91fe-5bd3c476e4f7"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b1439856-2b59-11eb-89c1-678a053c5c86"),
				Status:        recording.StatusRecording,
				Format:        "wav",

				OnEndFlowID: uuid.FromStringOrNil("bf507a98-053b-11f0-8d47-5f7eaa3ba62c"),

				RecordingName: "call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000",
				Filenames: []string{
					"call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000_in.wav",
					"call_b1439856-2b59-11eb-89c1-678a053c5c86_2020-04-18T03:22:17.995000_out.wav",
				},

				// AsteriskID: "3e:50:6b:43:bb:30",
				ChannelIDs: []string{
					"b10c2e84-2b59-11eb-b963-db658ca2c824",
					"125a1ea4-8cb9-11ed-b34c-336ac5eeeec4",
				},

				TMStart:  "2020-04-18T03:22:18.995000Z",
				TMEnd:    "2020-04-18T03:22:19.995000Z",
				TMCreate: "2020-04-18T03:22:17.995000Z",
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

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any()).Return(nil)
			if err := h.RecordingCreate(ctx, tt.recording); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RecordingGet(ctx, tt.recording.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any()).Return(nil)
			res, err := h.RecordingGet(ctx, tt.recording.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}

			resGetByRecordingName, err := h.RecordingGetByRecordingName(ctx, tt.recording.RecordingName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, resGetByRecordingName) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, resGetByRecordingName)
			}
		})
	}
}

func Test_RecordingList(t *testing.T) {

	type test struct {
		name string

		recordings []*recording.Recording

		filters map[recording.Field]any

		responseCurTime string

		expectedRes []*recording.Recording
	}

	tests := []test{
		{
			name: "normal",
			recordings: []*recording.Recording{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("72ccda84-878d-11eb-ba5a-973cd51aa68a"),
						CustomerID: uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c9b4cb8a-878e-11eb-9855-7b5ad1e3392c"),
						CustomerID: uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					},
				},
			},

			filters: map[recording.Field]any{
				recording.FieldCustomerID: uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
				recording.FieldDeleted:    false,
			},
			responseCurTime: "2020-04-18T03:22:17.995000Z",

			expectedRes: []*recording.Recording{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("72ccda84-878d-11eb-ba5a-973cd51aa68a"),
						CustomerID: uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					},
					Filenames:  []string{},
					ChannelIDs: []string{},

					TMStart:  "",
					TMEnd:    "",
					TMCreate: "2020-04-18T03:22:17.995000Z",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c9b4cb8a-878e-11eb-9855-7b5ad1e3392c"),
						CustomerID: uuid.FromStringOrNil("f15430d8-7f43-11ec-b82c-b7ffeefaf0b9"),
					},
					Filenames:  []string{},
					ChannelIDs: []string{},

					TMStart:  "",
					TMEnd:    "",
					TMCreate: "2020-04-18T03:22:17.995000Z",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			name: "empty",

			recordings: []*recording.Recording{},

			filters: map[recording.Field]any{
				recording.FieldCustomerID: uuid.FromStringOrNil("08cb92b0-7f44-11ec-8753-6f51eae532cc"),
			},
			responseCurTime: "2020-04-18T03:22:17.995000Z",

			expectedRes: []*recording.Recording{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, recording := range tt.recordings {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().RecordingSet(ctx, gomock.Any()).Return(nil)
				_ = h.RecordingCreate(ctx, recording)
			}

			res, err := h.RecordingList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_RecordingDelete(t *testing.T) {

	type test struct {
		name      string
		recording *recording.Recording

		id uuid.UUID

		responseCurTime string

		expectedRes *recording.Recording
	}

	tests := []test{
		{
			name: "normal",
			recording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("86d8f342-8eb5-11ed-b1b3-cf6176be331f"),
				},
			},

			id:              uuid.FromStringOrNil("86d8f342-8eb5-11ed-b1b3-cf6176be331f"),
			responseCurTime: "2020-04-18T03:22:18.995000Z",

			expectedRes: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("86d8f342-8eb5-11ed-b1b3-cf6176be331f"),
				},

				Filenames:  []string{},
				ChannelIDs: []string{},

				TMCreate: "2020-04-18T03:22:18.995000Z",
				TMUpdate: "2020-04-18T03:22:18.995000Z",
				TMDelete: "2020-04-18T03:22:18.995000Z",
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any())
			if err := h.RecordingCreate(ctx, tt.recording); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any())
			if err := h.RecordingDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RecordingGet(ctx, tt.recording.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RecordingSet(ctx, gomock.Any())
			res, err := h.RecordingGet(ctx, tt.recording.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
