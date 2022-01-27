package customer

import (
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

func TestHasPermission(t *testing.T) {
	type test struct {
		name string

		customer   Customer
		permission uuid.UUID
		expectRes  bool
	}

	tests := []test{
		{
			"has admin permission",
			Customer{
				Username: "test",
				PermissionIDs: []uuid.UUID{
					permission.PermissionAdmin.ID,
				},
			},
			permission.PermissionAdmin.ID,
			true,
		},
		{
			"check admin permission bu has no admin permission",
			Customer{
				Username:      "test",
				PermissionIDs: []uuid.UUID{},
			},
			permission.PermissionAdmin.ID,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.customer.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
