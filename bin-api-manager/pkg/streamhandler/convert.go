package streamhandler

import (
	"fmt"
	"monorepo/bin-api-manager/models/stream"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/pion/rtp"
)

const (
	defaultRTPPayloadType = 0 // ITU-T G.711 PCM μ-Law audio 64 kbit/s
)

// ConvertFromWebsocket returns converted byte for
// asterisk -> api-manager -> websocket client
func (h *streamHandler) ConvertFromAsterisk(st *stream.Stream, m []byte, sequence uint16, timestamp uint32, ssrc uint32) ([]byte, uint16, uint32, error) {

	switch st.Encapsulation {
	case stream.EncapsulationAudiosocket:
		return m, 0, 0, nil

	case stream.EncapsulationRTP:
		return h.convertAudiosocketToRTP(m, sequence, timestamp, ssrc)

	case stream.EncapsulationSLN:
		res, err := h.convertAudiosocketToSLN(m)
		return res, 0, 0, err

	default:
		return nil, 0, 0, fmt.Errorf("unsupported encapsulation: %s", st.Encapsulation)
	}
}

// ConvertFromWebsocket returns converted byte for
// websocket client -> api-manager -> asterisk
func (h *streamHandler) ConvertFromWebsocket(st *stream.Stream, m []byte) ([]byte, error) {

	switch st.Encapsulation {
	case stream.EncapsulationAudiosocket:
		return m, nil

	case stream.EncapsulationRTP:
		return h.convertRTPToAudiosocket(m)

	case stream.EncapsulationSLN:
		return h.convertSLNToAudiosocket(m)

	default:
		return nil, fmt.Errorf("unsupported encapsulation: %s", st.Encapsulation)
	}
}

func (h *streamHandler) convertAudiosocketToRTP(m []byte, sequence uint16, timestamp uint32, ssrc uint32) ([]byte, uint16, uint32, error) {
	ap := audiosocket.MessageFromData(m)

	// Ensure this is a PCM payload type
	if ap.Kind() != audiosocket.KindSlin {
		return nil, 0, 0, fmt.Errorf("unsupported AudioSocket message type: 0x%x", ap.Kind())
	}

	numSamples := ap.ContentLength()
	resTimestamp := timestamp + uint32(numSamples)
	resSequence := sequence + 1

	tmp := &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			PayloadType:    defaultRTPPayloadType,
			SequenceNumber: resSequence,
			Timestamp:      resTimestamp,
			SSRC:           ssrc,
		},
		Payload: ap.Payload(),
	}

	resData, err := tmp.Marshal()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not marshal the RTP data. err: %v", err)
	}

	return resData, resSequence, resTimestamp, nil
}

func (h *streamHandler) convertAudiosocketToSLN(m []byte) ([]byte, error) {
	ap := audiosocket.MessageFromData(m)

	// Ensure this is a PCM payload type
	if ap.Kind() != audiosocket.KindSlin {
		return nil, fmt.Errorf("unsupported AudioSocket message type: 0x%x", ap.Kind())
	}

	return ap.Payload(), nil
}

func (h *streamHandler) convertRTPToAudiosocket(m []byte) ([]byte, error) {
	// Unmarshal the RTP packet
	tmp := &rtp.Packet{}
	if err := tmp.Unmarshal(m); err != nil {
		return nil, fmt.Errorf("could not unmarshal RTP data: %v", err)
	}

	// Ensure the payload type is supported
	if tmp.PayloadType != defaultRTPPayloadType {
		return nil, fmt.Errorf("unsupported RTP payload type: %d", tmp.PayloadType)
	}

	// Handle the RTP payload (e.g., PCM data)
	audioPayload := tmp.Payload

	res := audiosocket.SlinMessage(audioPayload)
	return res, nil
}

func (h *streamHandler) convertSLNToAudiosocket(m []byte) ([]byte, error) {
	res := audiosocket.SlinMessage(m)
	return res, nil
}
