package streaminghandler

import (
	"testing"
)

func Test_validateProvider(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectedRes STTProvider
		expectError bool
	}{
		{
			name:        "uppercase GCP",
			input:       "GCP",
			expectedRes: STTProviderGCP,
			expectError: false,
		},
		{
			name:        "lowercase gcp",
			input:       "gcp",
			expectedRes: STTProviderGCP,
			expectError: false,
		},
		{
			name:        "mixed case Gcp",
			input:       "Gcp",
			expectedRes: STTProviderGCP,
			expectError: false,
		},
		{
			name:        "uppercase AWS",
			input:       "AWS",
			expectedRes: STTProviderAWS,
			expectError: false,
		},
		{
			name:        "lowercase aws",
			input:       "aws",
			expectedRes: STTProviderAWS,
			expectError: false,
		},
		{
			name:        "mixed case Aws",
			input:       "Aws",
			expectedRes: STTProviderAWS,
			expectError: false,
		},
		{
			name:        "with leading spaces",
			input:       "  GCP",
			expectedRes: STTProviderGCP,
			expectError: false,
		},
		{
			name:        "with trailing spaces",
			input:       "AWS  ",
			expectedRes: STTProviderAWS,
			expectError: false,
		},
		{
			name:        "with surrounding spaces",
			input:       "  gcp  ",
			expectedRes: STTProviderGCP,
			expectError: false,
		},
		{
			name:        "invalid provider",
			input:       "AZURE",
			expectedRes: "",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectedRes: "",
			expectError: true,
		},
		{
			name:        "invalid lowercase",
			input:       "azure",
			expectedRes: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateProvider(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tc.input, err)
				}
				if result != tc.expectedRes {
					t.Errorf("Expected provider '%s' for input '%s', got '%s'", tc.expectedRes, tc.input, result)
				}
			}
		})
	}
}
