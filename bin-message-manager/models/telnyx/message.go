package telnyx

import "time"

type MessageResponse struct {
	Data MessageData `json:"data"`
}

type MessageData struct {
	RecordType            string        `json:"record_type"`
	Direction             string        `json:"direction"`
	ID                    string        `json:"id"`
	Type                  string        `json:"type"`
	OrganizationID        string        `json:"organization_id"`
	MessagingProfileID    string        `json:"messaging_profile_id"`
	From                  FromInfo      `json:"from"`
	To                    []ToInfo      `json:"to"`
	Cc                    []interface{} `json:"cc"` // Assuming it's always an empty array, use []string if it can contain strings
	Text                  string        `json:"text"`
	Media                 []interface{} `json:"media"` // Assuming it's always an empty array, use []string if it can contain strings
	WebhookURL            string        `json:"webhook_url"`
	WebhookFailoverURL    string        `json:"webhook_failover_url"`
	Encoding              string        `json:"encoding"`
	Parts                 int           `json:"parts"`
	Tags                  []string      `json:"tags"`
	Cost                  CostInfo      `json:"cost"`
	TcrCampaignID         interface{}   `json:"tcr_campaign_id"` // Assuming this can be null (empty), so using interface{}
	TcrCampaignBillable   bool          `json:"tcr_campaign_billable"`
	TcrCampaignRegistered interface{}   `json:"tcr_campaign_registered"` // Assuming this can be null
	ReceivedAt            time.Time     `json:"received_at"`
	SentAt                interface{}   `json:"sent_at"`      // Assuming this can be null
	CompletedAt           interface{}   `json:"completed_at"` // Assuming this can be null
	ValidUntil            time.Time     `json:"valid_until"`
	Errors                []interface{} `json:"errors"` // Assuming it's always an empty array, use []string if it can contain strings
	CostBreakdown         CostBreakdown `json:"cost_breakdown"`
}

type FromInfo struct {
	PhoneNumber string `json:"phone_number"`
	Carrier     string `json:"carrier"`
	LineType    string `json:"line_type"`
}

type ToInfo struct {
	PhoneNumber string `json:"phone_number"`
	Status      string `json:"status"`
	Carrier     string `json:"carrier"`
	LineType    string `json:"line_type"`
}

type CostInfo struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type CostBreakdown struct {
	Rate       RateInfo `json:"rate"`
	CarrierFee RateInfo `json:"carrier_fee"`
}

type RateInfo struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
