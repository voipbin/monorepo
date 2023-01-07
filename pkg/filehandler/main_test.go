package filehandler

import (
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
