package requesthandler

// func Test_TranscribeV1StreamingCreate(t *testing.T) {

// 	type test struct {
// 		name string

// 		customerID    uuid.UUID
// 		referenceID   uuid.UUID
// 		referenceType tstranscribe.Type

// 		language string

// 		expectTarget  string
// 		expectRequest *rabbitmqhandler.Request
// 		response      *rabbitmqhandler.Response

// 		expectResult *tstranscribe.Transcribe
// 	}

// 	tests := []test{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("7ce083de-8734-11ec-82e8-df2232923291"),
// 			uuid.FromStringOrNil("c52a575e-8735-11ec-87b6-d3b5433e0e30"),
// 			tstranscribe.TypeCall,
// 			"en-US",

// 			"bin-manager.transcribe-manager.request",
// 			&rabbitmqhandler.Request{
// 				URI:      "/v1/streamings",
// 				Method:   rabbitmqhandler.RequestMethodPost,
// 				DataType: ContentTypeJSON,
// 				Data:     []byte(`{"customer_id":"7ce083de-8734-11ec-82e8-df2232923291","reference_id":"c52a575e-8735-11ec-87b6-d3b5433e0e30","reference_type":"call","language":"en-US"}`),
// 			},
// 			&rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`{"id":"6ee12898-8774-11ec-a4c9-0b9b176502bd"}`),
// 			},
// 			&tstranscribe.Transcribe{
// 				ID: uuid.FromStringOrNil("6ee12898-8774-11ec-a4c9-0b9b176502bd"),
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

// 			res, err := reqHandler.TranscribeV1StreamingCreate(ctx, tt.customerID, tt.referenceID, tt.referenceType, tt.language)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(tt.expectResult, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
// 			}
// 		})
// 	}
// }
