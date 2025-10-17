package pipecatcallhandler

// func Test_runnerStartPython(t *testing.T) {

// 	type test struct {
// 		name string

// 		request *sock.Request

// 		responsePipecatcall *pipecatcall.Pipecatcall

// 		expectPipecatcallID uuid.UUID
// 		expectRes           *sock.Response
// 	}

// 	tests := []test{
// 		{
// 			name: "normal",

// 			request: &sock.Request{
// 				URI:    "/v1/pipecatcalls/e594fff6-ab0a-11f0-8220-1fe5a6807315/stop",
// 				Method: sock.RequestMethodPost,
// 			},

// 			responsePipecatcall: &pipecatcall.Pipecatcall{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("e594fff6-ab0a-11f0-8220-1fe5a6807315"),
// 				},
// 			},

// 			expectPipecatcallID: uuid.FromStringOrNil("e594fff6-ab0a-11f0-8220-1fe5a6807315"),
// 			expectRes: &sock.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`{"id":"e594fff6-ab0a-11f0-8220-1fe5a6807315","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSock := sockhandler.NewMockSockHandler(mc)
// 			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

// 			h := &listenHandler{
// 				sockHandler:        mockSock,
// 				pipecatcallHandler: mockPipecatcall,
// 			}
// 			ctx := context.Background()

// 			mockPipecatcall.EXPECT().Stop(ctx, tt.expectPipecatcallID).Return(tt.responsePipecatcall, nil)
// 			res, err := h.processRequest(tt.request)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(res, tt.expectRes) != true {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
