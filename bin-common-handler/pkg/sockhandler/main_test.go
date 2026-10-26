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
