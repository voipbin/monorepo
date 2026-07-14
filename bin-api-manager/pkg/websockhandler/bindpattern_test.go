package websockhandler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopicToBindPattern(t *testing.T) {
	cases := []struct {
		topic    string
		expected string
	}{
		{"customer_id:abc123:call:*", "customer_id.abc123.#"},
		{"agent_id:def456:queue:*", "agent_id.def456.#"},
		{"customer_id:abc123", "customer_id.abc123.#"},
	}
	for _, c := range cases {
		got, err := topicToBindPattern(c.topic)
		require.NoError(t, err)
		require.Equal(t, c.expected, got)
	}
}

func TestTopicToBindPattern_InvalidFormat(t *testing.T) {
	_, err := topicToBindPattern("invalidtopic")
	require.Error(t, err)
}
