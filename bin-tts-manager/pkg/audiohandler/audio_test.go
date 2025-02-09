package audiohandler

// func Test_getVoiceName(t *testing.T) {

// 	type test struct {
// 		name string

// 		lang   string
// 		gender tts.Gender

// 		expectRes string
// 	}

// 	tests := []test{
// 		{
// 			"en-US female",

// 			"en-US",
// 			tts.GenderFemale,

// 			"en-US-Wavenet-F",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := &audioHandler{}

// 			res := h.getVoiceName(tt.lang, tt.gender)
// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
