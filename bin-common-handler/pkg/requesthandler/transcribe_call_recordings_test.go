package requesthandler

// func Test_TranscribeV1CallRecordingCreate(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customerID uuid.UUID
// 		callID     uuid.UUID
// 		language   string
// 		timeout    int
// 		delay      int

// 		expectTarget  string
// 		expectRequest *sock.Request
// 		response      *sock.Response

// 		expectResult []tstranscribe.Transcribe
// 	}{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("7ce083de-8734-11ec-82e8-df2232923291"),
// 			uuid.FromStringOrNil("c52a575e-8735-11ec-87b6-d3b5433e0e30"),
// 			"en-US",
// 			3000,
// 			0,

// 			"bin-manager.transcribe-manager.request",
// 			&sock.Request{
// 				URI:      "/v1/call_recordings",
// 				Method:   sock.RequestMethodPost,
// 				DataType: ContentTypeJSON,
// 				Data:     []byte(`{"customer_id":"7ce083de-8734-11ec-82e8-df2232923291","reference_id":"c52a575e-8735-11ec-87b6-d3b5433e0e30","language":"en-US"}`),
// 			},
// 			&sock.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`[{"id":"10e438e2-a3eb-11eb-889c-975ac37d96fe"}]`),
// 			},
// 			[]tstranscribe.Transcribe{
// 				{
// 					ID: uuid.FromStringOrNil("10e438e2-a3eb-11eb-889c-975ac37d96fe"),
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
// 			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

// 			res, err := reqHandler.TranscribeV1CallRecordingCreate(ctx, tt.customerID, tt.callID, tt.language, tt.timeout, tt.delay)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(tt.expectResult, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
// 			}
// 		})
// 	}
// }
