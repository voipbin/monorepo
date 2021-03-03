package models

// User data model
type User struct {
	// gorm.Model
	ID           uint64 `json:"id"`       // User's ID
	Username     string `json:"username"` // User's username
	PasswordHash string `json:"-"`        // Hashed Password

	Permission UserPermission `json:"permission"` // User's permission.

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// Serialize serializes user data
// Used it for JWT generation.
func (u *User) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	}
}

// Read reads the user info
func (u *User) Read(m map[string]interface{}) {
	u.ID = uint64(m["id"].(float64))
	u.Username = m["username"].(string)
	u.Permission = UserPermission(m["permission"].(float64))
}

// HasPermission returns true if the user has the given permission
func (u *User) HasPermission(perm UserPermission) bool {

	if u.Permission&perm == 0 {
		return false
	}
	return true
}

// UserPermission type
type UserPermission uint64

// Permission
const (
	UserPermissionNone  UserPermission = 0x0000
	UserPermissionAdmin UserPermission = 0x0001
)
