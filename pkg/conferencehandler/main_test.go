package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
)

func TestGenerateBridgeName(t *testing.T) {
	type test struct {
		name       string
		confType   conference.Type
		id         uuid.UUID
		joining    bool
		expectName string
	}

	tests := []test{
		{
			"Type none",
			conference.TypeNone,
			uuid.FromStringOrNil("3a3c10fc-934d-11ea-89ac-9fc52ba9880b"),
			true,
			"conference_type=,conference_id=3a3c10fc-934d-11ea-89ac-9fc52ba9880b,join=true",
		},
		{
			"Type conference",
			conference.TypeConference,
			uuid.FromStringOrNil("85d782a8-934d-11ea-afcc-db85d6e1a911"),
			false,
			"conference_type=conference,conference_id=85d782a8-934d-11ea-afcc-db85d6e1a911,join=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := generateBridgeName(tt.confType, tt.id, tt.joining)
			if res != tt.expectName {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectName, res)
			}
		})
	}
}
