package processmanager

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var allowedCommands = map[string]bool{
	"tcpdump": true,
}

var shellMetachars = regexp.MustCompile("[;|&$`]")

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.pcap$`)

func validateCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}
	if !allowedCommands[command] {
		return fmt.Errorf("command %q not in whitelist", command)
	}
	return nil
}

func sanitizeParameters(params []string) error {
	for _, p := range params {
		if shellMetachars.MatchString(p) {
			return fmt.Errorf("parameter contains shell metacharacter: %q", p)
		}
	}
	return nil
}

func validateWritePath(params []string) error {
	wIndex := -1
	for i, p := range params {
		if p == "-w" {
			wIndex = i
			break
		}
	}
	if wIndex == -1 || wIndex+1 >= len(params) {
		return fmt.Errorf("-w flag with output path is required")
	}

	path := params[wIndex+1]

	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, "/tmp/") {
		return fmt.Errorf("write path must be under /tmp/, got %q", path)
	}

	if !strings.HasSuffix(cleaned, ".pcap") {
		return fmt.Errorf("write path must end with .pcap, got %q", path)
	}

	filename := filepath.Base(cleaned)
	if !uuidPattern.MatchString(filename) {
		return fmt.Errorf("filename must be <uuid>.pcap, got %q", filename)
	}

	return nil
}
