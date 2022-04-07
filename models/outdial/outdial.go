package outdial

import "github.com/gofrs/uuid"

// Outdial defines
type Outdial struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	
	CampaignID uuid.UUID `json:"campaign_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Data string `json:"data"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
