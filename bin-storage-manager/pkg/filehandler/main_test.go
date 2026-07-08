package filehandler

import (
	"os"
	"testing"
)

func Test_getFilename(t *testing.T) {

	type test struct {
		name string

		target string

		expectRes string
	}

	tests := []test{
		{
			"recording file",

			"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",

			"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getFilename(tt.target)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_filenameHash(t *testing.T) {
	type test struct {
		name string

		filenames []string

		expectRes string
	}

	tests := []test{
		{
			"recording file",

			[]string{
				"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
				"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_out.wav",
			},

			"tmp/0c44d476cdf0b43d377c82044155c14b4aba49bb.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := createZipFilepathHash(tt.filenames)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

// Test_NewFileHandler_MissingCredentials verifies that NewFileHandler fails
// fast with a non-nil error (not a silent nil interface) when
// GOOGLE_APPLICATION_CREDENTIALS is unset. Callers rely on this error to
// abort startup instead of continuing with a nil FileHandler that would
// panic on the first request.
func Test_NewFileHandler_MissingCredentials(t *testing.T) {
	origCredPath, hadCredPath := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if hadCredPath {
		defer func() { _ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", origCredPath) }()
	} else {
		defer func() { _ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS") }()
	}
	_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	res, err := NewFileHandler(nil, nil, nil, "test-project", "test-bucket-media", "test-bucket-tmp")
	if err == nil {
		t.Errorf("Wrong match. expect: non-nil error, got: nil error")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil FileHandler, got: %v", res)
	}
}
