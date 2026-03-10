package callhandler

import (
	"fmt"
	"strings"
)

// extractLocalPorts extracts all "local port" values from an RTPEngine NG query response.
// The response has nested structure: tags -> <tag> -> medias -> [] -> streams -> [] -> "local port".
func extractLocalPorts(queryResponse map[string]interface{}) ([]int, error) {
	tagsRaw, ok := queryResponse["tags"]
	if !ok {
		return nil, fmt.Errorf("no 'tags' in query response")
	}

	tags, ok := tagsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'tags' is not a map")
	}

	var ports []int

	for _, tagDataRaw := range tags {
		tagData, ok := tagDataRaw.(map[string]interface{})
		if !ok {
			continue
		}

		mediasRaw, ok := tagData["medias"]
		if !ok {
			continue
		}

		medias, ok := mediasRaw.([]interface{})
		if !ok {
			continue
		}

		for _, mediaRaw := range medias {
			media, ok := mediaRaw.(map[string]interface{})
			if !ok {
				continue
			}

			streamsRaw, ok := media["streams"]
			if !ok {
				continue
			}

			streams, ok := streamsRaw.([]interface{})
			if !ok {
				continue
			}

			for _, streamRaw := range streams {
				stream, ok := streamRaw.(map[string]interface{})
				if !ok {
					continue
				}

				portRaw, ok := stream["local port"]
				if !ok {
					continue
				}

				switch p := portRaw.(type) {
				case float64:
					ports = append(ports, int(p))
				case int64:
					ports = append(ports, int(p))
				case int:
					ports = append(ports, p)
				}
			}
		}
	}

	if len(ports) == 0 {
		return nil, fmt.Errorf("no local ports found in query response")
	}

	return ports, nil
}

// buildBPFFilter builds a tcpdump BPF filter string for the given ports.
func buildBPFFilter(ports []int) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = fmt.Sprintf("udp port %d", p)
	}
	return strings.Join(parts, " or ")
}
