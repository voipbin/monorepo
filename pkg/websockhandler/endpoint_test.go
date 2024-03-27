package websockhandler

import (
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_endpointInit(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			endpointInit()

			for i := minPort; i < maxPort; i++ {
				if gPortsAvailable[i] != uuid.Nil {
					t.Errorf("Wrong match. expect: %v, got: %v", uuid.Nil, gPortsAvailable[i])
				}
			}
		})
	}
}

func Test_portGet(t *testing.T) {
	tests := []struct {
		name string

		callID  uuid.UUID
		release bool

		expectRes int
	}{
		{
			name: "normal",

			callID:    uuid.FromStringOrNil("92c09dc4-e91b-11ee-a389-3b8e61af88a5"),
			release:   false,
			expectRes: 10000,
		},
		{
			name: "get next port",

			callID:    uuid.FromStringOrNil("14529554-e91c-11ee-bf4e-fba39ce96ed7"),
			release:   false,
			expectRes: 10001,
		},
		{
			name: "get next port and release",

			callID:    uuid.FromStringOrNil("6ea56efa-e91c-11ee-91d2-63e2b467ef49"),
			release:   true,
			expectRes: 10002,
		},
		{
			name: "get the same port because it was released in the previous",

			callID:    uuid.FromStringOrNil("6aedfba6-e91c-11ee-9e5d-87f7e6d05659"),
			release:   false,
			expectRes: 10002,
		},
	}

	endpointInit()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := portGet(tt.callID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

			if tt.release {
				portRelease(tt.callID)
			}

			t.Logf("res: %d", res)
		})
	}
}

func Test_portGet_concurrent(t *testing.T) {
	tests := []struct {
		name string

		concurrent int
		times      int
	}{
		{
			name: "normal",

			concurrent: 30,
			times:      200,
		},
	}

	endpointInit()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			for i := 0; i < tt.concurrent; i++ {
				go func() {
					for j := 0; j < tt.times; j++ {
						id := uuid.Must(uuid.NewV4())
						tmp := portGet(id)
						if tmp == -1 {
							t.Errorf("Wrong match. expect: ok, got: %v", tmp)
							return
						}
						time.Sleep(time.Millisecond * 10)
						portRelease(id)
					}
				}()
			}

			time.Sleep(time.Second * 5)
		})
	}
}

func Test_endpointLocalGet(t *testing.T) {

	tests := []struct {
		name string

		podIP  string
		callID uuid.UUID

		release bool

		expectRes string
	}{
		{
			name: "normal",

			podIP:  "127.0.0.1",
			callID: uuid.FromStringOrNil("317b5c7a-e920-11ee-849c-4f799bd9d1fb"),

			release: false,

			expectRes: "127.0.0.1:10000",
		},
		{
			name: "increase port number",

			podIP:  "127.0.0.1",
			callID: uuid.FromStringOrNil("b86d693a-e920-11ee-8e7e-bfc4bc1fd477"),

			release: true,

			expectRes: "127.0.0.1:10001",
		},
		{
			name: "same port number becuase released in the previous",

			podIP:  "127.0.0.1",
			callID: uuid.FromStringOrNil("db9cc112-e920-11ee-b26c-93e1660749a9"),

			release: false,

			expectRes: "127.0.0.1:10001",
		},
	}

	endpointInit()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if err := os.Setenv("POD_IP", tt.podIP); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res := endpointLocalGet(tt.callID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}

			if tt.release {
				endpointLocalRelease(tt.callID)
			}
		})
	}
}
