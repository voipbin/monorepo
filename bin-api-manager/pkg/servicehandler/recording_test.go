package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"
	smfile "monorepo/bin-storage-manager/models/file"
	smrecordingpeak "monorepo/bin-storage-manager/models/recordingpeak"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_RecordingList(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		size  uint64
		token string

		responseRecording []cmrecording.Recording

		expectFilters map[cmrecording.Field]any
		expectRes     []*cmrecording.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			10,
			"2020-10-20T01:00:00.995000Z",

			[]cmrecording.Recording{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Filenames: []string{
						"call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Filenames: []string{
						"call_2f167946-b2b4-4370-94fa-d6c2c57c84da_2020-12-04T18:48:03Z",
					},
				},
			},

			map[cmrecording.Field]any{
				cmrecording.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				cmrecording.FieldDeleted:    false,
			},
			[]*cmrecording.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseRecording, nil)

			res, err := h.RecordingList(ctx, tt.agent, tt.size, tt.token)

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res[0])
			}
		})
	}
}

func Test_RecordingPlayfilesGet(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")
	fileInID := uuid.FromStringOrNil("11111111-6146-11eb-be45-83bc6e54dfb9")
	fileOutID := uuid.FromStringOrNil("22222222-6146-11eb-be45-83bc6e54dfb9")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(&cmrecording.Recording{
		Identity: commonidentity.Identity{
			ID:         recordingID,
			CustomerID: customerID,
		},
		ReferenceType: cmrecording.ReferenceTypeCall,
	}, nil)

	mockReq.EXPECT().StorageV1FileList(ctx, "", uint64(100), gomock.Any()).Return([]smfile.File{
		{
			Identity:    commonidentity.Identity{ID: fileInID, CustomerID: customerID},
			Filename:    "call_rec_in.wav",
			Filesize:    100,
			URIDownload: "https://example.com/in",
		},
		{
			Identity:    commonidentity.Identity{ID: fileOutID, CustomerID: customerID},
			Filename:    "call_rec_out.wav",
			Filesize:    200,
			URIDownload: "https://example.com/out",
		},
	}, nil)

	mockReq.EXPECT().StorageV1RecordingPeaks(ctx, recordingID, 30000).Return(map[string]smrecordingpeak.RecordingFilePeak{
		"call_rec_in.wav":  {Peaks: []float64{0.1, 0.2}, Duration: 1.5},
		"call_rec_out.wav": {Peaks: []float64{0.3, 0.4}, Duration: 2.5},
	}, nil)

	res, err := h.RecordingPlayfilesGet(ctx, agent, recordingID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Wrong count. expect: 2, got: %d", len(res))
	}
	// in first, out second (sort order)
	if res[0].Direction == nil || *res[0].Direction != openapi_server.ApiManagerRecordingPlayfileDirectionIn {
		t.Errorf("Wrong direction[0]. expect: in, got: %v", res[0].Direction)
	}
	if res[1].Direction == nil || *res[1].Direction != openapi_server.ApiManagerRecordingPlayfileDirectionOut {
		t.Errorf("Wrong direction[1]. expect: out, got: %v", res[1].Direction)
	}
	if res[0].Peaks == nil || len(*res[0].Peaks) != 2 {
		t.Errorf("Wrong peaks[0]. got: %v", res[0].Peaks)
	}
	if res[0].Duration == nil || *res[0].Duration != 1.5 {
		t.Errorf("Wrong duration[0]. got: %v", res[0].Duration)
	}
}

func Test_RecordingPlayfilesGet_confbridgeNoDirection(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")
	fileID := uuid.FromStringOrNil("11111111-6146-11eb-be45-83bc6e54dfb9")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	// confbridge single file named with an _in suffix must NOT be labeled "in".
	mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(&cmrecording.Recording{
		Identity:      commonidentity.Identity{ID: recordingID, CustomerID: customerID},
		ReferenceType: cmrecording.ReferenceTypeConfbridge,
	}, nil)
	mockReq.EXPECT().StorageV1FileList(ctx, "", uint64(100), gomock.Any()).Return([]smfile.File{
		{
			Identity:    commonidentity.Identity{ID: fileID, CustomerID: customerID},
			Filename:    "confbridge_rec_in.wav",
			URIDownload: "https://example.com/cb",
		},
	}, nil)
	mockReq.EXPECT().StorageV1RecordingPeaks(ctx, recordingID, 30000).Return(map[string]smrecordingpeak.RecordingFilePeak{}, nil)

	res, err := h.RecordingPlayfilesGet(ctx, agent, recordingID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("Wrong count. expect: 1, got: %d", len(res))
	}
	if res[0].Direction == nil || *res[0].Direction != openapi_server.ApiManagerRecordingPlayfileDirectionNone {
		t.Errorf("confbridge file should have no direction. got: %v", res[0].Direction)
	}
}

func Test_RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		agent       *auth.AuthIdentity
		recordingID uuid.UUID

		responseRecording *cmrecording.Recording
		expectRes         *cmrecording.WebhookMessage
	}{
		{
			"normal",

			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),

			&cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},

			&cmrecording.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().CallV1RecordingDelete(ctx, tt.recordingID).Return(tt.responseRecording, nil)

			res, err := h.RecordingDelete(ctx, tt.agent, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RecordingTranscribeList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")
	aiManagerID := uuid.FromStringOrNil("3b3ba1a8-c1d5-11ec-9e14-000000000001")

	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		size  uint64
		token string

		responseRecording  *cmrecording.Recording
		responseTranscribe []tmtranscribe.Transcribe

		expectFilters map[tmtranscribe.Field]any
		expectRes     []*tmtranscribe.WebhookMessage
	}{
		{
			name:  "normal - returns ai-manager owned transcribe too (owner-agnostic)",
			agent: agentAdmin,
			size:  10,
			token: "2020-10-20T01:00:00.995000Z",

			responseRecording: &cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID:         recordingID,
					CustomerID: customerID,
				},
			},
			responseTranscribe: []tmtranscribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a1000000-0000-11eb-be45-000000000001"),
						CustomerID: aiManagerID,
					},
					ReferenceType: tmtranscribe.ReferenceTypeRecording,
					ReferenceID:   recordingID,
				},
			},

			expectFilters: map[tmtranscribe.Field]any{
				tmtranscribe.FieldReferenceType: string(tmtranscribe.ReferenceTypeRecording),
				tmtranscribe.FieldReferenceID:   recordingID,
				tmtranscribe.FieldDeleted:       false,
			},
			expectRes: []*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a1000000-0000-11eb-be45-000000000001"),
						CustomerID: aiManagerID,
					},
					ReferenceType: tmtranscribe.ReferenceTypeRecording,
					ReferenceID:   recordingID,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseTranscribe, nil)

			res, err := h.RecordingTranscribeList(ctx, tt.agent, recordingID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RecordingTranscribeList_permissionDenied(t *testing.T) {
	recordingOwner := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomer := uuid.FromStringOrNil("aaaaaaaa-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")

	// agent belongs to a different customer than the recording's owner
	agentOther := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: otherCustomer,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(&cmrecording.Recording{
		Identity: commonidentity.Identity{ID: recordingID, CustomerID: recordingOwner},
	}, nil)
	// no TranscribeV1TranscribeList call expected (blocked at permission gate)

	_, err := h.RecordingTranscribeList(ctx, agentOther, recordingID, 10, "2020-10-20T01:00:00.995000Z")
	if err == nil {
		t.Errorf("Wrong match. expect: permission denied error, got: nil")
	}
}

func Test_RecordingTranscriptList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")
	transcribeID := uuid.FromStringOrNil("b2000000-0000-11eb-be45-000000000002")
	aiManagerID := uuid.FromStringOrNil("3b3ba1a8-c1d5-11ec-9e14-000000000001")

	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	token := "2020-10-20T01:00:00.995000Z"
	expectFilters := map[tmtranscript.Field]any{
		tmtranscript.FieldTranscribeID: transcribeID,
		tmtranscript.FieldDeleted:      false,
	}
	responseTranscript := []tmtranscript.Transcript{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("c3000000-0000-11eb-be45-000000000003"),
				CustomerID: aiManagerID,
			},
			TranscribeID: transcribeID,
		},
	}
	expectRes := []*tmtranscript.WebhookMessage{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("c3000000-0000-11eb-be45-000000000003"),
				CustomerID: aiManagerID,
			},
			TranscribeID: transcribeID,
		},
	}

	mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(&cmrecording.Recording{
		Identity: commonidentity.Identity{ID: recordingID, CustomerID: customerID},
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity:      commonidentity.Identity{ID: transcribeID, CustomerID: aiManagerID},
		ReferenceType: tmtranscribe.ReferenceTypeRecording,
		ReferenceID:   recordingID,
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, token, uint64(10), expectFilters).Return(responseTranscript, nil)

	res, err := h.RecordingTranscriptList(ctx, agentAdmin, recordingID, transcribeID, 10, token)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if reflect.DeepEqual(res, expectRes) != true {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

func Test_RecordingTranscriptList_transcribeNotBelongToRecording(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	recordingID := uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9")
	otherRecordingID := uuid.FromStringOrNil("99999999-6146-11eb-be45-83bc6e54dfb9")
	transcribeID := uuid.FromStringOrNil("b2000000-0000-11eb-be45-000000000002")

	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	mockReq.EXPECT().CallV1RecordingGet(ctx, recordingID).Return(&cmrecording.Recording{
		Identity: commonidentity.Identity{ID: recordingID, CustomerID: customerID},
	}, nil)
	// transcribe belongs to a DIFFERENT recording -> gate 2 must block
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity:      commonidentity.Identity{ID: transcribeID, CustomerID: customerID},
		ReferenceType: tmtranscribe.ReferenceTypeRecording,
		ReferenceID:   otherRecordingID,
	}, nil)
	// no TranscriptList call expected

	_, err := h.RecordingTranscriptList(ctx, agentAdmin, recordingID, transcribeID, 10, "2020-10-20T01:00:00.995000Z")
	if err == nil {
		t.Errorf("Wrong match. expect: permission denied error, got: nil")
	}
}
