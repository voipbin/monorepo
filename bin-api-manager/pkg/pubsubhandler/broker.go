package pubsubhandler

import (
	"sync"
)

type broker struct {
	mu   sync.RWMutex
	subs map[*subHandler]struct{}
}

// NewBrokerHandler creates a new BrokerHandler.
func NewBrokerHandler() BrokerHandler {
	return &broker{
		subs: make(map[*subHandler]struct{}),
	}
}

// Publish delivers the message to every registered subscriber whose
// subscription matches the topic. It is safe for concurrent use and never
// blocks on a slow subscriber.
func (b *broker) Publish(topic string, msg string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for s := range b.subs {
		s.deliver(topic, msg)
	}

	return nil
}

// NewSubHandler creates a new subscriber and registers it to the broker.
func (b *broker) NewSubHandler() SubHandler {
	s := &subHandler{
		broker: b,
		ch:     make(chan message, subscriberBufferSize),
		done:   make(chan struct{}),
	}

	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()

	return s
}

// unregister removes the subscriber from the broker. It acquires the write
// lock, so by the time it returns no in-flight Publish (which holds the read
// lock) can still deliver to the subscriber. The subscriber's data channel is
// never closed, which makes a send-on-closed-channel panic impossible under
// any Publish/Terminate interleaving.
func (b *broker) unregister(s *subHandler) {
	b.mu.Lock()
	delete(b.subs, s)
	b.mu.Unlock()
}
