// Package mcpserverhandler owns CRUD for customer-registered McpServer rows,
// their secret envelope encryption, and the SSRF guard applied at
// create/update time. See docs/plans/2026-09-11-mcp-tool-integration-design.md.
package mcpserverhandler

import (
	"strings"

	"monorepo/bin-common-handler/pkg/secretcrypto"
)

// NewSecretCrypto parses the "<version>:<base64-32-byte-key>[,...]" config string of the shared
// secretcrypto package (MCP_SECRET_ENCRYPTION_KEYS and ENGINE_BASE_KEY_ENCRYPTION_KEYS).
//
// An empty string, and a list whose entries are all blank, means that no key is configured yet:
// Encrypt then fails and nothing can be decrypted. Any other malformed list is an error, so the
// caller fails closed at boot; the error never holds the text of the list.
func NewSecretCrypto(raw string) (*secretcrypto.Crypto, error) {
	blank := true
	for _, entry := range strings.Split(raw, ",") {
		if strings.TrimSpace(entry) != "" {
			blank = false
			break
		}
	}
	if blank {
		return secretcrypto.New("")
	}

	return secretcrypto.New(raw)
}

// EncryptSecret encrypts the plaintext under the current (highest-version) key and returns it in the
// layout of the ai_mcp_servers columns: ciphertext (with the tag) and nonce, plus the key version
// to store next to them.
func EncryptSecret(c *secretcrypto.Crypto, plaintext string) (ciphertext []byte, nonce []byte, version int, err error) {
	blob, version, err := c.Encrypt(plaintext)
	if err != nil {
		return nil, nil, 0, err
	}

	ciphertext, nonce, err = secretcrypto.SplitBlob(blob)
	if err != nil {
		return nil, nil, 0, err
	}

	return ciphertext, nonce, version, nil
}

// DecryptSecret decrypts the ciphertext and nonce columns of a row with the key of the row's own
// version, never the current one, so rotating the key list does not break older rows. It
// is also the single place that checks the two columns: a missing or malformed column is an error.
func DecryptSecret(c *secretcrypto.Crypto, ciphertext []byte, nonce []byte, version int) (string, error) {
	blob, err := secretcrypto.JoinSplit(ciphertext, nonce)
	if err != nil {
		return "", err
	}

	return c.Decrypt(blob, version)
}
