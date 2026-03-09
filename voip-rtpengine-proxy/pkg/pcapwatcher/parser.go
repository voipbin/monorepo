package pcapwatcher

import (
	"fmt"
	"strings"
)

// parsePcapPathFromMetadata reads the first line of a metadata file content
// to extract the pcap file path.
func parsePcapPathFromMetadata(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty metadata file")
	}

	// First line is the pcap file path
	firstLine := strings.SplitN(content, "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	if firstLine == "" {
		return "", fmt.Errorf("no pcap path in metadata file")
	}

	return firstLine, nil
}

// buildObjectPath constructs the flat GCS object path for a pcap file.
func buildObjectPath(filename string) string {
	return fmt.Sprintf("rtp-recordings/%s", filename)
}
