package sockhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// Test_NewSockHandler covers the constructor's two branches. Neither
// branch connects to a broker: TypeRabbitMQ delegates to
// rabbitmqhandler.NewRabbit, which only fills in struct fields (the actual
// connection happens later, in Connect()).
func Test_NewSockHandler(t *testing.T) {
	t.Run("TypeRabbitMQ returns a non-nil Rabbit handler", func(t *testing.T) {
		h := NewSockHandler(sock.TypeRabbitMQ, "")
		if h == nil {
			t.Fatalf("Wrong match. expect: a non-nil handler, got: nil")
		}

		want := reflect.TypeOf(rabbitmqhandler.NewRabbit(""))
		got := reflect.TypeOf(h)
		if got != want {
			t.Errorf("Wrong match. expect: %v, got: %v", want, got)
		}
	})

	t.Run("an unknown type returns a true nil interface", func(t *testing.T) {
		h := NewSockHandler(sock.Type("unknown"), "")
		if h != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", h)
		}
	})
}

type fakeSockHandler struct {
	SockHandler
}

func Test_SetRabbitFactory(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"the factory is used for the RabbitMQ type only", func(t *testing.T) {
			var gotURI string
			fake := &fakeSockHandler{}
			restore := SetRabbitFactory(func(serverURI string) SockHandler {
				gotURI = serverURI
				return fake
			})
			defer restore()

			if h := NewSockHandler(sock.TypeRabbitMQ, "amqp://example"); h != SockHandler(fake) || gotURI != "amqp://example" {
				t.Errorf("Wrong match. expect: the fake with the uri, got: %v, %q", h, gotURI)
			}

			if h := NewSockHandler(sock.Type("unknown"), ""); h != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", h)
			}
		}},

		{"restore and nested restore return the previous factory", func(t *testing.T) {
			first, second := &fakeSockHandler{}, &fakeSockHandler{}
			restoreFirst := SetRabbitFactory(func(string) SockHandler { return first })
			restoreSecond := SetRabbitFactory(func(string) SockHandler { return second })

			if h := NewSockHandler(sock.TypeRabbitMQ, ""); h != SockHandler(second) {
				t.Errorf("Wrong match. expect: second, got: %v", h)
			}
			restoreSecond()
			if h := NewSockHandler(sock.TypeRabbitMQ, ""); h != SockHandler(first) {
				t.Errorf("Wrong match. expect: first, got: %v", h)
			}
			restoreFirst()

			h := NewSockHandler(sock.TypeRabbitMQ, "")
			if want := reflect.TypeOf(rabbitmqhandler.NewRabbit("")); reflect.TypeOf(h) != want {
				t.Errorf("Wrong match. expect: %v, got: %T", want, h)
			}
		}},

		{"nil installs the default", func(t *testing.T) {
			restore := SetRabbitFactory(func(string) SockHandler { return &fakeSockHandler{} })
			defer restore()

			restoreNil := SetRabbitFactory(nil)
			h := NewSockHandler(sock.TypeRabbitMQ, "")
			if want := reflect.TypeOf(rabbitmqhandler.NewRabbit("")); reflect.TypeOf(h) != want {
				t.Errorf("Wrong match. expect: %v, got: %T", want, h)
			}

			restoreNil()
			if _, ok := NewSockHandler(sock.TypeRabbitMQ, "").(*fakeSockHandler); !ok {
				t.Errorf("Wrong match. expect: the fake factory again")
			}
		}},

		{"a factory may call the setter and read concurrently", func(t *testing.T) {
			var inner func()
			restore := SetRabbitFactory(func(string) SockHandler {
				inner = SetRabbitFactory(func(string) SockHandler { return &fakeSockHandler{} })
				return &fakeSockHandler{}
			})
			defer restore()

			NewSockHandler(sock.TypeRabbitMQ, "")
			defer inner()

			done := make(chan struct{})
			go func() {
				defer close(done)
				for i := 0; i < 200; i++ {
					SetRabbitFactory(func(string) SockHandler { return &fakeSockHandler{} })()
				}
			}()
			for i := 0; i < 200; i++ {
				NewSockHandler(sock.TypeRabbitMQ, "")
			}
			<-done
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
