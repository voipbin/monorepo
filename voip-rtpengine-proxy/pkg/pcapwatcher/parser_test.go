package pcapwatcher

import (
	"testing"
)

func TestExtractCallIDFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "standard format",
			filename: "abc123def-randomtag",
			want:     "abc123def",
		},
		{
			name:     "uuid-style call-id",
			filename: "550e8400-e29b-41d4-a716-446655440000-tag123",
			want:     "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "no tag separator",
			filename: "simplecallid",
			want:     "simplecallid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCallIDFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("extractCallIDFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestParsePcapPathFromMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "standard metadata",
			content: "/var/spool/rtpengine/pcap/abc123-tag-ssrc.pcap\n\nSDP mode: offer\n",
			want:    "/var/spool/rtpengine/pcap/abc123-tag-ssrc.pcap",
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
		{
			name:    "path only",
			content: "/var/spool/rtpengine/pcap/test.pcap\n",
			want:    "/var/spool/rtpengine/pcap/test.pcap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePcapPathFromMetadata(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePcapPathFromMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePcapPathFromMetadata() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildObjectPath(t *testing.T) {
	tests := []struct {
		name     string
		callID   string
		filename string
		want     string
	}{
		{
			name:     "normal call",
			callID:   "abc123",
			filename: "abc123-tag1-ssrc1.pcap",
			want:     "rtp-recordings/abc123/abc123-tag1-ssrc1.pcap",
		},
		{
			name:     "call-id with special chars",
			callID:   "call@host",
			filename: "call@host-tag-ssrc.pcap",
			want:     "rtp-recordings/call@host/call@host-tag-ssrc.pcap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildObjectPath(tt.callID, tt.filename)
			if got != tt.want {
				t.Errorf("buildObjectPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
