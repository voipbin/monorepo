package common

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_ParseSIPURI(t *testing.T) {

	type test struct {
		name string

		uri string

		expectResUUID uuid.UUID
		expectResExt  string
	}

	tests := []test{
		{
			"normal",

			"test11@1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",

			uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
			"test11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			customerID, ext, err := ParseSIPURI(tt.uri)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if customerID != tt.expectResUUID {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResUUID, customerID)
			}

			if ext != tt.expectResExt {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResExt, ext)
			}

		})
	}
}
