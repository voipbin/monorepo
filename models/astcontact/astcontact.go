package astcontact

// AstContact for ast_contact table
type AstContact struct {
	ID                  string  `json:"id"`
	URI                 string  `json:"uri"`
	ExpirationTime      int     `json:"expiration_time"`
	QualifyFrequency    int     `json:"qualify_frequency"`
	OutboundProxy       string  `json:"outbound_proxy"`
	Path                string  `json:"path"`
	UserAgent           string  `json:"user_agent"`
	QualifyTimeout      float64 `json:"qualify_timeout"`
	RegServer           string  `json:"reg_server"`
	AuthenticateQualify string  `json:"authenticate_qualify"`
	ViaAddr             string  `json:"via_addr"`
	ViaPort             int     `json:"via_port"`
	CallID              string  `json:"call_id"`
	Endpoint            string  `json:"endpoint"`
	PruneOnBoot         string  `json:"prune_on_boot"`
}
