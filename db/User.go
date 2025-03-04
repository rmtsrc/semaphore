package db

import (
	"time"
)

// User is the model for an entity which has access to the API
type User struct {
	ID       int       `db:"id" json:"id"`
	Created  time.Time `db:"created" json:"created"`
	Username string    `db:"username" json:"username" binding:"required"`
	Name     string    `db:"name" json:"name" binding:"required"`
	Email    string    `db:"email" json:"email" binding:"required"`
	Password string    `db:"password" json:"-"` // password hash
	Admin    bool      `db:"admin" json:"admin"`
	External bool      `db:"external" json:"external"`
	Alert    bool      `db:"alert" json:"alert"`
	Pro      bool      `db:"pro" json:"pro"`

	Totp *UserTotp `db:"-" json:"totp,omitempty"`
}

type UserTotp struct {
	ID           int       `db:"id" json:"id"`
	Created      time.Time `db:"created" json:"created"`
	UserID       int       `db:"user_id" json:"user_id"`
	URL          string    `db:"url" json:"url"`
	RecoveryHash string    `db:"recovery_hash" json:"-"`
	RecoveryCode string    `db:"-" json:"recovery_code,omitempty"`
}

type UserWithProjectRole struct {
	Role ProjectUserRole `db:"role" json:"role"`
	User
}

// UserWithPwd extends User structure with field for unhashed password received from JSON.
type UserWithPwd struct {
	Pwd string `db:"-" json:"password"` // unhashed password from JSON
	User
}

func ValidateUser(user User) error {
	if user.Username == "" {
		return &ValidationError{Message: "Username cannot be empty"}
	}
	if user.Email == "" {
		return &ValidationError{Message: "Email cannot be empty"}
	}
	if user.Name == "" {
		return &ValidationError{Message: "Name cannot be empty"}
	}
	return nil
}
