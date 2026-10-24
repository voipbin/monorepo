package numberhandlertwilio

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

func Test_NewNumberHandler(t *testing.T) {
	h := NewNumberHandler(nil, nil, "test-sid", "test-token")

	res, ok := h.(*numberHandlerTwilio)
	if !ok {
		t.Fatalf("Wrong type. got: %T", h)
	}
	if res.sid != "test-sid" || res.token != "test-token" {
		t.Errorf("Wrong injected values. got: %q, %q", res.sid, res.token)
	}
	if res.requestExternal == nil {
		t.Errorf("Wrong match. expect: request external, got: nil")
	}
}

func Test_unimplemented(t *testing.T) {
	h := &numberHandlerTwilio{}

	tests := []struct {
		name   string
		call   func() error
		expect string
	}{
		{
			name: "GetAvailableNumbers",
			call: func() error {
				res, err := h.GetAvailableNumbers("US", 5)
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return err
			},
			expect: "unimplemented: GetAvailableNumbers",
		},
		{
			name: "CreateNumber",
			call: func() error {
				res, err := h.CreateNumber(uuid.FromStringOrNil("6f1c5f1e-0c1d-11ef-9b6e-8f3c2d4a7b22"), "+15550100001", uuid.Nil, "name", "detail")
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return err
			},
			expect: "unimplemented: CreateNumber",
		},
		{
			name: "ReleaseNumber",
			call: func() error {
				return h.ReleaseNumber(context.Background(), &number.Number{})
			},
			expect: "unimplemented: ReleaseNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil || !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong error. expect: %s, got: %v", tt.expect, err)
			}
		})
	}
}
