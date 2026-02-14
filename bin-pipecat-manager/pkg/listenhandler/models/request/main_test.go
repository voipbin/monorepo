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
			if tt.p.PageSize != tt.p.PageSize {
				t.Errorf("PageSize mismatch")
			}
			if tt.p.PageToken != tt.p.PageToken {
				t.Errorf("PageToken mismatch")
			}
		})
	}
}
