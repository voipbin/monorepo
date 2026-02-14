package buckethandler

import (
	"testing"
)

func Test_NewBucketHandler(t *testing.T) {
	tests := []struct {
		name                     string
		osMediaBucketDirectory   string
		osAddress                string
		expectedOsBucketDir      string
		expectedOsLocalAddress   string
	}{
		{
			name:                    "normal",
			osMediaBucketDirectory:  "/shared-data",
			osAddress:               "10.96.0.112",
			expectedOsBucketDir:     "/shared-data",
			expectedOsLocalAddress:  "10-96-0-112.bin-manager.pod.cluster.local",
		},
		{
			name:                    "different address format",
			osMediaBucketDirectory:  "/var/media",
			osAddress:               "192.168.1.100",
			expectedOsBucketDir:     "/var/media",
			expectedOsLocalAddress:  "192-168-1-100.bin-manager.pod.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewBucketHandler(tt.osMediaBucketDirectory, tt.osAddress)
			if h == nil {
				t.Fatal("expected handler, got nil")
			}

			bh, ok := h.(*bucketHandler)
			if !ok {
				t.Fatal("handler is not bucketHandler type")
			}

			if bh.osBucketDirectory != tt.expectedOsBucketDir {
				t.Errorf("osBucketDirectory: expected %s, got %s", tt.expectedOsBucketDir, bh.osBucketDirectory)
			}

			if bh.osLocalAddress != tt.expectedOsLocalAddress {
				t.Errorf("osLocalAddress: expected %s, got %s", tt.expectedOsLocalAddress, bh.osLocalAddress)
			}
		})
	}
}
