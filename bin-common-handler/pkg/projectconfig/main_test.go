package projectconfig

import (
	"os"
	"testing"
)

func Test_getEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_GETENV_UNSET",
			defaultValue: "default_value",
			setEnv:       false,
			expected:     "default_value",
		},
		{
			name:         "returns env value when set",
			key:          "TEST_GETENV_SET",
			defaultValue: "default_value",
			envValue:     "env_value",
			setEnv:       true,
			expected:     "env_value",
		},
		{
			name:         "returns default when env is empty string",
			key:          "TEST_GETENV_EMPTY",
			defaultValue: "default_value",
			envValue:     "",
			setEnv:       true,
			expected:     "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			_ = os.Unsetenv(tt.key)

			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func Test_load(t *testing.T) {
	tests := []struct {
		name                     string
		envBaseDomain            string
		envBucketName            string
		expectedBaseDomain       string
		expectedDomainConference string
		expectedDomainPSTN       string
		expectedDomainTrunk      string
		expectedDomainRegistrar  string
		expectedBucketName       string
	}{
		{
			name:                     "uses defaults when no env vars set",
			envBaseDomain:            "",
			envBucketName:            "",
			expectedBaseDomain:       "voipbin.net",
			expectedDomainConference: "conference.voipbin.net",
			expectedDomainPSTN:       "pstn.voipbin.net",
			expectedDomainTrunk:      ".trunk.voipbin.net",
			expectedDomainRegistrar:  ".registrar.voipbin.net",
			expectedBucketName:       "voipbin-voip-media-bucket-europe-west4",
		},
		{
			name:                     "uses custom domain when env var set",
			envBaseDomain:            "example.com",
			envBucketName:            "",
			expectedBaseDomain:       "example.com",
			expectedDomainConference: "conference.example.com",
			expectedDomainPSTN:       "pstn.example.com",
			expectedDomainTrunk:      ".trunk.example.com",
			expectedDomainRegistrar:  ".registrar.example.com",
			expectedBucketName:       "voipbin-voip-media-bucket-europe-west4",
		},
		{
			name:                     "uses custom bucket when env var set",
			envBaseDomain:            "",
			envBucketName:            "custom-bucket-name",
			expectedBaseDomain:       "voipbin.net",
			expectedDomainConference: "conference.voipbin.net",
			expectedDomainPSTN:       "pstn.voipbin.net",
			expectedDomainTrunk:      ".trunk.voipbin.net",
			expectedDomainRegistrar:  ".registrar.voipbin.net",
			expectedBucketName:       "custom-bucket-name",
		},
		{
			name:                     "uses all custom values when both env vars set",
			envBaseDomain:            "localhost",
			envBucketName:            "local-bucket",
			expectedBaseDomain:       "localhost",
			expectedDomainConference: "conference.localhost",
			expectedDomainPSTN:       "pstn.localhost",
			expectedDomainTrunk:      ".trunk.localhost",
			expectedDomainRegistrar:  ".registrar.localhost",
			expectedBucketName:       "local-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			_ = os.Unsetenv("PROJECT_BASE_DOMAIN")
			_ = os.Unsetenv("PROJECT_BUCKET_NAME")

			if tt.envBaseDomain != "" {
				t.Setenv("PROJECT_BASE_DOMAIN", tt.envBaseDomain)
			}

			if tt.envBucketName != "" {
				t.Setenv("PROJECT_BUCKET_NAME", tt.envBucketName)
			}

			result := load()

			if result.ProjectBaseDomain != tt.expectedBaseDomain {
				t.Errorf("ProjectBaseDomain = %v, want %v", result.ProjectBaseDomain, tt.expectedBaseDomain)
			}
			if result.DomainConference != tt.expectedDomainConference {
				t.Errorf("DomainConference = %v, want %v", result.DomainConference, tt.expectedDomainConference)
			}
			if result.DomainPSTN != tt.expectedDomainPSTN {
				t.Errorf("DomainPSTN = %v, want %v", result.DomainPSTN, tt.expectedDomainPSTN)
			}
			if result.DomainTrunkSuffix != tt.expectedDomainTrunk {
				t.Errorf("DomainTrunkSuffix = %v, want %v", result.DomainTrunkSuffix, tt.expectedDomainTrunk)
			}
			if result.DomainRegistrarSuffix != tt.expectedDomainRegistrar {
				t.Errorf("DomainRegistrarSuffix = %v, want %v", result.DomainRegistrarSuffix, tt.expectedDomainRegistrar)
			}
			if result.ProjectBucketName != tt.expectedBucketName {
				t.Errorf("ProjectBucketName = %v, want %v", result.ProjectBucketName, tt.expectedBucketName)
			}
		})
	}
}
