package rabbitmqhandler

import "fmt"

func (h *rabbit) TopicCreate(name string) error {

	if errDeclare := h.ExchangeDeclare(name, "fanout", true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the queue for event. err: %v", errDeclare)
	}

	return nil
}
