package channel

// SIPDataKey defines typed keys for Channel.SIPData map entries.
// Values are populated from Kamailio's Redis hash (kamailio:<sip-call-id>).
type SIPDataKey = string

const (
	SIPDataKeyCallID           SIPDataKey = "call_id"
	SIPDataKeyFromUser         SIPDataKey = "from_user"
	SIPDataKeyFromName         SIPDataKey = "from_name"
	SIPDataKeyFromDomain       SIPDataKey = "from_domain"
	SIPDataKeyFromURI          SIPDataKey = "from_uri"
	SIPDataKeyToUser           SIPDataKey = "to_user"
	SIPDataKeyToName           SIPDataKey = "to_name"
	SIPDataKeyToDomain         SIPDataKey = "to_domain"
	SIPDataKeyToURI            SIPDataKey = "to_uri"
	SIPDataKeyPAI              SIPDataKey = "pai"
	SIPDataKeyRTPEngineAddress SIPDataKey = "rtpengine_address"
	SIPDataKeyFromTag          SIPDataKey = "from_tag"
	SIPDataKeyDirection        SIPDataKey = "direction"
	SIPDataKeySourceIP         SIPDataKey = "source_ip"
	SIPDataKeyTransport        SIPDataKey = "transport"
	SIPDataKeyDomain           SIPDataKey = "domain"
)
