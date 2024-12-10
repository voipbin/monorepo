package streamhandler

import (
	"encoding/binary"
	"fmt"
	"monorepo/bin-api-manager/models/stream"

	"github.com/pion/rtp"
)

const (
	audioSocketHeaderSize = 12 // AudioSocket header size (4 bytes each for sampleRate, sequenceNumber, timestamp)
	payloadType           = 7  // Payload type for G.711 μ-law (μ-law)
	sampleRate            = 8000
	ssrc                  = 12345
)

// ConvertFromWebsocket returns converted byte for
// asterisk -> api-manager -> websocket client
func (h *streamHandler) ConvertFromAudiosocket(st *stream.Stream, m []byte) ([]byte, error) {

	switch st.Encapsulation {
	case stream.EncapsulationAudiosocket:
		return m, nil

	case stream.EncapsulationRTP:
		rtpData, err := h.convertAudiosocketToRTP(m)
		if err != nil {
			return nil, err
		}
		return rtpData, nil

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

	case stream.EncapsulationRTP:
		rtpData, err := h.convertRTPToAudiosocket(m)
		if err != nil {
			return nil, err
		}
		return rtpData, nil

	default:
		return nil, fmt.Errorf("unsupported encapsulation: %s", st.Encapsulation)
	}
}

func (h *streamHandler) convertAudiosocketToRTP(audioSocketData []byte) ([]byte, error) {
	if len(audioSocketData) < audioSocketHeaderSize {
		return nil, fmt.Errorf("AudioSocket data too short")
	}

	// Extract AudioSocket header and payload
	header := audioSocketData[:audioSocketHeaderSize]
	payload := audioSocketData[audioSocketHeaderSize:]

	// Parse AudioSocket header fields
	sequenceNumber := binary.BigEndian.Uint32(header[4:8])
	timestamp := binary.BigEndian.Uint32(header[8:12])

	// Construct the RTP packet
	packet := &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			PayloadType:    payloadType,
			SequenceNumber: uint16(sequenceNumber),
			Timestamp:      timestamp,
			SSRC:           ssrc,
		},
		Payload: payload,
	}

	res, err := packet.Marshal()
	if err != nil {
		return nil, fmt.Errorf("could not marshal RTP packet. err: %v", err)
	}

	return res, nil
}

func (h *streamHandler) convertRTPToAudiosocket(rtpData []byte) ([]byte, error) {

	rtpPacket := &rtp.Packet{}
	err := rtpPacket.Unmarshal(rtpData)
	if err != nil {
		fmt.Println("Could not unmarshal RTP packet:", err)
		return nil, fmt.Errorf("could not unmarshal RTP packet: %v", err)
	}

	// Construct the AudioSocket header
	audioSocketHeader := make([]byte, audioSocketHeaderSize)

	// Add sample rate (4 bytes)
	binary.BigEndian.PutUint32(audioSocketHeader[:4], sampleRate)

	// Add RTP sequence number (4 bytes)
	binary.BigEndian.PutUint32(audioSocketHeader[4:8], uint32(rtpPacket.SequenceNumber))

	// Add RTP timestamp (4 bytes)
	binary.BigEndian.PutUint32(audioSocketHeader[8:12], rtpPacket.Timestamp)

	// Combine header and payload to form the AudioSocket message
	res := append(audioSocketHeader, rtpPacket.Payload...)

	return res, nil
}
