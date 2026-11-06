package providerhandler

import (
	"errors"
	"fmt"
	"strings"

	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
)

// maxSecretBytes is the longest provider secret that is accepted and encrypted.
const maxSecretBytes = 1024

// NewSecretCrypto parses the key list of the provider secrets ("<version>:<base64 32 byte key>[,...]").
//
// An empty list, and a list whose entries are all empty after trimming, means that no key is
// configured: nothing can be decrypted and Encrypt fails with provider.ErrSecretKeyNotConfigured.
// No limit is set on that value because Encrypt fails on the missing key first. The error of a
// list that cannot be parsed wraps secretcrypto.ErrInvalidKeySet and never holds key text.
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

	return secretcrypto.New(raw, secretcrypto.WithMaxPlaintext(maxSecretBytes))
}

// encryptSecret encrypts the secret of a provider; the provider name is the AAD that binds the
// ciphertext to its row. The library errors are mapped to the errors of the provider package.
func encryptSecret(c *secretcrypto.Crypto, secret string, name number.ProviderName) ([]byte, int, error) {
	res, version, err := c.Encrypt(secret, secretcrypto.AADLiteral(string(name)))
	switch {
	case errors.Is(err, secretcrypto.ErrKeyNotConfigured):
		return nil, 0, provider.ErrSecretKeyNotConfigured

	case errors.Is(err, secretcrypto.ErrTooLong):
		return nil, 0, fmt.Errorf("%w: the secret is longer than %d bytes", provider.ErrInvalidArgument, maxSecretBytes)
	}

	return res, version, err
}

// decryptSecret decrypts the secret of a provider row with the key of the version stored in the row.
func decryptSecret(c *secretcrypto.Crypto, blob []byte, version int, name number.ProviderName) (string, error) {
	return c.Decrypt(blob, version, secretcrypto.AADLiteral(string(name)))
}
