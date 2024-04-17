package confbridgehandler

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/bridge"
)

func TestGenerateBridgeName(t *testing.T) {
	type test struct {
		name          string
		referenceType bridge.ReferenceType
		id            uuid.UUID
		expectName    string
	}

	tests := []test{
		{
			"Type unknown",
			bridge.ReferenceTypeUnknown,
			uuid.FromStringOrNil("3a3c10fc-934d-11ea-89ac-9fc52ba9880b"),
			"reference_type=unknown,reference_id=3a3c10fc-934d-11ea-89ac-9fc52ba9880b",
		},
		{
			"Type confbridge",
			bridge.ReferenceTypeConfbridge,
			uuid.FromStringOrNil("85d782a8-934d-11ea-afcc-db85d6e1a911"),
			"reference_type=confbridge,reference_id=85d782a8-934d-11ea-afcc-db85d6e1a911",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := generateBridgeName(tt.referenceType, tt.id)
			if res != tt.expectName {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectName, res)
			}
		})
	}
}
