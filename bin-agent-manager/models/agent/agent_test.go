package agent

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Agent implements eventtopic.SubscriptionIdentifier explicitly (VOIP-1419). The assertion
// pins the POINTER type: the event data reaches notifyhandler as a pointer and the interface
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would not satisfy
// the interface.
var _ eventtopic.SubscriptionIdentifier = (*Agent)(nil)

func Test_EventSubscriptionID(t *testing.T) {
	agentID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	directID := uuid.Must(uuid.NewV4())

	a := &Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		DirectID: directID,
	}

	res := a.EventSubscriptionID()
	if res != agentID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", agentID.String(), res)
	}

	// The subscription address is the agent's OWN id -- not any other uuid the payload
	// happens to carry. These assertions are the mutation check: an implementation that
	// returns the wrong field fails here.
	if res == customerID.String() {
		t.Errorf("Agent must not be addressed by its customer id. got: %s", res)
	}
	if res == directID.String() {
		t.Errorf("Agent must not be addressed by its direct id. got: %s", res)
	}
}

func Test_EventSubscriptionID_zeroValue(t *testing.T) {
	a := &Agent{}

	if res := a.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}

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
