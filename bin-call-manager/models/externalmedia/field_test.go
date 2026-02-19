package externalmedia

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{"field_id", FieldID, "id"},
		{"field_asterisk_id", FieldAsteriskID, "asterisk_id"},
		{"field_channel_id", FieldChannelID, "channel_id"},
		{"field_bridge_id", FieldBridgeID, "bridge_id"},
		{"field_playback_id", FieldPlaybackID, "playback_id"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_status", FieldStatus, "status"},
		{"field_type", FieldType, "type"},
		{"field_local_ip", FieldLocalIP, "local_ip"},
		{"field_local_port", FieldLocalPort, "local_port"},
		{"field_external_host", FieldExternalHost, "external_host"},
		{"field_encapsulation", FieldEncapsulation, "encapsulation"},
		{"field_transport", FieldTransport, "transport"},
		{"field_connection_type", FieldConnectionType, "connection_type"},
		{"field_format", FieldFormat, "format"},
		{"field_direction_listen", FieldDirectionListen, "direction_listen"},
		{"field_direction_speak", FieldDirectionSpeak, "direction_speak"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
		{"field_tm_delete", FieldTMDelete, "tm_delete"},
		{"field_deleted", FieldDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
