package domain

import (
	"fmt"
	"strings"
	"time"
)

// Role is the authorization level of a user account.
type Role string

const (
	// RoleAdministrator manages the registries and the workshop flow.
	RoleAdministrator Role = "ADMINISTRATOR"
	// RoleTechnician writes on the service orders assigned to them.
	RoleTechnician Role = "TECHNICIAN"
)

// Valid reports whether the role is one the product recognizes.
func (r Role) Valid() bool {
	return r == RoleAdministrator || r == RoleTechnician
}

// User is an authentication account. The plain password never reaches this
// struct: only the hash produced by the authentication use case does.
type User struct {
	ID                     string
	Username               string
	PasswordHash           string
	Role                   Role
	FullName               string
	IsActive               bool
	RequiresPasswordChange bool
	CreatedAt              time.Time
}

// NewUser builds a user after validating the rules that must always hold. Defaults IsActive to true and RequiresPasswordChange to false.
func NewUser(id, username, passwordHash, fullName string, role Role, createdAt time.Time) (User, error) {
	return NewUserWithStatus(id, username, passwordHash, fullName, role, true, createdAt)
}

// NewUserWithStatus builds a user with an explicit active status after validating invariants.
func NewUserWithStatus(id, username, passwordHash, fullName string, role Role, isActive bool, createdAt time.Time) (User, error) {
	return NewUserWithSecurity(id, username, passwordHash, fullName, role, isActive, false, createdAt)
}

// NewUserWithSecurity builds a user with explicit active and password-change flags.
func NewUserWithSecurity(id, username, passwordHash, fullName string, role Role, isActive, requiresPasswordChange bool, createdAt time.Time) (User, error) {
	username = strings.TrimSpace(username)
	fullName = strings.TrimSpace(fullName)
	if id == "" {
		return User{}, fmt.Errorf("%w: user identifier is required", ErrInvalidInput)
	}
	if username == "" {
		return User{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if passwordHash == "" {
		return User{}, fmt.Errorf("%w: password hash is required", ErrInvalidInput)
	}
	if fullName == "" {
		return User{}, fmt.Errorf("%w: full name is required", ErrInvalidInput)
	}
	if !role.Valid() {
		return User{}, fmt.Errorf("%w: unknown role %q", ErrInvalidInput, role)
	}
	return User{
		ID:                     id,
		Username:               username,
		PasswordHash:           passwordHash,
		Role:                   role,
		FullName:               fullName,
		IsActive:               isActive,
		RequiresPasswordChange: requiresPasswordChange,
		CreatedAt:              createdAt,
	}, nil
}

// Deactivate revokes access for the user.
func (u *User) Deactivate() {
	u.IsActive = false
}

// Activate restores access for the user.
func (u *User) Activate() {
	u.IsActive = true
}
