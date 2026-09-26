package domain

import (
	"fmt"
	"unicode"
)

// OrderPermissions describes the allowed operations for a specific user on a service order.
type OrderPermissions struct {
	CanAdvance         bool `json:"canAdvance"`
	CanAddDiagnostic   bool `json:"canAddDiagnostic"`
	CanAddIntervention bool `json:"canAddIntervention"`
	CanAssign          bool `json:"canAssign"`
}

// CalculateOrderPermissions is a pure function evaluating authorization and lifecycle
// permissions based on the actor identity (role, technician ID), active order assignment,
// and current order status.
func CalculateOrderPermissions(
	actorRole Role,
	actorTechnicianID string,
	order ServiceOrder,
	activeTechnicianID string,
) OrderPermissions {
	if !order.Status.IsOpen() || order.Status == StatusDelivered {
		return OrderPermissions{}
	}

	nextStatus, hasNext := order.Status.Next()
	canAdvance := hasNext && order.Status.CanMoveTo(nextStatus)
	// Invariant: cannot advance to an operational status (IN_DIAGNOSIS or IN_REPAIR) without an assigned technician.
	if (nextStatus == StatusInDiagnosis || nextStatus == StatusInRepair) && activeTechnicianID == "" {
		canAdvance = false
	}

	switch actorRole {
	case RoleAdministrator:
		return OrderPermissions{
			CanAdvance:         canAdvance,
			CanAddDiagnostic:   false,
			CanAddIntervention: false,
			CanAssign:          order.Status.CanAssignTechnician(),
		}
	case RoleTechnician:
		// Strict isolation: if order is unassigned, actor is empty, or actor is not the assigned technician
		if actorTechnicianID == "" || activeTechnicianID == "" || actorTechnicianID != activeTechnicianID {
			return OrderPermissions{}
		}
		return OrderPermissions{
			CanAdvance:         canAdvance,
			CanAddDiagnostic:   order.Status.CanAddDiagnostic(),
			CanAddIntervention: order.Status.CanAddIntervention(),
			CanAssign:          false,
		}
	default:
		return OrderPermissions{}
	}
}

// ValidatePasswordComplexity enforces that passwords must have at least 8 characters,
// with at least one uppercase letter, one lowercase letter, one digit, and one special character.
func ValidatePasswordComplexity(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters long", ErrWeakPassword)
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("%w: password must contain at least one uppercase letter", ErrWeakPassword)
	}
	if !hasLower {
		return fmt.Errorf("%w: password must contain at least one lowercase letter", ErrWeakPassword)
	}
	if !hasDigit {
		return fmt.Errorf("%w: password must contain at least one number", ErrWeakPassword)
	}
	if !hasSpecial {
		return fmt.Errorf("%w: password must contain at least one special character", ErrWeakPassword)
	}

	return nil
}
