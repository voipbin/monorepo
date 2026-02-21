package externalmedia

// Field represents a database field name for ExternalMedia
type Field string

const (
	FieldID         Field = "id"          // id
	FieldAsteriskID Field = "asterisk_id" // asterisk_id
	FieldChannelID  Field = "channel_id"  // external media channel id
	FieldBridgeID   Field = "bridge_id"   // bridge id for external media snoop channel

	FieldPlaybackID Field = "playback_id" // playback id for reference channel's silence streaming out

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldStatus Field = "status" // status of the external media

	FieldLocalIP   Field = "local_ip"   // local_ip
	FieldLocalPort Field = "local_port" // local_port

	FieldExternalHost    Field = "external_host"    // external_host
	FieldEncapsulation   Field = "encapsulation"    // payload encapsulation protocol
	FieldTransport       Field = "transport"        // transport
	FieldTransportData   Field = "transport_data"   // transport-specific data
	FieldConnectionType  Field = "connection_type"  // connection_type
	FieldFormat          Field = "format"           // format
	FieldDirectionListen Field = "direction_listen" // direction of the external media channel
	FieldDirectionSpeak  Field = "direction_speak"  // direction of the external media channel

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
