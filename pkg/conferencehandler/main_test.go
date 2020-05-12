package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func TestGenerateBridgeName(t *testing.T) {
	type test struct {
		name       string
		confType   conference.Type
		id         uuid.UUID
		expectName string
	}

	tests := []test{
		{
			"Type none",
			conference.TypeNone,
			uuid.FromStringOrNil("3a3c10fc-934d-11ea-89ac-9fc52ba9880b"),
			"conference_type=,conference_id=3a3c10fc-934d-11ea-89ac-9fc52ba9880b",
		},
		{
			"Type echo",
			conference.TypeEcho,
			uuid.FromStringOrNil("e795ae9e-934c-11ea-90b5-af626e0c8e93"),
			"conference_type=echo,conference_id=e795ae9e-934c-11ea-90b5-af626e0c8e93",
		},
		{
			"Type transfer",
			conference.TypeTransfer,
			uuid.FromStringOrNil("721e7aa0-934d-11ea-bb8a-07680a5af6dc"),
			"conference_type=transfer,conference_id=721e7aa0-934d-11ea-bb8a-07680a5af6dc",
		},
		{
			"Type transfer",
			conference.TypeConference,
			uuid.FromStringOrNil("85d782a8-934d-11ea-afcc-db85d6e1a911"),
			"conference_type=conference,conference_id=85d782a8-934d-11ea-afcc-db85d6e1a911",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := generateBridgeName(tt.confType, tt.id)
			if res != tt.expectName {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectName, res)
			}
		})
	}
}
