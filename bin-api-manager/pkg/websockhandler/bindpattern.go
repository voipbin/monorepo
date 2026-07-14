package websockhandler

import (
	"fmt"
	"strings"
)

// topicToBindPattern converts a client-facing subscribe topic string (colon-delimited,
// "<scope>:<scope_id>:<resource>:<resource_id_or_*>") to the AMQP binding pattern
// (dot-delimited, "<scope>.<scope_id>.#") used for scope-first topic exchange binding.
// See VOIP-1258 design doc §5.
func topicToBindPattern(topic string) (string, error) {
	tmps := strings.Split(topic, ":")
	if len(tmps) < 2 {
		return "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return fmt.Sprintf("%s.%s.#", tmps[0], tmps[1]), nil
}
