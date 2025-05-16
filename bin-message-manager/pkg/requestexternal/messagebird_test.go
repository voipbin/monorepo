package requestexternal

// func Test_MessagebirdSendMessage(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		sender       string
// 		destinations []string
// 		text         string

// 		expectRes *messagebird.Message
// 	}{
// 		{
// 			"normal",

// 			"+821021656521",
// 			[]string{
// 				"+821021656521",
// 			},
// 			"Hello, this is test message. tteesstt",

// 			&messagebird.Message{
// 				Direction:  "mt",
// 				Type:       "sms",
// 				Originator: "+821021656521",
// 				Body:       "Hello, this is test message. tteesstt",
// 				DataCoding: "plain",

// 				Recipients: messagebird.RecipientStruct{
// 					TotalCount:               1,
// 					TotalSentCount:           1,
// 					TotalDeliveredCount:      0,
// 					TotalDeliveryFailedCount: 0,
// 					Items: []messagebird.Recipient{
// 						{
// 							Recipient:        821021656521,
// 							Status:           "sent",
// 							MessagePartCount: 1,
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

// 			h := &requestExternal{
// 				authtokenMessagebird: "4wuGDgAYlgYqB8RoYWbQ4HlwL", // test api key. does not send actual message
// 			}
// 			ctx := context.Background()

// 			res, err := h.MessagebirdSendMessage(ctx, tt.sender, tt.destinations, tt.text)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			tt.expectRes.ID = res.ID
// 			tt.expectRes.Href = res.Href
// 			tt.expectRes.Gateway = res.Gateway
// 			tt.expectRes.MClass = res.MClass
// 			tt.expectRes.Recipients.Items[0].StatusDatetime = res.Recipients.Items[0].StatusDatetime
// 			tt.expectRes.CreatedDatetime = res.CreatedDatetime
// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
