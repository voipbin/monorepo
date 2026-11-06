package confighandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kind is the role of a key.
type Kind string

// list of kinds
const (
	KindParam  Kind = "param"  // an operational parameter
	KindSecret Kind = "secret" // an encrypted secret
	KindSelect Kind = "select" // a provider selection, normally inside an atomic group
)

// Type is the value type of a key.
type Type string

// list of types
const (
	TypeString   Type = "string"
	TypeInt      Type = "int"
	TypeBool     Type = "bool"
	TypeDuration Type = "duration"
	TypeEnum     Type = "enum"
)

// secretRowType is the value_type of a stored secret.
const secretRowType = "secret"

// Limits derived from the table definition: secret_ciphertext is varbinary(2048) and holds
// nonce (12 bytes) + ciphertext + tag (16 bytes).
const maxSecretLen = 2048 - 12 - 16

var nameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9._]{0,127}$`)

// reservedEnv lists the bootstrap variables. A key can not use them as its Env, because the values
// are needed before the configs can be read.
var reservedEnv = map[string]bool{
	"DATABASE_DSN":              true,
	"DATABASE_DSN_ASTERISK":     true,
	"DATABASE_DSN_BIN":          true,
	"POSTGRESQL_DSN":            true,
	"CLICKHOUSE_ADDRESS":        true,
	"CLICKHOUSE_DATABASE":       true,
	"RABBITMQ_ADDRESS":          true,
	"REDIS_ADDRESS":             true,
	"REDIS_DATABASE":            true,
	"REDIS_DB":                  true,
	"REDIS_PASSWORD":            true,
	"PROMETHEUS_ENDPOINT":       true,
	"PROMETHEUS_LISTEN_ADDRESS": true,
	"LOG_LEVEL":                 true,
}

const reservedEnvSuffix = "_ENCRYPTION_KEYS"

// Key is the declaration of one configuration key.
type Key struct {
	Name string // lower case letters, digits, '.' and '_', up to 128 bytes
	Kind Kind
	Type Type

	// Env is the name of the environment variable that is used when the database has no row.
	// Empty skips the env step.
	Env string

	// Default is used when neither the database nor the env has a value. nil means no default.
	// string for TypeString and TypeEnum, int or int64 for TypeInt, bool, time.Duration.
	Default any

	// Min and Max bound TypeInt (the value) and TypeDuration (nanoseconds).
	Min, Max *int64

	// MaxLen bounds a TypeString and a secret value in bytes. Required for both.
	MaxLen int

	Enum []string // allowed values of TypeEnum

	// Group and Atomic: the keys of an atomic group are resolved together. When the anchor key of
	// the group (its only secret, else its only select key) has a row, every key of the group is
	// read from the database or its default only; otherwise from env or default only.
	Group  string
	Atomic bool

	// Sensitive hides the value of a non secret key in the history. A secret is always sensitive.
	Sensitive bool
}

// hidden reports whether the history must not carry the value.
func (k Key) hidden() bool {
	return k.Kind == KindSecret || k.Sensitive
}

// rowType is the value_type of a row of this key.
func (k Key) rowType() string {
	if k.Kind == KindSecret {
		return secretRowType
	}

	return string(k.Type)
}

// Registry holds the declared keys of one service.
type Registry struct {
	keys   []Key
	byName map[string]int
	anchor map[string]string // atomic group -> anchor key name
}

// NewRegistry validates the declarations. The errors name the key and the rule, never a value.
func NewRegistry(keys []Key) (*Registry, error) {
	res := &Registry{
		keys:   make([]Key, 0, len(keys)),
		byName: map[string]int{},
		anchor: map[string]string{},
	}

	groups := map[string][]Key{}
	envs := map[string]string{}
	for _, k := range keys {
		if err := validateKey(k); err != nil {
			return nil, err
		}
		if _, ok := res.byName[k.Name]; ok {
			return nil, fmt.Errorf("config key %q is declared more than once", k.Name)
		}

		if k.Env != "" {
			if other, ok := envs[k.Env]; ok {
				return nil, fmt.Errorf("config keys %q and %q declare the same environment name", other, k.Name)
			}
			envs[k.Env] = k.Name
		}

		k.Default = normalizeDefault(k)
		res.byName[k.Name] = len(res.keys)
		res.keys = append(res.keys, k)

		if k.Group != "" {
			groups[k.Group] = append(groups[k.Group], k)
		}
	}

	for name, members := range groups {
		atomic := members[0].Atomic
		for _, m := range members {
			if m.Atomic != atomic {
				return nil, fmt.Errorf("config group %q mixes atomic and non atomic keys", name)
			}
		}
		if !atomic {
			continue
		}

		anchor, err := findAnchor(name, members)
		if err != nil {
			return nil, err
		}
		res.anchor[name] = anchor
	}

	return res, nil
}

func findAnchor(group string, members []Key) (string, error) {
	var secrets, selects []string
	for _, m := range members {
		switch m.Kind {
		case KindSecret:
			secrets = append(secrets, m.Name)
		case KindSelect:
			selects = append(selects, m.Name)
		}
	}

	switch {
	case len(secrets) == 1:
		return secrets[0], nil
	case len(secrets) > 1:
		return "", fmt.Errorf("config group %q has more than one secret key, the anchor is ambiguous", group)
	case len(selects) == 1:
		return selects[0], nil
	}

	return "", fmt.Errorf("config group %q needs exactly one secret key or one select key as its anchor", group)
}

func validateKey(k Key) error {
	for _, check := range []func(Key) error{
		validateIdentity, validateEnv, validateShape, validateEnum, validateSecretRules, validateDefault,
	} {
		if err := check(k); err != nil {
			return err
		}
	}

	return nil
}

// validateIdentity checks the name, the kind and the type.
func validateIdentity(k Key) error {
	if !nameRegexp.MatchString(k.Name) {
		return fmt.Errorf("config key name %q is malformed", k.Name)
	}

	switch k.Kind {
	case KindParam, KindSecret, KindSelect:
	default:
		return fmt.Errorf("config key %q has an unknown kind", k.Name)
	}

	switch k.Type {
	case TypeString, TypeInt, TypeBool, TypeDuration, TypeEnum:
	default:
		return fmt.Errorf("config key %q has an unknown type", k.Name)
	}

	return nil
}

func validateEnv(k Key) error {
	if k.Env != "" && (reservedEnv[k.Env] || strings.HasSuffix(k.Env, reservedEnvSuffix)) {
		return fmt.Errorf("config key %q uses a bootstrap environment variable name", k.Name)
	}

	return nil
}

// validateShape checks the group, bool placement and the numeric bounds.
func validateShape(k Key) error {
	if k.Atomic && k.Group == "" {
		return fmt.Errorf("config key %q is atomic but has no group", k.Name)
	}

	if k.Type == TypeBool && (k.Kind != KindSelect || !k.Atomic) {
		return fmt.Errorf("config key %q: a bool is allowed only as a select key inside an atomic group", k.Name)
	}

	if (k.Min != nil || k.Max != nil) && k.Type != TypeInt && k.Type != TypeDuration {
		return fmt.Errorf("config key %q: min and max apply to int and duration only", k.Name)
	}
	if k.Min != nil && k.Max != nil && *k.Min > *k.Max {
		return fmt.Errorf("config key %q: min is greater than max", k.Name)
	}

	return nil
}

func validateEnum(k Key) error {
	if k.Type == TypeEnum && len(k.Enum) == 0 {
		return fmt.Errorf("config key %q: an enum needs values", k.Name)
	}
	if k.Type != TypeEnum && len(k.Enum) > 0 {
		return fmt.Errorf("config key %q: enum values apply to the enum type only", k.Name)
	}

	seen := map[string]bool{}
	for _, e := range k.Enum {
		if e == "" || seen[e] {
			return fmt.Errorf("config key %q: enum values must be non-empty and unique", k.Name)
		}
		seen[e] = true
	}

	return nil
}

// validateSecretRules checks the secret and the string limits.
func validateSecretRules(k Key) error {
	if k.Kind != KindSecret {
		if k.Type == TypeString && k.MaxLen < 1 {
			return fmt.Errorf("config key %q: a string needs max_len", k.Name)
		}

		return nil
	}

	if k.Type != TypeString {
		return fmt.Errorf("config key %q: a secret must be a string", k.Name)
	}
	if k.Default != nil {
		return fmt.Errorf("config key %q: a secret can not have a default", k.Name)
	}
	if k.MaxLen < 1 || k.MaxLen > maxSecretLen {
		return fmt.Errorf("config key %q: a secret needs max_len between 1 and %d", k.Name, maxSecretLen)
	}

	return nil
}

func validateDefault(k Key) error {
	if k.Default == nil {
		return nil
	}
	if _, err := checkTyped(k, k.Default); err != nil {
		return fmt.Errorf("config key %q: the default does not fit the declaration", k.Name)
	}

	return nil
}

// Keys returns the declarations in declaration order.
func (r *Registry) Keys() []Key {
	res := make([]Key, len(r.keys))
	copy(res, r.keys)

	return res
}

// Lookup returns the declaration of a key.
func (r *Registry) Lookup(name string) (Key, bool) {
	idx, ok := r.byName[name]
	if !ok {
		return Key{}, false
	}

	return r.keys[idx], true
}

// anchorOf returns the anchor key name of the atomic group of k, or k.Name when k is not atomic.
func (r *Registry) anchorOf(k Key) string {
	if !k.Atomic {
		return k.Name
	}

	return r.anchor[k.Group]
}

// normalizeDefault converts the accepted default forms to the canonical Go type.
func normalizeDefault(k Key) any {
	if k.Default == nil {
		return nil
	}
	if v, err := checkTyped(k, k.Default); err == nil {
		return v
	}

	return k.Default
}

// checkTyped converts v to the canonical type of k.Type and checks the range.
func checkTyped(k Key, v any) (any, error) {
	switch k.Type {
	case TypeString:
		s, ok := v.(string)
		if !ok {
			return nil, ErrInvalid
		}
		if k.MaxLen > 0 && len(s) > k.MaxLen {
			return nil, ErrInvalid
		}
		return s, nil

	case TypeEnum:
		s, ok := v.(string)
		if !ok {
			return nil, ErrInvalid
		}
		for _, e := range k.Enum {
			if e == s {
				return s, nil
			}
		}
		return nil, ErrInvalid

	case TypeInt:
		var n int64
		switch t := v.(type) {
		case int:
			n = int64(t)
		case int64:
			n = t
		default:
			return nil, ErrInvalid
		}
		if !inRange(k, n) {
			return nil, ErrInvalid
		}
		return n, nil

	case TypeBool:
		b, ok := v.(bool)
		if !ok {
			return nil, ErrInvalid
		}
		return b, nil

	case TypeDuration:
		d, ok := v.(time.Duration)
		if !ok {
			return nil, ErrInvalid
		}
		if !inRange(k, int64(d)) {
			return nil, ErrInvalid
		}
		return d, nil
	}

	return nil, ErrInvalid
}

func inRange(k Key, n int64) bool {
	if k.Min != nil && n < *k.Min {
		return false
	}
	if k.Max != nil && n > *k.Max {
		return false
	}

	return true
}

// parse converts text (a flag value or an environment value) to the typed value.
func (k Key) parse(raw string) (any, error) {
	switch k.Type {
	case TypeString, TypeEnum:
		return checkTyped(k, raw)

	case TypeInt:
		n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return nil, ErrInvalid
		}
		return checkTyped(k, n)

	case TypeBool:
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, ErrInvalid
		}
		return b, nil

	case TypeDuration:
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return nil, ErrInvalid
		}
		return checkTyped(k, d)
	}

	return nil, ErrInvalid
}

// encodeJSON returns the JSON stored in value_json: a string, a number or a bool; a duration is
// stored in its string form.
func (k Key) encodeJSON(v any) ([]byte, error) {
	if d, ok := v.(time.Duration); ok {
		return json.Marshal(d.String())
	}

	return json.Marshal(v)
}

// decodeJSON is the reverse of encodeJSON and validates the value against the declaration.
func (k Key) decodeJSON(b []byte) (any, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil, ErrInvalid
	}

	switch k.Type {
	case TypeString, TypeEnum:
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, ErrInvalid
		}
		return checkTyped(k, s)

	case TypeInt:
		var n int64
		if err := json.Unmarshal(b, &n); err != nil {
			return nil, ErrInvalid
		}
		return checkTyped(k, n)

	case TypeBool:
		var v bool
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, ErrInvalid
		}
		return v, nil

	case TypeDuration:
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, ErrInvalid
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, ErrInvalid
		}
		return checkTyped(k, d)
	}

	return nil, ErrInvalid
}
