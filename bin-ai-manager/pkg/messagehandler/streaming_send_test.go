package messagehandler

// func Test_streamingSendResponseHandleText(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		cc       *aicall.AIcall
// 		msgID    uuid.UUID
// 		chanText chan string

// 		expectedTexts   []string
// 		expectedMessage *message.Message
// 	}{
// 		{
// 			name: "normal",

// 			cc: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("4526df10-9522-11f0-927b-e7d26d7d0671"),
// 					CustomerID: uuid.FromStringOrNil("457d5d5e-9522-11f0-895e-3b40dece012e"),
// 				},
// 			},
// 			msgID: uuid.FromStringOrNil("4559a184-9522-11f0-bb51-e7e121b7e5a9"),
// 			chanText: func() chan string {
// 				ch := make(chan string, 10)
// 				ch <- "hello world"
// 				return ch
// 			}(),

// 			expectedTexts: []string{
// 				"hello world",
// 			},
// 			expectedMessage: &message.Message{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("4559a184-9522-11f0-bb51-e7e121b7e5a9"),
// 					CustomerID: uuid.FromStringOrNil("457d5d5e-9522-11f0-895e-3b40dece012e"),
// 				},
// 				AIcallID: uuid.FromStringOrNil("4526df10-9522-11f0-927b-e7d26d7d0671"),

// 				Direction: message.DirectionIncoming,
// 				Role:      message.RoleAssistant,
// 				Content:   "hello world",
// 				ToolCalls: []message.ToolCall{},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockGPT := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

// 			h := &messageHandler{
// 				utilHandler:   mockUtil,
// 				notifyHandler: mockNotify,
// 				db:            mockDB,

// 				engineOpenaiHandler: mockGPT,
// 				reqHandler:          mockReq,
// 			}
// 			ctx := context.Background()

// 			close(tt.chanText)

// 			for _, msg := range tt.expectedTexts {
// 				mockReq.EXPECT().TTSV1StreamingSayAdd(ctx, tt.cc.TTSStreamingPodID, tt.cc.TTSStreamingID, tt.msgID, msg).Return(nil)
// 			}

// 			// create message for tool call request
// 			mockDB.EXPECT().MessageCreate(ctx, tt.expectedMessage).Return(nil)
// 			mockDB.EXPECT().MessageGet(ctx, tt.expectedMessage.ID).Return(tt.expectedMessage, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedMessage.CustomerID, message.EventTypeMessageCreated, tt.expectedMessage)

// 			res, err := h.streamingSendResponseHandleText(ctx, tt.cc, tt.msgID, tt.chanText)
// 			if err != nil {
// 				t.Errorf("Wrong match. expected ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectedMessage) {
// 				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedMessage, res)
// 			}
// 		})
// 	}
// }

// func Test_streamingSendResponseHandleTool(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		cc           *aicall.AIcall
// 		chanToolCall chan *message.ToolCall

// 		responseActiveflow *fmactiveflow.Activeflow

// 		expectedActions              []fmaction.Action
// 		expectedMessageToolRequest   *message.Message
// 		expectedMessagesToolResponse []*message.Message
// 		expectedTerminate            bool

// 		expectedRes *message.Message
// 	}{
// 		{
// 			name: "normal",

// 			cc: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("1178198a-9451-11f0-b070-677fbb141403"),
// 				},
// 				ActiveflowID: uuid.FromStringOrNil("aaccbdb4-9453-11f0-9537-fb7307d705d5"),
// 			},
// 			chanToolCall: func() chan *message.ToolCall {
// 				ch := make(chan *message.ToolCall, 10)
// 				ch <- &message.ToolCall{
// 					ID:   "aa9101f2-9453-11f0-9a77-fbab20dbb541",
// 					Type: message.ToolTypeFunction,
// 					Function: message.FunctionCall{
// 						Name:      string(fmaction.TypeConnect),
// 						Arguments: `{"destinations": [{"target":"+1234567890"}]}`,
// 					},
// 				}
// 				return ch
// 			}(),

// 			responseActiveflow: &fmactiveflow.Activeflow{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("aaccbdb4-9453-11f0-9537-fb7307d705d5"),
// 				},
// 			},

// 			expectedActions: []fmaction.Action{
// 				{
// 					Type: fmaction.TypeConnect,
// 					Option: map[string]any{
// 						"source": map[string]any{},
// 						"destinations": []any{
// 							map[string]any{
// 								"target": "+1234567890",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			expectedMessageToolRequest: &message.Message{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("e4e311fe-94af-11f0-a70a-871c7d4f55db"),
// 				},
// 				AIcallID: uuid.FromStringOrNil("1178198a-9451-11f0-b070-677fbb141403"),

// 				Direction: message.DirectionIncoming,
// 				Role:      message.RoleAssistant,
// 				Content:   "",
// 				ToolCalls: []message.ToolCall{
// 					{
// 						ID:   "aa9101f2-9453-11f0-9a77-fbab20dbb541",
// 						Type: message.ToolTypeFunction,
// 						Function: message.FunctionCall{
// 							Name:      string(fmaction.TypeConnect),
// 							Arguments: `{"destinations": [{"target":"+1234567890"}]}`,
// 						},
// 					},
// 				},
// 			},
// 			expectedMessagesToolResponse: []*message.Message{
// 				{
// 					Identity: identity.Identity{
// 						ID: uuid.FromStringOrNil("e5280250-94af-11f0-9bbe-8b0c7e647091"),
// 					},
// 					AIcallID: uuid.FromStringOrNil("1178198a-9451-11f0-b070-677fbb141403"),

// 					Direction:  message.DirectionOutgoing,
// 					Role:       message.RoleTool,
// 					Content:    `{"result": "success"}`,
// 					ToolCalls:  []message.ToolCall{},
// 					ToolCallID: "aa9101f2-9453-11f0-9a77-fbab20dbb541",
// 				},
// 			},
// 			expectedTerminate: true,
// 			expectedRes: &message.Message{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("e4e311fe-94af-11f0-a70a-871c7d4f55db"),
// 				},
// 				AIcallID: uuid.FromStringOrNil("1178198a-9451-11f0-b070-677fbb141403"),

// 				Direction: message.DirectionIncoming,
// 				Role:      message.RoleAssistant,
// 				Content:   "",
// 				ToolCalls: []message.ToolCall{
// 					{
// 						ID:   "aa9101f2-9453-11f0-9a77-fbab20dbb541",
// 						Type: message.ToolTypeFunction,
// 						Function: message.FunctionCall{
// 							Name:      string(fmaction.TypeConnect),
// 							Arguments: `{"destinations": [{"target":"+1234567890"}]}`,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockGPT := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

// 			h := &messageHandler{
// 				utilHandler:   mockUtil,
// 				notifyHandler: mockNotify,
// 				db:            mockDB,

// 				engineOpenaiHandler: mockGPT,
// 				reqHandler:          mockReq,
// 			}
// 			ctx := context.Background()

// 			close(tt.chanToolCall)

// 			mockReq.EXPECT().FlowV1ActiveflowAddActions(ctx, tt.cc.ActiveflowID, tt.expectedActions).Return(tt.responseActiveflow, nil)

// 			// create message for tool call request
// 			mockUtil.EXPECT().UUIDCreate().Return(tt.expectedMessageToolRequest.ID)
// 			mockDB.EXPECT().MessageCreate(ctx, tt.expectedMessageToolRequest).Return(nil)
// 			mockDB.EXPECT().MessageGet(ctx, tt.expectedMessageToolRequest.ID).Return(tt.expectedMessageToolRequest, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedMessageToolRequest.CustomerID, message.EventTypeMessageCreated, tt.expectedMessageToolRequest)

// 			for _, msg := range tt.expectedMessagesToolResponse {
// 				mockUtil.EXPECT().UUIDCreate().Return(msg.ID)
// 				mockDB.EXPECT().MessageCreate(ctx, msg).Return(nil)
// 				mockDB.EXPECT().MessageGet(ctx, msg.ID).Return(msg, nil)
// 				mockNotify.EXPECT().PublishWebhookEvent(ctx, msg.CustomerID, message.EventTypeMessageCreated, msg)
// 			}

// 			if tt.expectedTerminate {
// 				mockReq.EXPECT().AIV1AIcallTerminate(ctx, tt.cc.ID).Return(tt.cc, nil)
// 			}

// 			res, err := h.streamingSendResponseHandleTool(ctx, tt.cc, tt.chanToolCall)
// 			if err != nil {
// 				t.Errorf("Wrong match. expected ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectedRes) {
// 				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, res)
// 			}
// 		})
// 	}
// }
