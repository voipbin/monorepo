package confighandler

import (
	"fmt"
	"io"
)

const redacted = "[REDACTED]"

// Secret holds a secret value. It prints as [REDACTED] with every fmt verb and in JSON and text
// encodings; Reveal is the only way to read the value.
type Secret struct {
	v string
}

// NewSecret wraps a value.
func NewSecret(v string) Secret {
	return Secret{v: v}
}

// Reveal returns the value.
func (s Secret) Reveal() string {
	return s.v
}

// String implements fmt.Stringer.
func (s Secret) String() string {
	return redacted
}

// GoString implements fmt.GoStringer.
func (s Secret) GoString() string {
	return redacted
}

// Format implements fmt.Formatter so that no verb (%v, %+v, %#v, %s, %q, %x, ...) prints the value.
func (s Secret) Format(f fmt.State, verb rune) {
	_, _ = io.WriteString(f, redacted)
}

// MarshalJSON implements json.Marshaler.
func (s Secret) MarshalJSON() ([]byte, error) {
	return []byte(`"` + redacted + `"`), nil
}

// MarshalText implements encoding.TextMarshaler.
func (s Secret) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}
