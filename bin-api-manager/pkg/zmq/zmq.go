package zmq

import "github.com/pebbe/zmq4"

// Bind starts the zmq server
func (h *zmq) Bind(sockType zmq4.Type, addr string) error {
	tmp, err := zmq4.NewSocket(sockType)
	if err != nil {
		return err
	}
	h.sockType = sockType
	h.socket = tmp

	if errBind := h.socket.Bind(addr); errBind != nil {
		return errBind
	}

	h.address = addr

	return nil
}

// Connect connects to the zmq server
func (h *zmq) Connect(sockType zmq4.Type, addr string) error {
	tmp, err := zmq4.NewSocket(sockType)
	if err != nil {
		return err
	}
	h.sockType = sockType
	h.socket = tmp

	if errConnect := h.socket.Connect(addr); errConnect != nil {
		return errConnect
	}

	h.address = addr

	return nil
}

// Terminate terminates the zmq socket.
func (h *zmq) Terminate() {
	h.socket.Close()
}

// Subscribe subscribes the given topic.
func (h *zmq) Subscribe(topic string) error {
	return h.socket.SetSubscribe(topic)
}

// Unsubscribe unsubscribe the given topic.
func (h *zmq) Unsubscribe(topic string) error {
	return h.socket.SetUnsubscribe(topic)
}

// Publish publishes the message with the given topic.
func (h *zmq) Publish(topic string, m string) error {
	_, err := h.socket.SendMessage(topic, m)
	if err != nil {
		return err
	}

	return nil
}

// Receive returns received message.
func (h *zmq) Receive() ([]string, error) {
	return h.socket.RecvMessage(0)
}

// ReceiveNoBlock returns received message or return nil .
func (h *zmq) ReceiveNoBlock() ([]string, error) {
	return h.socket.RecvMessage(zmq4.DONTWAIT)
}
