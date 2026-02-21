package utilhandler

import (
	"testing"
)

func Test_HashSHA256Hex(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "known vector",
			input:  "hello",
			expect: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:   "empty string",
			input:  "",
			expect: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:   "token format",
			input:  "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW",
			expect: "", // just verify length is 64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := HashSHA256Hex(tt.input)
			if len(res) != 64 {
				t.Errorf("Expected 64 char hex string, got %d chars", len(res))
			}
			if tt.expect != "" && res != tt.expect {
				t.Errorf("Wrong hash.\nexpect: %s\ngot:    %s", tt.expect, res)
			}
		})
	}
}

func Test_HashSHA256Hex_Deterministic(t *testing.T) {
	input := "vb_testtoken123456789"
	hash1 := HashSHA256Hex(input)
	hash2 := HashSHA256Hex(input)
	if hash1 != hash2 {
		t.Errorf("Hash is not deterministic. got %s and %s", hash1, hash2)
	}
}
