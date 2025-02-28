package chatbotcallhandler

// func Test_ServiceStart(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		chatbotID     uuid.UUID
// 		activeflowID  uuid.UUID
// 		referenceType chatbotcall.ReferenceType
// 		referenceID   uuid.UUID
// 		gender        chatbotcall.Gender
// 		language      string

// 		responseChatbot         *chatbot.Chatbot
// 		responseConfbridge      *cmconfbridge.Confbridge
// 		responseUUIDChatbotcall uuid.UUID
// 		responseChatbotcall     *chatbotcall.Chatbotcall
// 		responseMessage         *chatbotcall.Message
// 		responseUUIDAction      uuid.UUID

// 		expectChatbotcall         *chatbotcall.Chatbotcall
// 		expectChatbotcallMessages []chatbotcall.Message
// 		expectMessage             *chatbotcall.Message
// 		expectRes                 *service.Service
// 	}{
// 		{
// 			name: "normal",

// 			chatbotID:     uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
// 			activeflowID:  uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
// 			referenceType: chatbotcall.ReferenceTypeCall,
// 			referenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
// 			gender:        chatbotcall.GenderFemale,
// 			language:      "en-US",

// 			responseChatbot: &chatbot.Chatbot{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
// 					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
// 				},
// 				EngineType: chatbot.EngineTypeNone,
// 				InitPrompt: "hello, this is init prompt message.",
// 			},
// 			responseConfbridge: &cmconfbridge.Confbridge{
// 				ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
// 			},
// 			responseUUIDChatbotcall: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
// 			responseChatbotcall: &chatbotcall.Chatbotcall{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
// 				},
// 				ConfbridgeID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
// 			},
// 			responseMessage: &chatbotcall.Message{
// 				Role:    "assistant",
// 				Content: "test assistant message.",
// 			},
// 			responseUUIDAction: uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

// 			expectChatbotcall: &chatbotcall.Chatbotcall{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
// 					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
// 				},
// 				ChatbotID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
// 				ActiveflowID:      uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
// 				ChatbotEngineType: chatbot.EngineTypeNone,
// 				ReferenceType:     chatbotcall.ReferenceTypeCall,
// 				ReferenceID:       uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
// 				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
// 				Gender:            chatbotcall.GenderFemale,
// 				Language:          "en-US",
// 				Messages:          []chatbotcall.Message{},
// 				Status:            chatbotcall.StatusInitiating,
// 			},
// 			expectMessage: &chatbotcall.Message{
// 				Role:    chatbotcall.MessageRoleSystem,
// 				Content: "hello, this is init prompt message.",
// 			},
// 			expectChatbotcallMessages: []chatbotcall.Message{
// 				{
// 					Role:    "system",
// 					Content: "hello, this is init prompt message.",
// 				},
// 				{
// 					Role:    "assistant",
// 					Content: "test assistant message.",
// 				},
// 			},
// 			expectRes: &service.Service{
// 				ID:   uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
// 				Type: service.TypeChatbotcall,
// 				PushActions: []fmaction.Action{
// 					{
// 						ID:     uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),
// 						Type:   fmaction.TypeConfbridgeJoin,
// 						Option: []byte(`{"confbridge_id":"ec6d153d-dd5a-4eef-bc27-8fcebe100704"}`),
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
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
// 			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

// 			h := &chatbotcallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				chatbotHandler: mockChatbot,
// 				openaiHandler:  mockChatgpt,
// 			}

// 			ctx := context.Background()

// 			mockChatbot.EXPECT().Get(ctx, tt.chatbotID).Return(tt.responseChatbot, nil)
// 			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.responseChatbot.CustomerID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
// 			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
// 			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
// 			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

// 			mockChatgpt.EXPECT().ChatNew(ctx, tt.responseChatbotcall, tt.expectMessage).Return(tt.responseMessage, nil)
// 			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.responseChatbotcall.ID, tt.expectChatbotcallMessages).Return(nil)
// 			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)

// 			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAction)

// 			res, err := h.ServiceStart(ctx, tt.chatbotID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			time.Sleep(time.Millisecond * 100)

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
