package utilhandler

import "testing"

func Test_EmailIsValid(t *testing.T) {

	type test struct {
		name string

		email string

		expectRes bool
	}

	tests := []test{
		{
			name: "valid email",

			email:     "test@example.com",
			expectRes: true,
		},
		{
			name: "invalid email",

			email:     "invalid-email",
			expectRes: false,
		},
		{
			name:      "email with subdomain",
			email:     "test@sub.example.com",
			expectRes: true,
		},
		{
			name:      "email with plus sign",
			email:     "test+alias@example.com",
			expectRes: true,
		},
		{
			name:      "email with numbers",
			email:     "user123@example.com",
			expectRes: true,
		},
		{
			name:      "email with special characters",
			email:     "user.name+tag+sorting@example.com",
			expectRes: true,
		},
		{
			name:      "email with hyphen in domain",
			email:     "user@example-domain.com",
			expectRes: true,
		},
		{
			name:      "email with invalid domain",
			email:     "user@.com",
			expectRes: false,
		},
		{
			name:      "email with missing @ symbol",
			email:     "userexample.com",
			expectRes: false,
		},
		{
			name:      "email with consecutive dots",
			email:     "user..name@example.com",
			expectRes: false,
		},
		{
			name:      "email with trailing dot",
			email:     "user.@example.com",
			expectRes: false,
		},
		{
			name:      "email with spaces",
			email:     "user name@example.com",
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := EmailIsValid(tt.email)

			if tt.expectRes != res {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
