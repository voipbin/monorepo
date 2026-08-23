package servicehandler

import (
	"encoding/xml"
	"fmt"

	rmextension "monorepo/bin-registrar-manager/models/extension"
)

// lpConfig models the Linphone remote-provisioning (lpconfig) XML document.
// See http://www.linphone.org/xsds/lpconfig.xsd. Field values are raw
// strings; encoding/xml performs all escaping (never pre-escape values --
// mixing manual and encoder escaping double-escapes).
type lpConfig struct {
	XMLName  xml.Name    `xml:"config"`
	Xmlns    string      `xml:"xmlns,attr"`
	Sections []lpSection `xml:"section"`
}

type lpSection struct {
	Name    string    `xml:"name,attr"`
	Entries []lpEntry `xml:"entry"`
}

type lpEntry struct {
	Name      string `xml:"name,attr"`
	Overwrite string `xml:"overwrite,attr"`
	Value     string `xml:",chardata"`
}

// renderExtensionProvisioningXML renders the lpconfig XML served to a
// Linphone client for the given extension. The input MUST be the
// WebhookMessage form (Realm-first domain rule already applied) -- never the
// internal model, whose DomainName may hold a legacy customer UUID.
//
// transport=udp is explicit: the customer-domain TLS path is not yet
// validated (design doc §3.4); TLS transport is a follow-up ticket.
func renderExtensionProvisioningXML(wm *rmextension.WebhookMessage) ([]byte, error) {
	domain := wm.DomainName

	cfg := lpConfig{
		Xmlns: "http://www.linphone.org/xsds/lpconfig.xsd",
		Sections: []lpSection{
			{
				Name: "misc",
				Entries: []lpEntry{
					// transient_provisioning keeps recent SDKs from persisting
					// the provisioning URI, avoiding a re-fetch failure warning
					// after the token TTL expires. Older clients ignore it.
					{Name: "transient_provisioning", Overwrite: "true", Value: "1"},
				},
			},
			{
				Name: "proxy_0",
				Entries: []lpEntry{
					{Name: "reg_proxy", Overwrite: "true", Value: fmt.Sprintf("<sip:%s;transport=udp>", domain)},
					{Name: "reg_identity", Overwrite: "true", Value: fmt.Sprintf("sip:%s@%s", wm.Extension, domain)},
					{Name: "reg_expires", Overwrite: "true", Value: "3600"},
					{Name: "reg_sendregister", Overwrite: "true", Value: "1"},
					{Name: "publish", Overwrite: "true", Value: "0"},
				},
			},
			{
				Name: "auth_info_0",
				Entries: []lpEntry{
					{Name: "username", Overwrite: "true", Value: wm.Username},
					{Name: "domain", Overwrite: "true", Value: domain},
					{Name: "passwd", Overwrite: "true", Value: wm.Password},
					{Name: "realm", Overwrite: "true", Value: domain},
				},
			},
		},
	}

	body, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal lpconfig xml: %w", err)
	}

	res := append([]byte(xml.Header), body...)
	res = append(res, '\n')
	return res, nil
}
