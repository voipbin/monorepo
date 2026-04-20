package main

import (
	"net"
	"testing"
)

func Test_getKamailioID(t *testing.T) {
	type test struct {
		name      string
		ifaceName string
		wantErr   bool
	}

	// Find a real interface with a MAC address for the happy path.
	validIface := ""
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 {
			validIface = iface.Name
			break
		}
	}

	tests := []test{
		{
			name:      "interface not found",
			ifaceName: "nonexistent999",
			wantErr:   true,
		},
	}

	if validIface != "" {
		tests = append(tests, test{
			name:      "valid interface with MAC",
			ifaceName: validIface,
			wantErr:   false,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getKamailioID(tt.ifaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKamailioID(%q) error = %v, wantErr %v", tt.ifaceName, err, tt.wantErr)
			}
			if !tt.wantErr && res == "" {
				t.Errorf("getKamailioID(%q) returned empty string, expected MAC address", tt.ifaceName)
			}
		})
	}
}
