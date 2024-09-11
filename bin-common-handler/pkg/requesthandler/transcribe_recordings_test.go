package requesthandler

// func Test_TranscribeV1RecordingPost(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customerID  uuid.UUID
// 		recordingID uuid.UUID
// 		language    string

// 		expectTarget  string
// 		expectRequest *sock.Request
// 		response      *sock.Response

// 		expectResult *tstranscribe.Transcribe
// 	}{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("7ce083de-8734-11ec-82e8-df2232923291"),
// 			uuid.FromStringOrNil("138cbdc2-a3ea-11eb-9a91-3b876395af6e"),
// 			"en-US",

// 			"bin-manager.transcribe-manager.request",
// 			&sock.Request{
// 				URI:      "/v1/recordings",
// 				Method:   sock.RequestMethodPost,
// 				DataType: ContentTypeJSON,
// 				Data:     []byte(`{"customer_id":"7ce083de-8734-11ec-82e8-df2232923291","reference_id":"138cbdc2-a3ea-11eb-9a91-3b876395af6e","language":"en-US"}`),
// 			},
// 			&sock.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`{"id":"10e438e2-a3eb-11eb-889c-975ac37d96fe","customer_id":"7ce083de-8734-11ec-82e8-df2232923291","type":"recording","reference_id":"138cbdc2-a3ea-11eb-9a91-3b876395af6e","language":"en-US","webhook_uri":"","webhook_method":"","transcripts":[{"direction":"both","message":"Hello, this is voipbin. Thank you."}]}`),
// 			},
// 			&tstranscribe.Transcribe{
// 				ID:          uuid.FromStringOrNil("10e438e2-a3eb-11eb-889c-975ac37d96fe"),
// 				CustomerID:  uuid.FromStringOrNil("7ce083de-8734-11ec-82e8-df2232923291"),
// 				Type:        tstranscribe.TypeRecording,
// 				ReferenceID: uuid.FromStringOrNil("138cbdc2-a3ea-11eb-9a91-3b876395af6e"),
// 				Language:    "en-US",
// 				Transcripts: []tstranscript.Transcript{
// 					{
// 						Direction: tscommon.DirectionBoth,
// 						Message:   "Hello, this is voipbin. Thank you.",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 			reqHandler := requestHandler{
// 				sock: mockSock,
// 			}

// 			ctx := context.Background()
// 			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

// 			res, err := reqHandler.TranscribeV1RecordingCreate(ctx, tt.customerID, tt.recordingID, tt.language)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectResult, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
// 			}
// 		})
// 	}
// }
