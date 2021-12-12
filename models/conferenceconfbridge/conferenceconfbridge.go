package conferenceconfbridge

import "github.com/gofrs/uuid"

// ConferenceConfbridge defines
type ConferenceConfbridge struct {
	ConferenceID uuid.UUID `json:"conference_id"`
	ConfbridgeID uuid.UUID `json:"confbridge_id"`
}
