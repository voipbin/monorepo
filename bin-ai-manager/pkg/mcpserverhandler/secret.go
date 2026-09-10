// Package mcpserverhandler owns CRUD for customer-registered McpServer rows,
// their secret envelope encryption, and the SSRF guard applied at
// create/update time. See docs/plans/2026-09-11-mcp-tool-integration-design.md.
package mcpserverhandler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SecretCrypto holds the parsed, ordered set of AES-256-GCM keys configured
// via MCP_SECRET_ENCRYPTION_KEYS (design §6). CurrentVersion is the highest
// configured version, used for all new writes; every configured version
// remains available for decrypting rows encrypted under it.
//
// Exported (rather than kept package-private) because pkg/mcptoolhandler's
// CallTool/ListTools decrypt a row's secret using the same key set at a
// different call site (design §16 "mcpToolHandler interface").
type SecretCrypto struct {
	keys           map[int][]byte
	currentVersion int
}

// NewSecretCrypto parses the "<version>:<base64-32-byte-key>[,...]" config
// string. Returns an error for any malformed entry -- fail closed at boot,
// per design §6. An empty string is valid input (no keys configured yet);
// Encrypt then fails until one is configured.
func NewSecretCrypto(raw string) (*SecretCrypto, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &SecretCrypto{keys: map[int][]byte{}}, nil
	}

	keys := map[int][]byte{}
	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		idx := strings.Index(entry, ":")
		if idx < 0 {
			return nil, fmt.Errorf("mcp secret encryption key entry malformed (expected <version>:<base64-key>): %q", entry)
		}

		versionStr := strings.TrimSpace(entry[:idx])
		keyStr := strings.TrimSpace(entry[idx+1:])

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("mcp secret encryption key version is not an integer: %q", versionStr)
		}
		if version <= 0 {
			return nil, fmt.Errorf("mcp secret encryption key version must be > 0, got %d", version)
		}

		keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("mcp secret encryption key version %d is not valid base64: %w", version, err)
		}
		if len(keyBytes) != 32 {
			return nil, fmt.Errorf("mcp secret encryption key version %d must decode to exactly 32 bytes, got %d", version, len(keyBytes))
		}

		if _, exists := keys[version]; exists {
			return nil, fmt.Errorf("mcp secret encryption key version %d is configured more than once", version)
		}

		keys[version] = keyBytes
	}

	currentVersion := 0
	versions := make([]int, 0, len(keys))
	for v := range keys {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	if len(versions) > 0 {
		currentVersion = versions[len(versions)-1]
	}

	return &SecretCrypto{keys: keys, currentVersion: currentVersion}, nil
}

// Encrypt encrypts plaintext under the current (highest-version) key,
// returning ciphertext, nonce and the key version used.
func (c *SecretCrypto) Encrypt(plaintext string) (ciphertext []byte, nonce []byte, version int, err error) {
	if c.currentVersion == 0 {
		return nil, nil, 0, fmt.Errorf("no MCP secret encryption key is configured")
	}

	key, ok := c.keys[c.currentVersion]
	if !ok {
		return nil, nil, 0, fmt.Errorf("current mcp secret encryption key version %d not found", c.currentVersion)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("could not create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("could not create GCM: %w", err)
	}

	n := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(n); err != nil {
		return nil, nil, 0, fmt.Errorf("could not generate nonce: %w", err)
	}

	ct := gcm.Seal(nil, n, []byte(plaintext), nil)
	return ct, n, c.currentVersion, nil
}

// Decrypt decrypts ciphertext using the key identified by version -- the
// row's own KeyVersion, never the current config version -- so rotating the
// configured key set never breaks decryption of older rows (design §6).
func (c *SecretCrypto) Decrypt(ciphertext []byte, nonce []byte, version int) (string, error) {
	key, ok := c.keys[version]
	if !ok {
		return "", fmt.Errorf("mcp secret encryption key version %d is not configured", version)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("could not create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("could not create GCM: %w", err)
	}

	pt, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("could not decrypt secret: %w", err)
	}

	return string(pt), nil
}
