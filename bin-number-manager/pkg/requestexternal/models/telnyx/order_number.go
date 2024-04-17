package telnyx

// OrderNumberPhoneNumber struct
type OrderNumberPhoneNumber struct {
	ID                     string   `json:"id"`
	RequirementsMet        bool     `json:"requirements_met"`
	PhoneNumber            string   `json:"phone_number"`
	RegulatoryRequirements []string `json:"regulatory_requirements"`
	Status                 string   `json:"status"`
	RecordType             string   `json:"record_type"`
}

// OrderNumber struct
type OrderNumber struct {
	ID                 string                   `json:"id"`
	RequirementsMet    bool                     `json:"requirements_met"`
	CreatedAt          string                   `json:"created_at"`
	Status             string                   `json:"status"`
	ConnectionID       string                   `json:"connection_id"`
	PhoneNumbers       []OrderNumberPhoneNumber `json:"phone_numbers"`
	MessagingProfileID string                   `json:"messaging_profile_id"`
	CustomerReference  string                   `json:"customer_reference"`
	UpdatedAt          string                   `json:"updated_at"`
	BillingGroupID     string                   `json:"billing_group_id"`
	PhoneNumbersCount  int                      `json:"phone_numbers_count"`
	RecordType         string                   `json:"record_type"`
}
