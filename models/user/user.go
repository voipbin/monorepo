package user

// User data model
type User struct {
	// gorm.Model
	ID           uint64 `gorm:"primary_key" json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`

	TMCreate string `gorm:"column:tm_create" json:"tm_create"`
	TMUpdate string `gorm:"column:tm_update" json:"tm_update"`
	TMDelete string `gorm:"column:tm_delete" json:"tm_delete"`
}

// Serialize serializes user data
func (u *User) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
	}
}

// Read reads the user info
func (u *User) Read(m map[string]interface{}) {
	u.ID = uint64(m["id"].(float64))
	u.Username = m["username"].(string)
}
