package chathandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_sortUUIDs(t *testing.T) {

	tests := []struct {
		name string

		uuids []uuid.UUID

		expectRes []uuid.UUID
	}{
		{
			name: "normal",

			uuids: []uuid.UUID{
				uuid.FromStringOrNil("8fa6e36c-412e-11ef-84f0-f3338545bce7"),
				uuid.FromStringOrNil("8f28081c-412e-11ef-b2d1-73dda496575b"),
				uuid.FromStringOrNil("8f8231b6-412e-11ef-b704-df27abb8a1da"),
			},
			expectRes: []uuid.UUID{
				uuid.FromStringOrNil("8f28081c-412e-11ef-b2d1-73dda496575b"),
				uuid.FromStringOrNil("8f8231b6-412e-11ef-b704-df27abb8a1da"),
				uuid.FromStringOrNil("8fa6e36c-412e-11ef-84f0-f3338545bce7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := sortUUIDs(tt.uuids)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_convertUUIDsToCommaSeparatedString(t *testing.T) {

	tests := []struct {
		name string

		uuids []uuid.UUID

		expectRes string
	}{
		{
			name: "normal",

			uuids: []uuid.UUID{
				uuid.FromStringOrNil("8fa6e36c-412e-11ef-84f0-f3338545bce7"),
				uuid.FromStringOrNil("8f28081c-412e-11ef-b2d1-73dda496575b"),
				uuid.FromStringOrNil("8f8231b6-412e-11ef-b704-df27abb8a1da"),
			},
			expectRes: "8fa6e36c-412e-11ef-84f0-f3338545bce7,8f28081c-412e-11ef-b2d1-73dda496575b,8f8231b6-412e-11ef-b704-df27abb8a1da",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := convertUUIDsToCommaSeparatedString(tt.uuids)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
