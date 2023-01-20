package transcribehandler

// func TestTranscribeStreamingsHandle(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	mockDB := dbhandler.NewMockDBHandler(mc)
// 	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 	mockGoogle := transcripthandler.NewMockTranscriptHandler(mc)

// 	h := &transcribeHandler{
// 		reqHandler:        mockReq,
// 		db:                mockDB,
// 		notifyHandler:     mockNotify,
// 		transcriptHandler: mockGoogle,

// 		transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
// 	}

// 	tests := []struct {
// 		name string

// 		loopCount int
// 	}{
// 		{
// 			"normal",

// 			1000000,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			for i := 0; i < tt.loopCount; i++ {
// 				go func() {
// 					id := uuid.Must(uuid.NewV4())
// 					h.addTranscribeStreamings(id, []*streaming.Streaming{})
// 					tmp := h.getTranscribeStreamings(id)
// 					if tmp == nil {
// 						t.Errorf("Wrong match. expect: not nil, got: nil")
// 					}
// 					h.deleteTranscribeStreamings(id)
// 				}()
// 			}
// 		})
// 	}
// }
