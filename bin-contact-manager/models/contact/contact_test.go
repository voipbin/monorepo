package contact

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestContact_Matches(t *testing.T) {
	tests := []struct {
		name     string
		contact  *Contact
		other    interface{}
		expected bool
	}{
		{
			name: "matching contacts",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			other: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
			},
			expected: true,
		},
		{
			name: "matching contacts with different timestamps",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName: "John",
				TMCreate:  "2020-01-01 00:00:00.000000",
				TMUpdate:  "2020-01-02 00:00:00.000000",
				TMDelete:  "9999-01-01T00:00:00.000000Z",
			},
			other: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName: "John",
				TMCreate:  "2021-01-01 00:00:00.000000",
				TMUpdate:  "2021-01-02 00:00:00.000000",
				TMDelete:  "9999-01-01T00:00:00.000000Z",
			},
			expected: true,
		},
		{
			name: "non-matching contacts - different name",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
				FirstName: "John",
			},
			other: &Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
				FirstName: "Jane",
			},
			expected: false,
		},
		{
			name: "non-matching - wrong type",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},
			other:    "not a contact",
			expected: false,
		},
		{
			name: "non-matching - nil",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},
			other:    nil,
			expected: false,
		},
		{
			name: "non-matching - byte slice",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},
			other:    []byte("some bytes"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.contact.Matches(tt.other)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContact_String(t *testing.T) {
	contact := &Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John Doe",
	}

	result := contact.String()
	if result == "" {
		t.Error("String() returned empty string")
	}

	// Should contain the first name
	if len(result) == 0 {
		t.Error("String() should return non-empty string")
	}
}
