# Address Validation Functions Design

## Overview

Add validation functions to `bin-common-handler/models/address` package that validate the `Target` field format based on the `Type`. These validate the **normalized form** of addresses (after parsing/transformation).

## Validation Rules

| Type | Target Validation |
|------|-------------------|
| `TypeTel` | E.164: `+` followed by 7-15 digits |
| `TypeEmail` | RFC 5322 via `mail.ParseAddress()` |
| `TypeSIP` | `user@domain` format (contains `@`, non-empty user and domain parts) |
| `TypeAgent` | Valid UUID |
| `TypeConference` | Valid UUID |
| `TypeLine` | Valid UUID |
| `TypeExtension` | Valid UUID |
| `TypeNone` | Always valid (no target expected) |

## API Design

```go
// In bin-common-handler/models/address/validate.go

// Validate validates the Address Target field based on Type.
// Returns nil if valid, error with details if invalid.
func (a *Address) Validate() error

// ValidateTarget validates a target string for a specific type.
// Useful when validating before constructing an Address.
func ValidateTarget(addressType Type, target string) error

// Individual type validators (unexported, used internally)
func validateTel(target string) error      // E.164: + followed by 7-15 digits
func validateEmail(target string) error    // RFC 5322 via mail.ParseAddress
func validateSIP(target string) error      // user@domain format
func validateUUID(target string) error     // Valid UUID format
```

### Usage Examples

```go
// Validate an existing address
addr := &Address{Type: TypeTel, Target: "+14155551234"}
if err := addr.Validate(); err != nil {
    return fmt.Errorf("invalid address: %w", err)
}

// Validate before construction
if err := ValidateTarget(TypeEmail, "user@example.com"); err != nil {
    return err
}
```

## Implementation

```go
// bin-common-handler/models/address/validate.go

package address

import (
    "fmt"
    "net/mail"
    "regexp"
    "strings"

    "github.com/gofrs/uuid"
)

// Regex patterns
var (
    telRegex = regexp.MustCompile(`^\+[0-9]{7,15}$`)
)

// Validate validates the Address Target field based on Type.
func (a *Address) Validate() error {
    return ValidateTarget(a.Type, a.Target)
}

// ValidateTarget validates a target string for a specific type.
func ValidateTarget(addressType Type, target string) error {
    switch addressType {
    case TypeNone:
        return nil
    case TypeTel:
        return validateTel(target)
    case TypeEmail:
        return validateEmail(target)
    case TypeSIP:
        return validateSIP(target)
    case TypeAgent, TypeConference, TypeLine, TypeExtension:
        return validateUUID(target)
    default:
        return fmt.Errorf("unknown address type: %s", addressType)
    }
}

func validateTel(target string) error {
    if !telRegex.MatchString(target) {
        return fmt.Errorf("invalid tel format: must be + followed by 7-15 digits")
    }
    return nil
}

func validateEmail(target string) error {
    _, err := mail.ParseAddress(target)
    if err != nil {
        return fmt.Errorf("invalid email format: %w", err)
    }
    return nil
}

func validateSIP(target string) error {
    parts := strings.SplitN(target, "@", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        return fmt.Errorf("invalid sip format: must be user@domain")
    }
    return nil
}

func validateUUID(target string) error {
    if uuid.FromStringOrNil(target) == uuid.Nil {
        return fmt.Errorf("invalid uuid format")
    }
    return nil
}
```

## Test Cases

```go
// bin-common-handler/models/address/validate_test.go

func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        address Address
        wantErr bool
    }{
        // TypeTel
        {"tel valid min", Address{Type: TypeTel, Target: "+1234567"}, false},
        {"tel valid max", Address{Type: TypeTel, Target: "+123456789012345"}, false},
        {"tel valid us", Address{Type: TypeTel, Target: "+14155551234"}, false},
        {"tel valid kr", Address{Type: TypeTel, Target: "+821012345678"}, false},
        {"tel missing plus", Address{Type: TypeTel, Target: "14155551234"}, true},
        {"tel too short", Address{Type: TypeTel, Target: "+123456"}, true},
        {"tel too long", Address{Type: TypeTel, Target: "+1234567890123456"}, true},
        {"tel with letters", Address{Type: TypeTel, Target: "+1415555abcd"}, true},
        {"tel empty", Address{Type: TypeTel, Target: ""}, true},

        // TypeEmail
        {"email valid", Address{Type: TypeEmail, Target: "user@example.com"}, false},
        {"email valid with name", Address{Type: TypeEmail, Target: "User <user@example.com>"}, false},
        {"email missing at", Address{Type: TypeEmail, Target: "userexample.com"}, true},
        {"email missing domain", Address{Type: TypeEmail, Target: "user@"}, true},
        {"email empty", Address{Type: TypeEmail, Target: ""}, true},

        // TypeSIP
        {"sip valid", Address{Type: TypeSIP, Target: "user@example.com"}, false},
        {"sip valid with port", Address{Type: TypeSIP, Target: "user@example.com:5060"}, false},
        {"sip missing at", Address{Type: TypeSIP, Target: "userexample.com"}, true},
        {"sip missing user", Address{Type: TypeSIP, Target: "@example.com"}, true},
        {"sip missing domain", Address{Type: TypeSIP, Target: "user@"}, true},
        {"sip empty", Address{Type: TypeSIP, Target: ""}, true},

        // TypeAgent (UUID)
        {"agent valid", Address{Type: TypeAgent, Target: "a04a1f51-2495-48a5-9012-8081aa90b902"}, false},
        {"agent invalid uuid", Address{Type: TypeAgent, Target: "not-a-uuid"}, true},
        {"agent empty", Address{Type: TypeAgent, Target: ""}, true},

        // TypeConference (UUID)
        {"conference valid", Address{Type: TypeConference, Target: "34613ee5-5456-40fe-bb3b-395254270a9d"}, false},
        {"conference invalid", Address{Type: TypeConference, Target: "invalid"}, true},

        // TypeLine (UUID)
        {"line valid", Address{Type: TypeLine, Target: "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7"}, false},
        {"line invalid", Address{Type: TypeLine, Target: "invalid"}, true},

        // TypeExtension (UUID)
        {"extension valid", Address{Type: TypeExtension, Target: "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"}, false},
        {"extension invalid", Address{Type: TypeExtension, Target: "2000"}, true},

        // TypeNone
        {"none empty", Address{Type: TypeNone, Target: ""}, false},
        {"none with target", Address{Type: TypeNone, Target: "anything"}, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.address.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## File Structure

```
bin-common-handler/models/address/
├── main.go           # Existing - Address struct and Type constants
├── validate.go       # NEW - Validation functions
└── validate_test.go  # NEW - Validation tests
```

## Dependencies

All dependencies are already used in the codebase:
- `net/mail` - Standard library for email parsing
- `regexp` - Standard library for regex matching
- `strings` - Standard library for string operations
- `github.com/gofrs/uuid` - Already used throughout the monorepo

## Impact

This is a new feature addition with no breaking changes:
- Adds new exported functions to `bin-common-handler/models/address`
- Does not modify existing behavior
- Services can adopt validation incrementally
