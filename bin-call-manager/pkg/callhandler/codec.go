package callhandler

import (
	"strings"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
)

// embedCustomerCodecs copies OutboundCodecs from customer metadata into call
// metadata if the call does not already carry a codecs override.
// Returns the (possibly newly allocated) metadata map.
func embedCustomerCodecs(metadata map[string]any, outboundCodecs string) map[string]any {
	if _, alreadySet := metadata[call.MetadataKeyCodecs]; alreadySet {
		return metadata
	}
	if outboundCodecs == "" {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata[call.MetadataKeyCodecs] = outboundCodecs
	return metadata
}

// setChannelVariableCodecs adds the VBOUT-CODECS SIP header to outgoing channel
// variables if a codec preference is present in call metadata.
// CRLF characters in the value are rejected silently (header-injection defence).
func setChannelVariableCodecs(variables map[string]string, metadata map[string]any) {
	codecs, ok := metadata[call.MetadataKeyCodecs].(string)
	if !ok || codecs == "" {
		return
	}
	if strings.ContainsAny(codecs, "\r\n") {
		return
	}
	variables["PJSIP_HEADER(add,"+common.SIPHeaderCodecs+")"] = codecs
}
