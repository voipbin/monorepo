package streamhandler

import (
	"fmt"
	"monorepo/bin-api-manager/models/stream"
)

// ConvertFromWebsocket returns converted byte for
// asterisk -> api-manager -> websocket client
func (h *streamHandler) ConvertFromAudiosocket(st *stream.Stream, m []byte) ([]byte, error) {

	switch st.Encapsulation {
	case stream.EncapsulationAudiosocket:
		return m, nil

	default:
		return nil, fmt.Errorf("unsupported encapsulation: %s", st.Encapsulation)
	}
}

// ConvertFromWebsocket returns converted byte for
// websocket client -> api-manager -> asterisk
func (h *streamHandler) ConvertFromWebsocket(st *stream.Stream, m []byte) ([]byte, error) {

	switch st.Encapsulation {
	case stream.EncapsulationAudiosocket:
		return m, nil

	default:
		return nil, fmt.Errorf("unsupported encapsulation: %s", st.Encapsulation)
	}
}
