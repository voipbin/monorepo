// Package secretcrypto encrypts provider secrets with AES-256-GCM under a versioned key set.
// The pattern follows the one used by other services; it is kept here on purpose and is to be
// shared only when a third consumer appears.
package secretcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"monorepo/bin-number-manager/models/provider"
)

// MaxPlaintextBytes is the longest secret that can be encrypted.
const MaxPlaintextBytes = 1024

// SecretCrypto holds the parsed key set. The highest version is used for new writes and every
// configured version stays available for decrypting the rows written under it.
type SecretCrypto struct {
	keys           map[int][]byte
	currentVersion int
}

// New parses "<version>:<base64 32 byte key>[,...]". An empty string is valid and means that
// no key is configured: Encrypt fails and nothing can be decrypted. Error messages never
// contain the key material.
func New(raw string) (*SecretCrypto, error) {
	res := &SecretCrypto{keys: map[int][]byte{}}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return res, nil
	}

	for i, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		idx := strings.Index(entry, ":")
		if idx < 0 {
			return nil, fmt.Errorf("encryption key entry %d is malformed (expected <version>:<base64-key>)", i)
		}

		version, err := strconv.Atoi(strings.TrimSpace(entry[:idx]))
		if err != nil {
			return nil, fmt.Errorf("encryption key entry %d has a version that is not an integer", i)
		}
		if version <= 0 {
			return nil, fmt.Errorf("encryption key entry %d has a version that is not greater than 0", i)
		}

		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entry[idx+1:]))
		if err != nil {
			return nil, fmt.Errorf("encryption key version %d is not valid base64", version)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("encryption key version %d must decode to exactly 32 bytes", version)
		}

		if _, exists := res.keys[version]; exists {
			return nil, fmt.Errorf("encryption key version %d is configured more than once", version)
		}
		res.keys[version] = key
	}

	versions := make([]int, 0, len(res.keys))
	for v := range res.keys {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	if len(versions) > 0 {
		res.currentVersion = versions[len(versions)-1]
	}

	return res, nil
}

// HasKey returns true if at least one key is configured.
func (c *SecretCrypto) HasKey() bool {
	return c != nil && c.currentVersion != 0
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("could not create the cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("could not create the cipher")
	}

	return gcm, nil
}

// Encrypt encrypts the plaintext under the current key and returns nonce || ciphertext and the
// key version. aad (the provider name) binds the ciphertext to its row.
func (c *SecretCrypto) Encrypt(plaintext string, aad string) ([]byte, int, error) {
	if !c.HasKey() {
		return nil, 0, provider.ErrSecretKeyNotConfigured
	}
	if len(plaintext) > MaxPlaintextBytes {
		return nil, 0, fmt.Errorf("%w: the secret is longer than %d bytes", provider.ErrInvalidArgument, MaxPlaintextBytes)
	}

	gcm, err := newGCM(c.keys[c.currentVersion])
	if err != nil {
		return nil, 0, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, 0, errors.New("could not generate a nonce")
	}

	res := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(aad))
	return res, c.currentVersion, nil
}

// Decrypt decrypts data produced by Encrypt using the key of the given (row) version.
func (c *SecretCrypto) Decrypt(data []byte, version int, aad string) (string, error) {
	if c == nil {
		return "", errors.New("no encryption key is configured")
	}

	key, ok := c.keys[version]
	if !ok {
		return "", fmt.Errorf("encryption key version %d is not configured", version)
	}

	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize() {
		return "", errors.New("the ciphertext is too short")
	}

	pt, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], []byte(aad))
	if err != nil {
		return "", errors.New("could not decrypt the secret")
	}

	return string(pt), nil
}
