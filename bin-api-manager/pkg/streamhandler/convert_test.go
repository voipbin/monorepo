package streamhandler

import (
	"monorepo/bin-api-manager/models/stream"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_ConvertFromAudiosocket(t *testing.T) {
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

			res, err := h.ConvertFromAudiosocket(tt.st, tt.data)
			if err != nil {
				t.Errorf("Wrong match.\nexpected: ok\ngot: %v\n", err)
			}

			if !reflect.DeepEqual(res, tt.data) {
				t.Errorf("Wrong match.\nExpected: %+v\nGot: %+v", tt.data, res)
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
