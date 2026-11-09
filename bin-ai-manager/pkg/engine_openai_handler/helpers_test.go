package engine_openai_handler

import "context"

// StaticKey returns a KeyFunc that always hands the given key. It lives in a test file on purpose:
// production code must get its key from the config handler, never from a fixed value.
func StaticKey(key string) KeyFunc {
	return func(context.Context) (string, error) { return key, nil }
}
