package callhandler

import (
	"strings"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// embedCodecs copies the codec preference from an OutboundConfig into call
// metadata if the call does not already carry a per-call codecs override.
// Returns the (possibly newly allocated) metadata map.
func embedCodecs(metadata map[string]any, config *outboundconfig.OutboundConfig) map[string]any {
	if config == nil || config.Codecs == "" {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, alreadySet := metadata[call.MetadataKeyCodecs]; alreadySet {
		return metadata // per-call override wins
	}
	metadata[call.MetadataKeyCodecs] = config.Codecs
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
