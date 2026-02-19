package externalmedia

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestExternalMediaStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	e := ExternalMedia{
		ID:              id,
		AsteriskID:      "asterisk-1",
		ChannelID:       "channel-123",
		BridgeID:        "bridge-456",
		PlaybackID:      "playback-789",
		ReferenceType:   ReferenceTypeCall,
		ReferenceID:     referenceID,
		Status:          StatusRunning,
		LocalIP:         "192.168.1.100",
		LocalPort:       10000,
		ExternalHost:    "external.example.com:20000",
		Encapsulation:   EncapsulationRTP,
		Transport:       TransportUDP,
		ConnectionType:  "client",
		Format:          "slin16",
		DirectionListen: DirectionIn,
		DirectionSpeak:  DirectionOut,
	}

	if e.ID != id {
		t.Errorf("ExternalMedia.ID = %v, expected %v", e.ID, id)
	}
	if e.AsteriskID != "asterisk-1" {
		t.Errorf("ExternalMedia.AsteriskID = %v, expected %v", e.AsteriskID, "asterisk-1")
	}
	if e.ChannelID != "channel-123" {
		t.Errorf("ExternalMedia.ChannelID = %v, expected %v", e.ChannelID, "channel-123")
	}
	if e.BridgeID != "bridge-456" {
		t.Errorf("ExternalMedia.BridgeID = %v, expected %v", e.BridgeID, "bridge-456")
	}
	if e.PlaybackID != "playback-789" {
		t.Errorf("ExternalMedia.PlaybackID = %v, expected %v", e.PlaybackID, "playback-789")
	}
	if e.ReferenceType != ReferenceTypeCall {
		t.Errorf("ExternalMedia.ReferenceType = %v, expected %v", e.ReferenceType, ReferenceTypeCall)
	}
	if e.ReferenceID != referenceID {
		t.Errorf("ExternalMedia.ReferenceID = %v, expected %v", e.ReferenceID, referenceID)
	}
	if e.Status != StatusRunning {
		t.Errorf("ExternalMedia.Status = %v, expected %v", e.Status, StatusRunning)
	}
	if e.LocalIP != "192.168.1.100" {
		t.Errorf("ExternalMedia.LocalIP = %v, expected %v", e.LocalIP, "192.168.1.100")
	}
	if e.LocalPort != 10000 {
		t.Errorf("ExternalMedia.LocalPort = %v, expected %v", e.LocalPort, 10000)
	}
	if e.ExternalHost != "external.example.com:20000" {
		t.Errorf("ExternalMedia.ExternalHost = %v, expected %v", e.ExternalHost, "external.example.com:20000")
	}
	if e.Encapsulation != EncapsulationRTP {
		t.Errorf("ExternalMedia.Encapsulation = %v, expected %v", e.Encapsulation, EncapsulationRTP)
	}
	if e.Transport != TransportUDP {
		t.Errorf("ExternalMedia.Transport = %v, expected %v", e.Transport, TransportUDP)
	}
	if e.ConnectionType != "client" {
		t.Errorf("ExternalMedia.ConnectionType = %v, expected %v", e.ConnectionType, "client")
	}
	if e.Format != "slin16" {
		t.Errorf("ExternalMedia.Format = %v, expected %v", e.Format, "slin16")
	}
	if e.DirectionListen != DirectionIn {
		t.Errorf("ExternalMedia.DirectionListen = %v, expected %v", e.DirectionListen, DirectionIn)
	}
	if e.DirectionSpeak != DirectionOut {
		t.Errorf("ExternalMedia.DirectionSpeak = %v, expected %v", e.DirectionSpeak, DirectionOut)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_confbridge", ReferenceTypeConfbridge, "confbridge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestEncapsulationConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Encapsulation
		expected string
	}{
		{"encapsulation_rtp", EncapsulationRTP, "rtp"},
		{"encapsulation_audiosocket", EncapsulationAudioSocket, "audiosocket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestTransportConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Transport
		expected string
	}{
		{"transport_udp", TransportUDP, "udp"},
		{"transport_tcp", TransportTCP, "tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{"direction_none", DirectionNone, ""},
		{"direction_both", DirectionBoth, "both"},
		{"direction_in", DirectionIn, "in"},
		{"direction_out", DirectionOut, "out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_running", StatusRunning, "running"},
		{"status_terminating", StatusTerminating, "terminating"},
		{"status_terminated", StatusTerminated, "terminated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
