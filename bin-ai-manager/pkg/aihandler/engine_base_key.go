package aihandler

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
)

// Packed ciphertext of an engine base key: version(2, big endian) | nonce(12) | ciphertext+tag.
const (
	packedVersionLen = 2
	packedNonceLen   = 12
	packedTagLen     = 16
	packedMinLen     = packedVersionLen + packedNonceLen + packedTagLen
)

var (
	errEngineBaseKeyCryptoUnavailable = errors.New("engine base key encryption is not available")
	errEngineBaseKeyBadVersion        = errors.New("engine base key version is out of range")
	errEngineBaseKeyBadCiphertext     = errors.New("engine base key ciphertext is malformed")
)

// sealEngineBaseKey encrypts the key under the current key version. The errors carry no key material.
func (h *aiHandler) sealEngineBaseKey(key string) ([]byte, error) {
	if h.crypto == nil {
		return nil, errEngineBaseKeyCryptoUnavailable
	}

	ct, nonce, version, err := h.crypto.Encrypt(key)
	if err != nil {
		return nil, errEngineBaseKeyCryptoUnavailable
	}
	if version <= 0 || version > 65535 {
		return nil, errEngineBaseKeyBadVersion
	}
	if len(nonce) != packedNonceLen {
		return nil, errEngineBaseKeyBadCiphertext
	}

	res := make([]byte, 0, packedVersionLen+len(nonce)+len(ct))
	res = binary.BigEndian.AppendUint16(res, uint16(version))
	res = append(res, nonce...)
	res = append(res, ct...)

	return res, nil
}

// openEngineBaseKey decrypts a packed value. The errors carry no key material.
func (h *aiHandler) openEngineBaseKey(packed []byte) (string, error) {
	if h.crypto == nil {
		return "", errEngineBaseKeyCryptoUnavailable
	}
	if len(packed) < packedMinLen {
		return "", errEngineBaseKeyBadCiphertext
	}

	version := int(binary.BigEndian.Uint16(packed[:packedVersionLen]))
	nonce := packed[packedVersionLen : packedVersionLen+packedNonceLen]
	ct := packed[packedVersionLen+packedNonceLen:]

	res, err := h.crypto.Decrypt(ct, nonce, version)
	if err != nil {
		return "", fmt.Errorf("could not decrypt the engine base key (version %d)", version)
	}

	return res, nil
}

// FillEngineBaseKey returns a copy of the AI whose plaintext endpoint key is filled from the
// ciphertext. The input is never modified, so the plaintext can not reach the cache. A decrypt
// failure is logged without secrets and leaves the plaintext empty with the ciphertext set; it
// never fails the caller.
func (h *aiHandler) FillEngineBaseKey(a *ai.AI) *ai.AI {
	if a == nil {
		return nil
	}

	res := *a
	res.EngineBaseKey = ""
	if len(a.EngineBaseKeyEnc) == 0 {
		return &res
	}

	plain, err := h.openEngineBaseKey(a.EngineBaseKeyEnc)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "FillEngineBaseKey",
			"ai_id": a.ID,
		}).Errorf("Could not decrypt the engine base key. err: %v", err)
		return &res
	}
	res.EngineBaseKey = ai.RedactedString(plain)

	return &res
}

// endpointWrite is the decision of section 3.2: which of the two endpoint columns an update writes.
// The URL and the ciphertext are always written by the same UPDATE statement or not at all.
type endpointWrite struct {
	writeURL bool
	url      string
	writeEnc bool
	enc      []byte // nil is NULL
}

// endpointChanged reports whether the update changes the endpoint URL (byte inequality) or switches
// to a custom endpoint model.
func endpointChanged(pre *ai.AI, engineBaseURL string) bool {
	return !ai.IsCustomEndpointModel(pre.EngineModel) || engineBaseURL != pre.EngineBaseURL
}

// decideEndpointWrite computes the column writes once from the stored snapshot (section 3.2).
func (h *aiHandler) decideEndpointWrite(pre *ai.AI, model ai.EngineModel, engineBaseURL string, key *string) (endpointWrite, error) {
	if !ai.IsCustomEndpointModel(model) {
		// switching away (or staying away) clears both
		return endpointWrite{writeURL: true, url: "", writeEnc: true, enc: nil}, nil
	}

	if key != nil && *key != "" {
		enc, err := h.sealEngineBaseKey(*key)
		if err != nil {
			return endpointWrite{}, err
		}
		return endpointWrite{writeURL: true, url: engineBaseURL, writeEnc: true, enc: enc}, nil
	}

	if endpointChanged(pre, engineBaseURL) {
		return endpointWrite{writeURL: true, url: engineBaseURL, writeEnc: true, enc: nil}, nil
	}

	if key != nil {
		// unchanged URL and an explicit empty key: delete the key, leave the URL alone
		return endpointWrite{writeEnc: true, enc: nil}, nil
	}

	return endpointWrite{}, nil
}
