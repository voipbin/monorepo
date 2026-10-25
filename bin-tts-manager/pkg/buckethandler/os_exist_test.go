package buckethandler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func Test_OSFileExist(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "sample.wav")
	if err := os.WriteFile(existing, []byte("data"), 0o600); err != nil {
		t.Fatalf("could not create the fixture: %v", err)
	}

	tests := []struct {
		name string

		target string

		expectRes bool
	}{
		{name: "existing file", target: existing, expectRes: true},
		{name: "existing directory", target: dir, expectRes: true},
		{name: "missing file", target: filepath.Join(dir, "missing.wav"), expectRes: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &bucketHandler{osBucketDirectory: dir}

			if got := h.OSFileExist(context.Background(), tt.target); got != tt.expectRes {
				t.Errorf("Wrong result. expect: %v, got: %v", tt.expectRes, got)
			}
		})
	}
}
