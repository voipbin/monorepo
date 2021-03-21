package astaor

// AstAOR strcut is for Asterisk's AOR info
type AstAOR struct {
	ID *string

	MaxContacts    *int
	RemoveExisting *string

	DefaultExpiration *int
	MinimumExpiration *int
	MaximumExpiration *int

	OutboundProxy *string
	SupportPath   *string

	AuthenticateQualify *string
	QualifyFrequency    *int
	QualifyTimeout      *float32

	Contact            *string
	Mailboxes          *string
	VoicemailExtension *string
}
