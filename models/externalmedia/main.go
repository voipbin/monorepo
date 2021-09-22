package externalmedia

import (
	"github.com/gofrs/uuid"
)

// ExternalMedia defines external media detail info
type ExternalMedia struct {
	CallID     uuid.UUID `json:"call_id"`     // call id
	AsteriskID string    `json:"asterisk_id"` // asterisk id
	ChannelID  string    `json:"channel_id"`  // external media channel id

	LocalIP   string `json:"local_ip"`
	LocalPort int    `json:"local_port"`

	ExternalHost   string `json:"external_host"`
	Encapsulation  string `json:"encapsulation"`
	Transport      string `json:"transport"`
	ConnectionType string `json:"connection_type"`
	Format         string `json:"format"`
	Direction      string `json:"direction"`
}
