package pubsubhandler

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_Subscribe_prefix_matching(t *testing.T) {

	tests := []struct {
		name string

		subscribes []string
		topic      string

		expectDelivered bool
	}{
		{
			name: "exact match",

			subscribes: []string{"customer_id:00000000-0000-0000-0000-000000000001:call"},
			topic:      "customer_id:00000000-0000-0000-0000-000000000001:call",

			expectDelivered: true,
		},
		{
			name: "prefix match",

			subscribes: []string{"customer_id:00000000-0000-0000-0000-000000000001:call"},
			topic:      "customer_id:00000000-0000-0000-0000-000000000001:call_created:00000000-0000-0000-0000-000000000002",

			expectDelivered: true,
		},
		{
			name: "no match",

			subscribes: []string{"customer_id:00000000-0000-0000-0000-000000000001:call"},
			topic:      "customer_id:00000000-0000-0000-0000-000000000009:call",

			expectDelivered: false,
		},
		{
			name: "no subscription",

			subscribes: []string{},
			topic:      "customer_id:00000000-0000-0000-0000-000000000001:call",

			expectDelivered: false,
		},
		{
			name: "subscribed prefix longer than topic",

			subscribes: []string{"customer_id:00000000-0000-0000-0000-000000000001:call_created"},
			topic:      "customer_id:00000000-0000-0000-0000-000000000001:call",

			expectDelivered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBrokerHandler()

			sub := b.NewSubHandler().(*subHandler)
			defer sub.Terminate()

			for _, topic := range tt.subscribes {
				if errSub := sub.Subscribe(topic); errSub != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errSub)
				}
			}

			if errPub := b.Publish(tt.topic, "test message"); errPub != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errPub)
			}

			delivered := len(sub.ch) > 0
			if delivered != tt.expectDelivered {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectDelivered, delivered)
			}
		})
	}
}

func Test_deliver_once_on_overlapping_prefixes(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler().(*subHandler)
	defer sub.Terminate()

	// two overlapping prefixes matching the same topic
	if errSub := sub.Subscribe("customer_id:x:"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}
	if errSub := sub.Subscribe("customer_id:x:call"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}

	if errPub := b.Publish("customer_id:x:call_created", "test message"); errPub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errPub)
	}

	if len(sub.ch) != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", len(sub.ch))
	}
}

func Test_Subscribe_resubscribe_idempotent(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler().(*subHandler)
	defer sub.Terminate()

	for i := 0; i < 3; i++ {
		if errSub := sub.Subscribe("topic"); errSub != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", errSub)
		}
	}

	if len(sub.topics) != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", len(sub.topics))
	}
}

func Test_Unsubscribe(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler().(*subHandler)
	defer sub.Terminate()

	if errSub := sub.Subscribe("topic"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}
	if errUnsub := sub.Unsubscribe("topic"); errUnsub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errUnsub)
	}

	// unsubscribing a topic that was never subscribed is a no-op
	if errUnsub := sub.Unsubscribe("unknown"); errUnsub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errUnsub)
	}

	if errPub := b.Publish("topic", "test message"); errPub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errPub)
	}

	if len(sub.ch) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(sub.ch))
	}
}

func Test_deliver_drop_on_full_buffer(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler().(*subHandler)
	defer sub.Terminate()

	if errSub := sub.Subscribe("topic"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}

	// fill the buffer beyond capacity. nobody consumes sub.ch, so the
	// overflow must be dropped without blocking Publish.
	chDone := make(chan struct{})
	go func() {
		defer close(chDone)
		for i := 0; i < subscriberBufferSize+100; i++ {
			if errPub := b.Publish("topic", fmt.Sprintf("message %d", i)); errPub != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errPub)
			}
		}
	}()

	select {
	case <-chDone:
	case <-time.After(time.Second * 5):
		t.Errorf("Publish blocked on a full subscriber buffer")
	}

	if len(sub.ch) != subscriberBufferSize {
		t.Errorf("Wrong match. expect: %d, got: %d", subscriberBufferSize, len(sub.ch))
	}
}

func Test_Terminate_idempotent(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler()

	// multiple Terminate calls must not panic
	sub.Terminate()
	sub.Terminate()
}

func Test_Publish_after_Terminate_no_delivery(t *testing.T) {
	b := NewBrokerHandler()

	sub := b.NewSubHandler().(*subHandler)
	if errSub := sub.Subscribe("topic"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}

	sub.Terminate()

	if errPub := b.Publish("topic", "test message"); errPub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errPub)
	}

	if len(sub.ch) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(sub.ch))
	}
}

// Test_concurrent_publish_subscribe_terminate stresses the
// Publish/Subscribe/Terminate interleavings. Run with -race: the design must
// hold that no send-on-closed-channel panic and no data race is possible.
func Test_concurrent_publish_subscribe_terminate(t *testing.T) {
	b := NewBrokerHandler()

	var wg sync.WaitGroup

	// concurrent publishers, mirroring subscribehandler's
	// goroutine-per-event publishing pattern
	stopPub := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for {
				select {
				case <-stopPub:
					return
				default:
					_ = b.Publish("topic", fmt.Sprintf("message from publisher %d", n))
				}
			}
		}(i)
	}

	// concurrent subscriber lifecycles, mirroring websocket connect/
	// subscribe/unsubscribe/disconnect churn
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := b.NewSubHandler()
			_ = sub.Subscribe("topic")
			_ = sub.Subscribe("topic2")
			_ = sub.Unsubscribe("topic2")
			time.Sleep(time.Millisecond * 10)
			sub.Terminate()
		}()
	}

	time.Sleep(time.Millisecond * 200)
	close(stopPub)

	chDone := make(chan struct{})
	go func() {
		defer close(chDone)
		wg.Wait()
	}()

	select {
	case <-chDone:
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting for concurrent goroutines to finish")
	}
}
