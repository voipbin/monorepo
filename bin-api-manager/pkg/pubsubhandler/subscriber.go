package pubsubhandler

import (
	"slices"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// subHandler is a per-WebSocket-connection subscriber.
type subHandler struct {
	broker *broker

	mu     sync.Mutex
	topics []string // subscribed topic prefixes

	ch        chan message  // buffered delivery channel; never closed
	done      chan struct{} // closed by Terminate to signal shutdown
	closeOnce sync.Once
}

// deliver forwards the message to this subscriber when one of its subscribed
// prefixes matches the topic. Matching is prefix-based and deliver-once: even
// when several subscribed prefixes match the same topic, the message is
// enqueued a single time. When the buffer is full the message is dropped for
// this subscriber only.
func (s *subHandler) deliver(topic string, msg string) {
	s.mu.Lock()
	matched := false
	for _, prefix := range s.topics {
		if strings.HasPrefix(topic, prefix) {
			matched = true
			break
		}
	}
	s.mu.Unlock()

	if !matched {
		return
	}

	select {
	case s.ch <- message{topic: topic, data: msg}:
	default:
		promPubsubDroppedTotal.Inc()
	}
}

// Subscribe subscribes the topic prefix. Re-subscribing the same topic is a
// no-op.
func (s *subHandler) Subscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Subscribe",
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	// check already subscribed topic
	if slices.Contains(s.topics, topic) {
		log.Debugf("The topic already subscribed. topic: %s", topic)
		return nil
	}

	s.topics = append(s.topics, topic)
	log.Debugf("Subscribed the topic correctly. topic: %s", topic)

	return nil
}

// Unsubscribe unsubscribes the topic prefix.
func (s *subHandler) Unsubscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Unsubscribe",
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.Index(s.topics, topic)
	if idx == -1 {
		// nothing to unsubscribe
		return nil
	}
	s.topics = slices.Delete(s.topics, idx, idx+1)
	log.Debugf("Unsubscribed the topic correctly. topic: %s", topic)

	return nil
}

// Terminate unregisters the subscriber from the broker and signals the Run
// loop to stop. It is idempotent. The data channel is intentionally never
// closed; see broker.unregister for the shutdown-safety rationale.
func (s *subHandler) Terminate() {
	log := logrus.WithFields(logrus.Fields{
		"func": "Terminate",
	})

	s.closeOnce.Do(func() {
		s.broker.unregister(s)
		close(s.done)
		log.Debug("Terminated the subscriber.")
	})
}
