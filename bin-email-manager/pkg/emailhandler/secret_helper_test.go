package emailhandler

import "context"

// staticSecret returns a SecretFunc that always returns v.
func staticSecret(v string) SecretFunc {
	return func(context.Context) (string, error) {
		return v, nil
	}
}
