package user

// User data model
type User struct {
	// gorm.Model
	ID           uint64 `json:"id"`       // User's ID
	Username     string `json:"username"` // User's username
	PasswordHash string `json:"-"`        // Hashed Password

	Permission Permission `json:"permission"` // User's permission.

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
	u.Permission = Permission(m["permission"].(float64))
}

// HasPermission returns true if the user has the given permission
func (u *User) HasPermission(perm Permission) bool {
	return u.Permission&perm != 0
}

// Permission type
type Permission uint64

// Permission
const (
	PermissionNone  Permission = 0x0000
	PermissionAdmin Permission = 0x0001
)
