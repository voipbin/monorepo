package linehandler

// note: this test changes the actual test account's webhook address on line.
// need to be careful to test this.
// func Test_Setup(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		account *account.Account
// 	}{
// 		{
// 			name: "normal",

// 			account: &account.Account{
// 				ID:     uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
// 				Secret: "ba5f0575d826d5b4a052a43145ef1391",
// 				// Token:      "<your line token>",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := lineHandler{}
// 			ctx := context.Background()

// 			if err := h.Setup(ctx, tt.account); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
