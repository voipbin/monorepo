package user

import "testing"

func TestHasPermission(t *testing.T) {
	type test struct {
		name       string
		user       User
		permission Permission
		expectRes  bool
	}

	tests := []test{
		{
			"normal",
			User{
				Username:   "test",
				Permission: PermissionAdmin,
			},
			PermissionAdmin,
			true,
		},
		{
			"permission is number",
			User{
				Username:   "test",
				Permission: 1,
			},
			PermissionAdmin,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.user.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
