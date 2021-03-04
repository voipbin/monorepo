package conference

// func TestNewConference(t *testing.T) {
// 	type test struct {
// 		name             string
// 		conferenceType   Type
// 		bridgeID         string
// 		reqConf          *Conference
// 		expectConference *Conference
// 	}

// 	tests := []test{
// 		{
// 			"normal echo",
// 			TypeConference,
// 			"c9e43a42-9bf7-11ea-b110-bbb4c8d9c1de",
// 			&Conference{
// 				Type:   TypeConference,
// 				Name:   "test conference",
// 				Detail: "simple conference for test",
// 			},
// 			&Conference{
// 				ID:       uuid.FromStringOrNil("54c3e73e-9bfd-11ea-8bb8-7fc0647db6b5"),
// 				Type:     TypeConference,
// 				BridgeID: "c9e43a42-9bf7-11ea-b110-bbb4c8d9c1de",
// 				Name:     "test conference",
// 				Detail:   "simple conference for test",
// 				CallIDs:  []uuid.UUID{},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			cf := NewConference(tt.expectConference.ID, tt.conferenceType, tt.bridgeID, tt.reqConf)

// 			tt.expectConference.ID = cf.ID
// 			if reflect.DeepEqual(cf, tt.expectConference) != true {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot:%v\n", tt.expectConference, cf)
// 			}
// 		})
// 	}
// }
