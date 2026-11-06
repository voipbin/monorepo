// Package secretcrypto encrypts small secrets with AES-256-GCM under a versioned key list.
//
// The key list format is "<version>:<base64 of 32 bytes>[,...]". The highest version is used for
// new writes and every configured version stays available for reading the rows written under it.
// The additional authenticated data (AAD) is optional: a nil AAD is the legacy mode that is
// byte-identical to gcm.Seal(nil, nonce, plaintext, nil).
//
// Three stored layouts exist in the services; the adapter functions map them to the standard
// blob (nonce || ciphertext || tag) without any data migration:
//
//   - the standard blob and a version column (number_providers, the configs tables)
//   - ciphertext and nonce in two columns (ai_mcp_servers): JoinSplit and SplitBlob
//   - version (2 bytes, big endian) || nonce || ciphertext in one column (engine base key):
//     PackVersioned and UnpackVersioned
//
// No error returned by this package contains key text, key list text, plaintext or ciphertext.
package secretcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Sentinel errors. The package does not depend on any service package; services map the
// sentinels to their own errors.
var (
	// ErrKeyNotConfigured is returned when no key is configured.
	ErrKeyNotConfigured = errors.New("secret encryption key is not configured")

	// ErrTooLong is returned when the plaintext is longer than the configured limit.
	ErrTooLong = errors.New("secret is longer than the limit")

	// ErrDecrypt is returned when the blob cannot be decrypted: unknown key version, too short,
	// or the authentication failed (wrong key, wrong AAD, modified data).
	ErrDecrypt = errors.New("could not decrypt the secret")

	// ErrInvalidKeySet is returned when the key list cannot be parsed.
	ErrInvalidKeySet = errors.New("invalid encryption key list")

	// ErrMalformed is returned when a stored layout is not valid.
	ErrMalformed = errors.New("malformed ciphertext layout")
)

const (
	keyLen     = 32
	nonceLen   = 12
	tagLen     = 16
	minBlobLen = nonceLen + tagLen

	versionLen       = 2
	maxPackedVersion = 65535
	minPackedLen     = versionLen + minBlobLen
)

// Crypto holds the parsed key list.
type Crypto struct {
	keys           map[int][]byte
	currentVersion int
	maxPlaintext   int
}

// Option configures New.
type Option func(*Crypto)

// WithMaxPlaintext sets the longest plaintext in bytes that Encrypt accepts.
// n <= 0 means unlimited, which is the default.
func WithMaxPlaintext(n int) Option {
	return func(c *Crypto) {
		c.maxPlaintext = n
	}
}

// New parses "<version>:<base64 32 byte key>[,...]". An empty string is valid and means that no
// key is configured: Encrypt fails with ErrKeyNotConfigured and nothing can be decrypted.
// Errors wrap ErrInvalidKeySet and name only the position of the entry, never its text.
func New(raw string, opts ...Option) (*Crypto, error) {
	res := &Crypto{keys: map[int][]byte{}}
	for _, opt := range opts {
		opt(res)
	}

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
			return nil, fmt.Errorf("%w: entry %d is malformed (expected <version>:<base64-key>)", ErrInvalidKeySet, i)
		}

		version, err := strconv.Atoi(strings.TrimSpace(entry[:idx]))
		if err != nil {
			return nil, fmt.Errorf("%w: entry %d has a version that is not an integer", ErrInvalidKeySet, i)
		}
		if version <= 0 {
			return nil, fmt.Errorf("%w: entry %d has a version that is not greater than 0", ErrInvalidKeySet, i)
		}

		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entry[idx+1:]))
		if err != nil {
			return nil, fmt.Errorf("%w: entry %d is not valid base64", ErrInvalidKeySet, i)
		}
		if len(key) != keyLen {
			return nil, fmt.Errorf("%w: entry %d must decode to exactly %d bytes", ErrInvalidKeySet, i, keyLen)
		}

		if _, exists := res.keys[version]; exists {
			return nil, fmt.Errorf("%w: entry %d repeats a version", ErrInvalidKeySet, i)
		}
		res.keys[version] = key
	}

	if len(res.keys) == 0 {
		return nil, fmt.Errorf("%w: the list has no entry", ErrInvalidKeySet)
	}

	versions := res.Versions()
	if len(versions) > 0 {
		res.currentVersion = versions[len(versions)-1]
	}

	return res, nil
}

// HasKey returns true if at least one key is configured. It is safe to call on a nil Crypto.
func (c *Crypto) HasKey() bool {
	return c != nil && c.currentVersion != 0
}

// CurrentVersion returns the version used for new writes, 0 if no key is configured.
func (c *Crypto) CurrentVersion() int {
	if c == nil {
		return 0
	}
	return c.currentVersion
}

// Versions returns the configured key versions in ascending order.
func (c *Crypto) Versions() []int {
	if c == nil {
		return nil
	}

	res := make([]int, 0, len(c.keys))
	for v := range c.keys {
		res = append(res, v)
	}
	sort.Ints(res)

	return res
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

// Encrypt encrypts the plaintext under the current key and returns nonce || ciphertext || tag and
// the key version. aad binds the ciphertext to its row; nil is the legacy no-AAD mode.
func (c *Crypto) Encrypt(plaintext string, aad []byte) ([]byte, int, error) {
	if !c.HasKey() {
		return nil, 0, ErrKeyNotConfigured
	}
	if c.maxPlaintext > 0 && len(plaintext) > c.maxPlaintext {
		return nil, 0, ErrTooLong
	}

	gcm, err := newGCM(c.keys[c.currentVersion])
	if err != nil {
		return nil, 0, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, 0, errors.New("could not generate a nonce")
	}

	return gcm.Seal(nonce, nonce, []byte(plaintext), aad), c.currentVersion, nil
}

// Decrypt decrypts a blob produced by Encrypt using the key of the given (row) version, never the
// current version, so rotating the key list does not break older rows.
// A nil Crypto or a Crypto without any key returns ErrKeyNotConfigured; every other failure
// returns ErrDecrypt.
func (c *Crypto) Decrypt(blob []byte, version int, aad []byte) (string, error) {
	if !c.HasKey() {
		return "", ErrKeyNotConfigured
	}

	key, ok := c.keys[version]
	if !ok {
		return "", ErrDecrypt
	}

	gcm, err := newGCM(key)
	if err != nil {
		return "", ErrDecrypt
	}

	if len(blob) < minBlobLen {
		return "", ErrDecrypt
	}

	pt, err := gcm.Open(nil, blob[:nonceLen], blob[nonceLen:], aad)
	if err != nil {
		return "", ErrDecrypt
	}

	return string(pt), nil
}

// AAD builds the additional authenticated data of a configs row: the UTF-8 bytes of the table
// name, a single 0x00 byte, then the UTF-8 bytes of the key name. The rule is a storage contract
// and must not change once data exists.
func AAD(table, key string) []byte {
	res := make([]byte, 0, len(table)+1+len(key))
	res = append(res, table...)
	res = append(res, 0x00)
	res = append(res, key...)

	return res
}

// AADLiteral returns the bytes of s as is (for example the provider name of number_providers).
func AADLiteral(s string) []byte {
	return []byte(s)
}

// JoinSplit joins the two columns of the ciphertext-and-nonce layout into the standard blob.
// Both empty means that no secret is stored and returns (nil, nil). Only one of them empty, a
// nonce that is not 12 bytes, or a ciphertext shorter than the 16 byte tag is ErrMalformed.
func JoinSplit(ct, nonce []byte) ([]byte, error) {
	if len(ct) == 0 && len(nonce) == 0 {
		return nil, nil
	}
	if len(nonce) != nonceLen || len(ct) < tagLen {
		return nil, ErrMalformed
	}

	res := make([]byte, 0, len(nonce)+len(ct))
	res = append(res, nonce...)
	res = append(res, ct...)

	return res, nil
}

// SplitBlob splits the standard blob into the ciphertext and the nonce columns.
func SplitBlob(blob []byte) ([]byte, []byte, error) {
	if len(blob) < minBlobLen {
		return nil, nil, ErrMalformed
	}

	nonce := make([]byte, nonceLen)
	copy(nonce, blob[:nonceLen])

	ct := make([]byte, len(blob)-nonceLen)
	copy(ct, blob[nonceLen:])

	return ct, nonce, nil
}

// PackVersioned prefixes the standard blob with the key version (2 bytes, big endian). The
// version must be 1..65535.
func PackVersioned(version int, blob []byte) ([]byte, error) {
	if version <= 0 || version > maxPackedVersion {
		return nil, ErrMalformed
	}
	if len(blob) < minBlobLen {
		return nil, ErrMalformed
	}

	res := make([]byte, 0, versionLen+len(blob))
	res = binary.BigEndian.AppendUint16(res, uint16(version))
	res = append(res, blob...)

	return res, nil
}

// UnpackVersioned splits a value made by PackVersioned into the version and the standard blob.
func UnpackVersioned(packed []byte) (int, []byte, error) {
	if len(packed) < minPackedLen {
		return 0, nil, ErrMalformed
	}

	version := int(binary.BigEndian.Uint16(packed[:versionLen]))
	if version == 0 {
		return 0, nil, ErrMalformed
	}

	blob := make([]byte, len(packed)-versionLen)
	copy(blob, packed[versionLen:])

	return version, blob, nil
}
