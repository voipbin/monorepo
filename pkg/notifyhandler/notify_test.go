package notifyhandler

// func TestNotifyEventCall(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	exchangeDelay := ""
// 	exchangeNotify := "bin-manager.call-manager.event"

// 	h := &notifyHandler{
// 		sock:           mockSock,
// 		reqHandler:     mockReq,
// 		exchangeDelay:  exchangeDelay,
// 		exchangeNotify: exchangeNotify,
// 	}

// 	type test struct {
// 		name          string
// 		eventType     EventType
// 		call          *call.Call
// 		expectEvent   *rabbitmqhandler.Event
// 		expectWebhook []byte
// 	}

// 	tests := []test{
// 		{
// 			"create normal",
// 			EventTypeCallCreated,
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("14aa3450-84eb-11eb-8285-23e72da33b42"),
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 				ChannelID:  "48a5446a-e3b1-11ea-b837-83239d9eb45f",
// 				Type:       call.TypeSipService,
// 				Direction:  call.DirectionIncoming,
// 				Destination: address.Address{
// 					Target: string(action.TypeAnswer),
// 				},
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeCallCreated),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"14aa3450-84eb-11eb-8285-23e72da33b42","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"48a5446a-e3b1-11ea-b837-83239d9eb45f","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 			},
// 			[]byte{},
// 		},
// 		{
// 			"create webhook uri",
// 			EventTypeCallCreated,
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("6d06310e-1072-11ec-9606-27a7b382621f"),
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 				ChannelID:  "72590c44-1072-11ec-85d2-73d06dfabfc2",
// 				Type:       call.TypeSipService,
// 				Direction:  call.DirectionIncoming,
// 				Destination: address.Address{
// 					Target: string(action.TypeAnswer),
// 				},
// 				WebhookURI: "test.com/webhook",
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeCallCreated),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"6d06310e-1072-11ec-9606-27a7b382621f","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"72590c44-1072-11ec-85d2-73d06dfabfc2","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 			},
// 			[]byte(`{"id":"6d06310e-1072-11ec-9606-27a7b382621f","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 		},
// 		{
// 			"update normal",
// 			EventTypeCallUpdated,
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("52678c48-853b-11eb-9693-bbab415f20a4"),
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 				ChannelID:  "5675714c-853b-11eb-a9a0-8340e19df2d1",
// 				Type:       call.TypeSipService,
// 				Direction:  call.DirectionIncoming,
// 				Destination: address.Address{
// 					Target: string(action.TypeAnswer),
// 				},
// 				WebhookURI: "test.com/webhook",
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeCallUpdated),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"52678c48-853b-11eb-9693-bbab415f20a4","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"5675714c-853b-11eb-a9a0-8340e19df2d1","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 			},
// 			[]byte(`{"id":"52678c48-853b-11eb-9693-bbab415f20a4","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 		},
// 		{
// 			"hungup normal",
// 			EventTypeCallHungup,
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("717b275c-853b-11eb-a029-5f71e2f0a0d3"),
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 				ChannelID:  "719ad61a-853b-11eb-aad3-0ff51d63724c",
// 				Type:       call.TypeSipService,
// 				Direction:  call.DirectionIncoming,
// 				Destination: address.Address{
// 					Target: string(action.TypeAnswer),
// 				},
// 				WebhookURI: "test.com/webhook",
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeCallHungup),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"717b275c-853b-11eb-a029-5f71e2f0a0d3","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"719ad61a-853b-11eb-aad3-0ff51d63724c","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 			},
// 			[]byte(`{"id":"717b275c-853b-11eb-a029-5f71e2f0a0d3","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
// 			if tt.call.WebhookURI != "" {
// 				mockReq.EXPECT().WMWebhookPOST("POST", tt.call.WebhookURI, dataTypeJSON, string(tt.eventType), tt.expectWebhook)
// 			}
// 			h.NotifyEvent(tt.eventType, tt.call.WebhookURI, tt.call)

// 			time.Sleep(time.Millisecond * 1000)
// 		})
// 	}
// }

// func TestNotifyEventRecordingStarted(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	exchangeDelay := ""
// 	exchangeNotify := "bin-manager.call-manager.event"

// 	h := &notifyHandler{
// 		sock:           mockSock,
// 		reqHandler:     mockReq,
// 		exchangeDelay:  exchangeDelay,
// 		exchangeNotify: exchangeNotify,
// 	}

// 	type test struct {
// 		name          string
// 		r             *recording.Recording
// 		expectEvent   *rabbitmqhandler.Event
// 		expectWebhook []byte
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&recording.Recording{
// 				ID:          uuid.FromStringOrNil("70fb0206-8618-11eb-96de-eb0202c2e333"),
// 				UserID:      1,
// 				Type:        recording.TypeCall,
// 				ReferenceID: uuid.FromStringOrNil("82f0c770-8618-11eb-971f-3bef56169bec"),
// 				Status:      recording.StatusRecording,
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeRecordingStarted),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"70fb0206-8618-11eb-96de-eb0202c2e333","user_id":1,"type":"call","reference_id":"82f0c770-8618-11eb-971f-3bef56169bec","status":"recording","format":"","filename":"","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 			},
// 			[]byte{},
// 		},
// 		{
// 			"with webhook_uri",
// 			&recording.Recording{
// 				ID:          uuid.FromStringOrNil("a29148ce-878b-11eb-a518-83192db03b8d"),
// 				UserID:      1,
// 				Type:        recording.TypeCall,
// 				ReferenceID: uuid.FromStringOrNil("a31758d8-878b-11eb-b410-3bd79a48fa1f"),
// 				Status:      recording.StatusRecording,
// 				WebhookURI:  "http://test.com/test_webhook",
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeRecordingStarted),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"a29148ce-878b-11eb-a518-83192db03b8d","user_id":1,"type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","filename":"","webhook_uri":"http://test.com/test_webhook","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 			},
// 			[]byte(`{"id":"a29148ce-878b-11eb-a518-83192db03b8d","type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","webhook_uri":"http://test.com/test_webhook","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
// 			if tt.r.WebhookURI != "" {
// 				mockReq.EXPECT().WMWebhookPOST("POST", tt.r.WebhookURI, dataTypeJSON, string(EventTypeRecordingStarted), tt.expectWebhook)
// 			}
// 			h.NotifyEvent(EventTypeRecordingStarted, tt.r.WebhookURI, tt.r)

// 			time.Sleep(time.Millisecond * 100)
// 		})
// 	}
// }

// func TestNotifyEventRecordingFinished(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	exchangeDelay := ""
// 	exchangeNotify := "bin-manager.call-manager.event"

// 	h := &notifyHandler{
// 		sock:           mockSock,
// 		reqHandler:     mockReq,
// 		exchangeDelay:  exchangeDelay,
// 		exchangeNotify: exchangeNotify,
// 	}

// 	type test struct {
// 		name          string
// 		r             *recording.Recording
// 		expectEvent   *rabbitmqhandler.Event
// 		expectWebhook []byte
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&recording.Recording{
// 				ID:          uuid.FromStringOrNil("d7edc1ec-8618-11eb-9740-9bc23366bed2"),
// 				UserID:      1,
// 				Type:        recording.TypeCall,
// 				ReferenceID: uuid.FromStringOrNil("dbb39734-8618-11eb-89c7-3f96da5df55e"),
// 				Status:      recording.StatusEnd,
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeRecordingFinished),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"d7edc1ec-8618-11eb-9740-9bc23366bed2","user_id":1,"type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","filename":"","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 			},
// 			[]byte{},
// 		},
// 		{
// 			"webhook uri",
// 			&recording.Recording{
// 				ID:          uuid.FromStringOrNil("7e886ff2-1070-11ec-ae24-6b3001e9028e"),
// 				UserID:      1,
// 				Type:        recording.TypeCall,
// 				ReferenceID: uuid.FromStringOrNil("dbb39734-8618-11eb-89c7-3f96da5df55e"),
// 				Status:      recording.StatusEnd,
// 				WebhookURI:  "test.com/webhook",
// 			},
// 			&rabbitmqhandler.Event{
// 				Type:      string(EventTypeRecordingFinished),
// 				Publisher: EventPublisher,
// 				DataType:  dataTypeJSON,
// 				Data:      []byte(`{"id":"7e886ff2-1070-11ec-ae24-6b3001e9028e","user_id":1,"type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","filename":"","webhook_uri":"test.com/webhook","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 			},
// 			[]byte(`{"id":"7e886ff2-1070-11ec-ae24-6b3001e9028e","type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","webhook_uri":"test.com/webhook","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
// 			if tt.r.WebhookURI != "" {
// 				mockReq.EXPECT().WMWebhookPOST("POST", tt.r.WebhookURI, dataTypeJSON, string(EventTypeRecordingFinished), tt.expectWebhook)
// 			}
// 			h.NotifyEvent(EventTypeRecordingFinished, tt.r.WebhookURI, tt.r)

// 			time.Sleep(time.Millisecond * 100)
// 		})
// 	}
// }
