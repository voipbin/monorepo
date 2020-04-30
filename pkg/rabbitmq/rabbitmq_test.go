package rabbitmq

import "testing"

func TestNewQueue(t *testing.T) {

	type test struct {
		name string
		url  string
	}

	tests := []test{
		{
			"has url",
			"amqp://guest:guest@rabbitmq.voipbin.net:5672",
		},
		{
			"has no url",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rabbit := NewRabbit(tt.url)

			if rabbit.GetURL() != tt.url {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}

	return
}
