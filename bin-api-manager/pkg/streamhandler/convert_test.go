package streamhandler

import (
	"monorepo/bin-api-manager/models/stream"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/pion/rtp"
	"go.uber.org/mock/gomock"
)

func Test_ConvertFromAudiosocket(t *testing.T) {
	type test struct {
		name string

		st        *stream.Stream
		data      []byte
		sequence  uint16
		timestamp uint32
		ssrc      uint32

		expectedData      []byte
		expectedSequence  uint16
		expectedTimestamp uint32
	}

	tests := []test{
		{
			name: "normal audiosocket",

			st: &stream.Stream{
				Encapsulation: stream.EncapsulationAudiosocket,
			},
			data: []byte{
				0x01,       // Type: UUID type
				0x10, 0x00, // Payload length: 16 bytes
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 1
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 2
			},
			sequence:  uint16(0),
			timestamp: uint32(0),
			ssrc:      uint32(0),

			expectedData: []byte{
				0x01,       // Type: UUID type
				0x10, 0x00, // Payload length: 16 bytes
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 1
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 2
			},
			expectedSequence:  uint16(0),
			expectedTimestamp: uint32(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &streamHandler{
				reqHandler: mockReq,
			}

			resData, resSequence, resTimestamp, err := h.ConvertFromAsterisk(tt.st, tt.data, tt.sequence, tt.timestamp, tt.ssrc)
			if err != nil {
				t.Errorf("Wrong match.\nexpected: ok\ngot: %v\n", err)
			}

			if !reflect.DeepEqual(resData, tt.data) {
				t.Errorf("Wrong match.\nExpected: %+v\nGot: %+v", tt.data, resData)
			}
			if resSequence != tt.sequence {
				t.Errorf("Wrong match.\nExpected: %d\nGot: %d", tt.sequence, resSequence)
			}
			if resTimestamp != tt.timestamp {
				t.Errorf("Wrong match.\nExpected: %d\nGot: %d", tt.timestamp, resTimestamp)
			}
		})
	}
}

func Test_ConvertFromWebsocket(t *testing.T) {
	type test struct {
		name string

		st   *stream.Stream
		data []byte
	}

	tests := []test{
		{
			name: "normal",

			st: &stream.Stream{
				Encapsulation: stream.EncapsulationAudiosocket,
			},
			data: []byte{
				0x01,       // Type: UUID type
				0x10, 0x00, // Payload length: 16 bytes
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 1
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // UUID part 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &streamHandler{
				reqHandler: mockReq,
			}

			res, err := h.ConvertFromWebsocket(tt.st, tt.data)
			if err != nil {
				t.Errorf("Wrong match.\nexpected: ok\ngot: %v\n", err)
			}

			if !reflect.DeepEqual(res, tt.data) {
				t.Errorf("Wrong match.\nExpected: %+v\nGot: %+v", tt.data, res)
			}
		})
	}
}

func Test_convertAudiosocketToRTP(t *testing.T) {
	type test struct {
		name string

		data             []byte
		initialSequence  uint16
		initialTimestamp uint32
		ssrc             uint32

		expectedData      []byte
		expectedSeq       uint16
		expectedTimestamp uint32
	}

	tests := []test{
		{
			name: "normal case",
			data: []byte{
				0x10,       // Type: PCM (SLIN)
				0x00, 0x10, // Payload length: 16 bytes
				// 16 bytes of PCM data (just a dummy example)
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			},
			initialSequence:  1000,
			initialTimestamp: 500,
			ssrc:             12345,

			expectedData: func() []byte {
				tmp := &rtp.Packet{
					Header: rtp.Header{
						Version:        2,
						PayloadType:    0, // For G.711 u-law (assuming default payload type)
						SequenceNumber: 1001,
						Timestamp:      516, // 500 + 16 samples
						SSRC:           12345,
					},
					Payload: []byte{
						0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
						0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
					},
				}
				res, err := tmp.Marshal()
				if err != nil {
					t.Fatalf("failed to marshal RTP packet: %v", err)
				}
				return res
			}(),
			expectedSeq:       1001,
			expectedTimestamp: 516, // 500 + 16 (number of samples)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamHandler{}

			resData, resSeq, resTimestamp, err := h.convertAudiosocketToRTP(tt.data, tt.initialSequence, tt.initialTimestamp, tt.ssrc)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
			if !reflect.DeepEqual(resData, tt.expectedData) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedData, resData)
			}
			if resSeq != tt.expectedSeq {
				t.Errorf("Wrong match. expected: %d, got: %d", tt.expectedSeq, resSeq)
			}
			if resTimestamp != tt.expectedTimestamp {
				t.Errorf("Wrong match. expected: %d, got: %d", tt.expectedTimestamp, resTimestamp)
			}

			res2, err := h.convertRTPToAudiosocket(resData)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res2, tt.data) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v\n", tt.data, res2)
			}
		})
	}
}

func Test_convertAudiosocketToSLNP(t *testing.T) {
	type test struct {
		name string

		data []byte

		expectedData []byte
	}

	tests := []test{
		{
			name: "normal case",
			data: []byte{
				0x10,       // Type: PCM (SLIN)
				0x00, 0x10, // Payload length: 16 bytes
				// 16 bytes of PCM data (just a dummy example)
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			},

			expectedData: []byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamHandler{}

			resData, err := h.convertAudiosocketToSLN(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
			if !reflect.DeepEqual(resData, tt.expectedData) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedData, resData)
			}

			res2, err := h.convertSLNToAudiosocket(resData)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res2, tt.data) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v\n", tt.data, res2)
			}
		})
	}
}
