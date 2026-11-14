// Package envsecret reads a platform secret from an environment variable.
//
// A variable that is not set, is empty, or still holds an unresolved placeholder of the deployment
// (a value that starts with "[[") means "not configured". The package never returns, logs or
// formats a value other than the one it hands to its caller.
package envsecret

import (
	"os"
	"strings"
)

// State tells what an environment variable holds.
type State int

// list of states
const (
	Unset       State = iota // not set, empty, or only white space
	Set                      // a usable value
	Placeholder              // an unresolved placeholder of the deployment, e.g. a value that starts with "[["
)

// placeholderPrefix starts the text of a deployment variable that was not resolved.
const placeholderPrefix = "[["

// String returns the name of the state.
func (s State) String() string {
	switch s {
	case Set:
		return "set"
	case Placeholder:
		return "placeholder"
	default:
		return "unset"
	}
}

// Lookup returns the trimmed value of the variable and its state. The value is empty unless the
// state is Set, so a placeholder is never handed to a caller as a secret.
func Lookup(name string) (string, State) {
	v := strings.TrimSpace(os.Getenv(name))
	switch {
	case v == "":
		return "", Unset
	case strings.HasPrefix(v, placeholderPrefix):
		return "", Placeholder
	default:
		return v, Set
	}
}
