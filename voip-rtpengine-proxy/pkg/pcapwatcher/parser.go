package pcapwatcher

import (
	"fmt"
	"strings"
)

// extractCallIDFromFilename extracts the call-id from a metadata filename.
// RTPEngine metadata filenames follow the pattern: <call-id>-<tag>
// The call-id may contain hyphens (e.g., UUID format), so we split on the
// last hyphen-separated segment that looks like a tag.
func extractCallIDFromFilename(filename string) string {
	// Remove file extension if present
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		filename = filename[:idx]
	}

	// The tag is the last hyphen-separated component.
	// Call-ids can contain hyphens, so find the last hyphen.
	if idx := strings.LastIndex(filename, "-"); idx >= 0 {
		return filename[:idx]
	}

	return filename
}

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

// buildObjectPath constructs the GCS object path for a pcap file.
func buildObjectPath(callID, filename string) string {
	return fmt.Sprintf("rtp-recordings/%s/%s", callID, filename)
}
