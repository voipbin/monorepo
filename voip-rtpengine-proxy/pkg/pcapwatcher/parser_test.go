package pcapwatcher

import (
	"testing"
)

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
		filename string
		want     string
	}{
		{
			name:     "pcap file",
			filename: "abc123-tag1-ssrc1.pcap",
			want:     "rtp-recordings/abc123-tag1-ssrc1.pcap",
		},
		{
			name:     "uuid-style filename",
			filename: "5e9545dd-84cf-4105-86db-b78b10a05ad7-c7ca1c4d60c937a7.pcap",
			want:     "rtp-recordings/5e9545dd-84cf-4105-86db-b78b10a05ad7-c7ca1c4d60c937a7.pcap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildObjectPath(tt.filename)
			if got != tt.want {
				t.Errorf("buildObjectPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
