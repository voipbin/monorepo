package transcribehandler

// func Test_CallRecording(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customerID uuid.UUID
// 		callID     uuid.UUID
// 		language   string

// 		responseCall        *cmcall.Call
// 		responseTranscribes []*transcribe.Transcribe

// 		expectRes []*transcribe.Transcribe
// 	}{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("419841c6-825d-11ec-823f-13ee3d677a1b"),
// 			uuid.FromStringOrNil("74582ca6-877c-11ec-937d-b3dc9da5953a"),
// 			"en-US",

// 			&cmcall.Call{
// 				RecordingIDs: []uuid.UUID{
// 					uuid.FromStringOrNil("a9e88118-877c-11ec-a30b-b7af76bdce58"),
// 				},
// 			},
// 			[]*transcribe.Transcribe{
// 				{
// 					ID: uuid.FromStringOrNil("564bbd4e-877d-11ec-84cc-978116c3fab9"),
// 				},
// 			},

// 			[]*transcribe.Transcribe{
// 				{
// 					ID: uuid.FromStringOrNil("564bbd4e-877d-11ec-84cc-978116c3fab9"),
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockTranscript := transcirpthandler.NewMockTranscriptHandler(mc)

// 			h := &transcribeHandler{
// 				reqHandler:        mockReq,
// 				db:                mockDB,
// 				notifyHandler:     mockNotify,
// 				transcriptHandler: mockTranscript,
// 			}

// 			ctx := context.Background()

// 			lang := getBCP47LanguageCode(tt.language)
// 			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)

// 			for i, recordingID := range tt.responseCall.RecordingIDs {
// 				mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(nil)
// 				mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribes[i], nil)
// 				mockTranscript.EXPECT().Recording(ctx, tt.customerID, tt.responseTranscribes[i].ID, recordingID, lang).Return(&transcript.Transcript{}, nil)
// 				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribes[i].CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribes[i])
// 			}

// 			res, err := h.CallRecording(ctx, tt.customerID, tt.callID, tt.language)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
