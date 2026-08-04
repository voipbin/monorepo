package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_FormatTagIDsFilter(t *testing.T) {
	type test struct {
		name string
		ids  []uuid.UUID

		expectRes string
	}

	tests := []test{
		{
			name: "multiple ids",
			ids: []uuid.UUID{
				uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
				uuid.FromStringOrNil("4fc21d6c-0000-11ee-0000-000000000000"),
			},
			expectRes: "5d443cfe-0000-11ee-0000-000000000000,4fc21d6c-0000-11ee-0000-000000000000",
		},
		{
			name:      "empty slice",
			ids:       []uuid.UUID{},
			expectRes: "",
		},
		{
			name:      "nil slice",
			ids:       nil,
			expectRes: "",
		},
		{
			name: "nil uuid dropped",
			ids: []uuid.UUID{
				uuid.Nil,
				uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
			},
			expectRes: "5d443cfe-0000-11ee-0000-000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := FormatTagIDsFilter(tt.ids)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_ParseTagIDsFilter(t *testing.T) {
	type test struct {
		name string
		in   string

		expectRes []uuid.UUID
		expectErr bool
	}

	tests := []test{
		{
			name: "multiple ids",
			in:   "5d443cfe-0000-11ee-0000-000000000000,4fc21d6c-0000-11ee-0000-000000000000",
			expectRes: []uuid.UUID{
				uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
				uuid.FromStringOrNil("4fc21d6c-0000-11ee-0000-000000000000"),
			},
		},
		{
			name: "uppercase normalized to lowercase",
			in:   "5D443CFE-0000-11EE-0000-000000000000",
			expectRes: []uuid.UUID{
				uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
			},
		},
		{
			name: "whitespace trimmed",
			in:   " 5d443cfe-0000-11ee-0000-000000000000 , 4fc21d6c-0000-11ee-0000-000000000000 ",
			expectRes: []uuid.UUID{
				uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
				uuid.FromStringOrNil("4fc21d6c-0000-11ee-0000-000000000000"),
			},
		},
		{
			name:      "empty string means no filter",
			in:        "",
			expectRes: nil,
		},
		{
			name:      "comma only is an error",
			in:        ",",
			expectErr: true,
		},
		{
			name:      "whitespace only is an error",
			in:        "   ",
			expectErr: true,
		},
		{
			name:      "invalid uuid is an error",
			in:        "not-a-uuid",
			expectErr: true,
		},
		{
			name:      "nil uuid is an error",
			in:        "00000000-0000-0000-0000-000000000000",
			expectErr: true,
		},
		{
			name:      "too many ids is an error",
			in:        strings.Repeat("5d443cfe-0000-11ee-0000-000000000000,", maxTagIDsFilter+1) + "5d443cfe-0000-11ee-0000-000000000000",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseTagIDsFilter(tt.in)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok (res: %v)", res)
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TagIDsFilter_RoundTrip(t *testing.T) {
	ids := []uuid.UUID{
		uuid.FromStringOrNil("5d443cfe-0000-11ee-0000-000000000000"),
		uuid.FromStringOrNil("4fc21d6c-0000-11ee-0000-000000000000"),
	}

	formatted := FormatTagIDsFilter(ids)
	parsed, err := ParseTagIDsFilter(formatted)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(parsed, ids) {
		t.Errorf("Wrong match. expect: %v, got: %v", ids, parsed)
	}
}
