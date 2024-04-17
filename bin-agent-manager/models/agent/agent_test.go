package agent

import "testing"

func Test_HasPermission(t *testing.T) {
	type test struct {
		name       string
		agent      Agent
		permission Permission
		expectRes  bool
	}

	tests := []test{
		{
			"normal",
			Agent{
				Username:   "test",
				Permission: PermissionProjectSuperAdmin,
			},
			PermissionProjectSuperAdmin,
			true,
		},
		{
			"permission is number",
			Agent{
				Username:   "test",
				Permission: 1,
			},
			PermissionProjectSuperAdmin,
			true,
		},
		{
			"has super admin",
			Agent{
				Username:   "test",
				Permission: PermissionProjectSuperAdmin | PermissionCustomerAdmin,
			},
			PermissionProjectSuperAdmin,
			true,
		},
		{
			"has admin",
			Agent{
				Username:   "test",
				Permission: PermissionProjectSuperAdmin | PermissionCustomerAdmin,
			},
			PermissionCustomerAdmin,
			true,
		},
		{
			"has no superadmin",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerAdmin,
			},
			PermissionProjectSuperAdmin,
			false,
		},
		{
			"has no manager",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerAdmin,
			},
			PermissionCustomerManager,
			false,
		},
		{
			"has 2 permissions and wants 1 permission",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerAdmin | PermissionCustomerManager,
			},
			PermissionCustomerManager,
			true,
		},
		{
			"has 2 permissions but has 1 right permission",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerAdmin | PermissionCustomerManager,
			},
			PermissionCustomerManager | PermissionProjectSuperAdmin,
			true,
		},
		{
			"has 2 permissions but has no right permission",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerAdmin | PermissionCustomerManager,
			},
			PermissionProjectSuperAdmin,
			false,
		},
		{
			"all permission",
			Agent{
				Username:   "test",
				Permission: PermissionCustomerManager,
			},
			PermissionAll,
			true,
		},
		{
			"all permission but agent has no permission",
			Agent{
				Username:   "test",
				Permission: PermissionNone,
			},
			PermissionAll,
			true,
		},
		{
			"none permission and agent has no permission",
			Agent{
				Username:   "test",
				Permission: PermissionNone,
			},
			PermissionNone,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.agent.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
