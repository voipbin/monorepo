package processmanager

import "testing"

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{"tcpdump allowed", "tcpdump", false},
		{"empty rejected", "", true},
		{"bash rejected", "bash", true},
		{"sh rejected", "sh", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommand(%q) error = %v, wantErr %v", tt.cmd, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeParameters(t *testing.T) {
	tests := []struct {
		name    string
		params  []string
		wantErr bool
	}{
		{"clean BPF filter", []string{"udp port 30000 or udp port 30002"}, false},
		{"semicolon rejected", []string{"udp port 30000; rm -rf /"}, true},
		{"pipe rejected", []string{"udp port 30000 | cat"}, true},
		{"backtick rejected", []string{"`whoami`"}, true},
		{"dollar rejected", []string{"$HOME"}, true},
		{"ampersand rejected", []string{"udp port 30000 && ls"}, true},
		{"empty list ok", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizeParameters(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizeParameters(%v) error = %v, wantErr %v", tt.params, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWritePath(t *testing.T) {
	tests := []struct {
		name    string
		params  []string
		wantErr bool
	}{
		{"valid path", []string{"-i", "eth0", "-w", "/tmp/00000000-0000-0000-0000-000000000001.pcap"}, false},
		{"no -w flag", []string{"-i", "eth0"}, true},
		{"path outside /tmp", []string{"-w", "/etc/evil.pcap"}, true},
		{"not pcap extension", []string{"-w", "/tmp/00000000-0000-0000-0000-000000000001.txt"}, true},
		{"not uuid filename", []string{"-w", "/tmp/evil.pcap"}, true},
		{"path traversal", []string{"-w", "/tmp/../etc/00000000-0000-0000-0000-000000000001.pcap"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWritePath(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWritePath(%v) error = %v, wantErr %v", tt.params, err, tt.wantErr)
			}
		})
	}
}
