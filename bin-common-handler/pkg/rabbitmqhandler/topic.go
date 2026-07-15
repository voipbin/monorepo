package rabbitmqhandler

import "fmt"

func (h *rabbit) TopicCreate(name string) error {

	if errDeclare := h.ExchangeDeclare(name, "fanout", true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the queue for event. err: %v", errDeclare)
	}

	return nil
}

// TopicCreateWithKind declares an exchange with the given kind ("fanout", "topic", "direct",
// etc.), durable=true (matching TopicCreate's existing durability). Added for VOIP-1258 to
// support topic-kind exchanges without touching TopicCreate's existing fanout-only behavior.
func (h *rabbit) TopicCreateWithKind(name string, kind string) error {
	if errDeclare := h.ExchangeDeclare(name, kind, true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the exchange with kind %s. err: %v", kind, errDeclare)
	}
	return nil
}
