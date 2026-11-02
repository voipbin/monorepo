package request

// V1DataMentionsGet represents the request body for GET /v1/mentions
type V1DataMentionsGet struct {
	CustomerID string `json:"customer_id"`
	AgentID    string `json:"agent_id"`
}
