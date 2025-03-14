package sendgrid

type SendGridEvent struct {
	Email       string      `json:"email"`
	Timestamp   int64       `json:"timestamp"`
	Event       string      `json:"event"`
	SgMessageId string      `json:"sg_message_id"`
	SgEventId   string      `json:"sg_event_id"`
	URL         string      `json:"url,omitempty"`
	Status      string      `json:"status,omitempty"`
	Category    interface{} `json:"category,omitempty"` // Can be string or []string
	ASMGroupID  int         `json:"asm_group_id,omitempty"`
	IP          string      `json:"ip,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
	Device      string      `json:"device,omitempty"`
	Geo         Geo         `json:"geo,omitempty"`
	Reason      string      `json:"reason,omitempty"`
	Type        string      `json:"type,omitempty"`
	Attempt     int         `json:"attempt,omitempty"`
	TLS         int         `json:"tls,omitempty"`
	CertErrors  string      `json:"cert_errors,omitempty"`
	UserID      string      `json:"user_id,omitempty"`
	CustomArgs  interface{} `json:"custom_args,omitempty"` // Use a map to handle dynamic custom args

	// Add other fields as needed based on the SendGrid documentation
	VoipbinMessageID string `json:"voipbin_message_id,omitempty"` // Voipbin message ID
}

type Geo struct {
	Country    string `json:"country,omitempty"`
	Region     string `json:"region,omitempty"`
	City       string `json:"city,omitempty"`
	Lat        string `json:"lat,omitempty"`
	Long       string `json:"long,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}
