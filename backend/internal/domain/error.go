package domain

import (
	"errors"
	"fmt"
)

// Domain errors. The transport layer maps each one to a status code; no layer
// below transport knows anything about HTTP.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")
	// ErrConflict is returned when a uniqueness rule rejects the operation.
	ErrConflict = errors.New("resource conflict")
	// ErrInvalidInput is returned when the payload violates a domain rule.
	ErrInvalidInput = errors.New("invalid input")
	// ErrUnauthorized is returned when credentials or the session token fail.
	ErrUnauthorized = errors.New("invalid credentials")
	// ErrInvalidCredentials is an alias to ErrUnauthorized.
	ErrInvalidCredentials = ErrUnauthorized
	// ErrForbidden is returned when the caller may not act on the resource.
	ErrForbidden = errors.New("operation not allowed for this user")
	// ErrInvalidTransition is returned when a status move leaves the lifecycle.
	ErrInvalidTransition = errors.New("service order status transition not allowed")
	// ErrTechnicianRequired is returned when advancing to an operational status without an active technician.
	ErrTechnicianRequired = errors.New("technician assignment required before advancing to operational status")
	// ErrAccountInactive is returned when an inactive user attempts to authenticate or execute an action.
	ErrAccountInactive = errors.New("user account is inactive")
	// ErrAuthorizationStateUnavailable is returned when the database cannot be queried during authorization (fail-closed).
	ErrAuthorizationStateUnavailable = errors.New("authorization state is unavailable")
	// ErrSelfDeactivation is returned when an administrator attempts to revoke access to their own account.
	ErrSelfDeactivation = errors.New("self deactivation is forbidden")
	// ErrLastAdministrator is returned when deactivation would leave the system with zero active administrators.
	ErrLastAdministrator = errors.New("cannot deactivate the last active administrator")
	// ErrTechnicianInactive is returned when attempting to assign a service order to an inactive technician.
	ErrTechnicianInactive = errors.New("technician is inactive")
	// ErrWeakPassword is returned when a password does not satisfy security complexity requirements.
	ErrWeakPassword = fmt.Errorf("%w: password does not meet complexity requirements", ErrInvalidInput)
)
