package streamhandler

import (
	"context"
	"monorepo/bin-api-manager/models/stream"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

func Test_Create(t *testing.T) {
	tests := []struct {
		name string

		id            uuid.UUID
		connWebsocket *websocket.Conn
		encapsulation stream.Encapsulation

		expectedStream *stream.Stream
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("ec942ccc-b7c8-11ef-ac15-6fa4681da390"),
			connWebsocket: &websocket.Conn{},
			encapsulation: stream.EncapsulationAudiosocket,

			expectedStream: &stream.Stream{
				ID:            uuid.FromStringOrNil("ec942ccc-b7c8-11ef-ac15-6fa4681da390"),
				ConnWebsocket: &websocket.Conn{},
				Encapsulation: stream.EncapsulationAudiosocket,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamHandler{
				streamData: make(map[string]*stream.Stream),
			}
			ctx := context.Background()

			res, err := h.Create(ctx, tt.id, tt.connWebsocket, tt.encapsulation)
			if err != nil {
				t.Errorf("Expected no error but got %v", err)
			}

			tmp, err := h.Get(tt.id)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tmp, res) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tmp, res)
			}
		})
	}
}

func Test_Create_mutex(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamHandler{
				streamData: make(map[string]*stream.Stream),
			}
			ctx := context.Background()

			for i := 0; i < 1000; i++ {
				go func() {
					id := utilhandler.UUIDCreate()
					encapsulation := stream.EncapsulationAudiosocket
					tmp, err := h.Create(ctx, id, nil, encapsulation)
					if err != nil {
						t.Errorf("Wrong match. expected: ok, got: %v", err)
					}
					if tmp.ID != id {
						t.Errorf("Wrong match. expected: %s, got: %s", id, tmp.ID)
					}

					tmp2, err := h.Get(id)
					if err != nil {
						t.Errorf("Wrong match. expected: ok, got: %v", err)
					}
					if tmp2.ID != id {
						t.Errorf("Wrong match. expected: %v, got: %v", id, tmp2.ID)
					}

					h.Terminate(id)

					_, err = h.Get(id)
					if err == nil {
						t.Errorf("Wrong match. expected: error, got: ok")
					}
				}()
			}

			time.Sleep(time.Second * 3)
		})
	}
}
