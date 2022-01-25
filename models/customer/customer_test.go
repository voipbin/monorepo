package customer

// func TestHasPermission(t *testing.T) {
// 	type test struct {
// 		name       string
// 		user       Customer
// 		permission Permission
// 		expectRes  bool
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			Customer{
// 				Username:   "test",
// 			},
// 			PermissionAdmin,
// 			true,
// 		},
// 		{
// 			"permission is number",
// 			Customer{
// 				Username:   "test",
// 			},
// 			PermissionAdmin,
// 			true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			res := tt.user.HasPermission(tt.permission)
// 			if res != tt.expectRes {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
