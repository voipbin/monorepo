package request

import "github.com/gofrs/uuid"

// V1DataContactsIDStars is the request body for POST and DELETE
// /v1/contacts/{id}/stars/{agent_id}. The contact and the agent are in the path.
type V1DataContactsIDStars struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataStarsGet is the request body for GET /v1/contacts/starred and GET /v1/contact_stars.
// page_size and page_token are query parameters. agent_id is the calling agent, taken by
// api-manager from the authenticated identity.
type V1DataStarsGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
	AgentID    uuid.UUID `json:"agent_id"`
}
