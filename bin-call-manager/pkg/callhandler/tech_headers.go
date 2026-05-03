package callhandler

import (
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/common"
)

// reservedTechHeaderKeys lists channel-variable keys that operator-supplied
// tech_headers are forbidden from setting. The set covers:
//   - headers the system writes conditionally (P-Asserted-Identity / Privacy
//     are only set for anonymous calls, so ordering alone cannot protect them),
//   - headers the system writes unconditionally (SDP-Transport, CALLERID(*)),
//   - internal trace headers (VB-CALL-ID / VB-CONFBRIDGE-ID) whose values must
//     match call-manager's own UUIDs for downstream correlation.
//
// Keys are checked both in their raw form (for CALLERID/PJSIP_HEADER entries)
// and after PJSIP_HEADER(add,...) wrapping (for SIP header names).
var reservedTechHeaderKeys = map[string]struct{}{
	"PJSIP_HEADER(add,P-Asserted-Identity)": {},
	"PJSIP_HEADER(add,Privacy)":             {},
	"PJSIP_HEADER(add,VBOUT-SDP_Transport)": {},
	"PJSIP_HEADER(add,VB-CALL-ID)":          {},
	"PJSIP_HEADER(add,VB-CONFBRIDGE-ID)":    {},
	"PJSIP_HEADER(add,VB-DIRECTION)":        {},
	"PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")": {}, // prevent provider override
	"CALLERID(name)":                        {},
	"CALLERID(num)":                         {},
	"CALLERID(pres)":                        {},
}

// mergeTechHeaders copies sanitized entries from src (raw operator-supplied
// tech_headers) into dst (Asterisk channel variables), wrapping each key
// with PJSIP_HEADER(add,...) so Asterisk attaches the header to the outgoing
// INVITE.
//
// Entries are skipped and logged as Debug when:
//   - key is empty
//   - key contains \r, \n, (, ), or ,
//   - wrapped key matches reservedTechHeaderKeys
//   - raw key itself matches reservedTechHeaderKeys (covers CALLERID(*) attempts)
//   - value contains \r or \n (CRLF injection defense)
//
// Skipped entries never fail the call — the rest of the tech config and the
// call itself proceed.
//
// Returns counts of applied and skipped entries for a single summary log at
// the call site.
func mergeTechHeaders(dst map[string]string, src map[string]string, log *logrus.Entry) (applied int, skipped int) {
	for k, v := range src {
		if k == "" {
			log.Debugf("Skipping tech_header with empty key.")
			skipped++
			continue
		}
		if _, reserved := reservedTechHeaderKeys[k]; reserved {
			log.Debugf("Skipping tech_header that collides with system-reserved key. key=%q", k)
			skipped++
			continue
		}
		if strings.ContainsAny(k, "\r\n(),") {
			log.Debugf("Skipping tech_header with invalid key char. key=%q", k)
			skipped++
			continue
		}
		if strings.ContainsAny(v, "\r\n") {
			log.Debugf("Skipping tech_header with CRLF in value. key=%q", k)
			skipped++
			continue
		}

		varKey := "PJSIP_HEADER(add," + k + ")"
		if _, reserved := reservedTechHeaderKeys[varKey]; reserved {
			log.Debugf("Skipping tech_header that collides with system-reserved header. key=%q", k)
			skipped++
			continue
		}

		dst[varKey] = v
		applied++
	}
	return applied, skipped
}
