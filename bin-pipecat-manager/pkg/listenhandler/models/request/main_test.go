package request

import "testing"

func TestPagination_Struct(t *testing.T) {
	tests := []struct {
		name string
		p    Pagination
	}{
		{
			name: "full pagination",
			p: Pagination{
				PageSize:  100,
				PageToken: "token-123",
			},
		},
		{
			name: "empty pagination",
			p:    Pagination{},
		},
		{
			name: "only page size",
			p: Pagination{
				PageSize: 50,
			},
		},
		{
			name: "only page token",
			p: Pagination{
				PageToken: "token-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure the Pagination struct is constructable with the given
			// fields. The struct has no behavior to validate beyond field
			// presence, so just round-trip the value.
			_ = tt.p
		})
	}
}
