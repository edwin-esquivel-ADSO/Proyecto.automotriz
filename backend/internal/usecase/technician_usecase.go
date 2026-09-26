package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workshop/internal/domain"
)

// TechnicianReader is the narrow port used to inspect technicians and their availability.
type TechnicianReader interface {
	ListWorkload(ctx context.Context) ([]domain.TechnicianWorkload, error)
	FindByID(ctx context.Context, id string) (domain.Technician, error)
	FindByUserID(ctx context.Context, userID string) (domain.Technician, error)
}

// TechnicianWriter manages technician account creation and access changes.
type TechnicianWriter interface {
	CreateWithAccount(ctx context.Context, user domain.User, tech domain.Technician) error
	UpdateAccessWithLock(ctx context.Context, actorUserID, technicianID string, active bool) (domain.TechnicianWorkload, error)
}

// TechnicianRepository is the combined port that TechnicianUseCase needs.
type TechnicianRepository interface {
	TechnicianReader
	TechnicianWriter
}

// TechnicianUseCase reports who is available to receive a service order and manages employee provisioning & access.
type TechnicianUseCase struct {
	technician TechnicianRepository
	newID      func() string
	now        func() time.Time
}

// NewTechnicianUseCase wires the technician use case.
func NewTechnicianUseCase(technician TechnicianRepository, newID func() string, now func() time.Time) TechnicianUseCase {
	return TechnicianUseCase{technician: technician, newID: newID, now: now}
}

// ListWorkload returns every technician with the order they currently hold.
func (t TechnicianUseCase) ListWorkload(ctx context.Context) ([]domain.TechnicianWorkload, error) {
	return t.technician.ListWorkload(ctx)
}

// FindByUserID resolves the mechanic profile of a signed in user.
func (t TechnicianUseCase) FindByUserID(ctx context.Context, userID string) (domain.Technician, error) {
	return t.technician.FindByUserID(ctx, userID)
}

// Create provisions a new technician employee account atomically.
// Notice: bcrypt hashing occurs OUTSIDE the SQL transaction, preventing long lock holds.
func (t TechnicianUseCase) Create(ctx context.Context, fullName, username, password, specialty string) (domain.Technician, error) {
	fullName = strings.TrimSpace(fullName)
	username = strings.TrimSpace(username)
	specialty = strings.TrimSpace(specialty)

	if len(fullName) < 3 {
		return domain.Technician{}, fmt.Errorf("%w: el nombre completo debe tener al menos 3 caracteres", domain.ErrInvalidInput)
	}
	if len(username) < 3 {
		return domain.Technician{}, fmt.Errorf("%w: el nombre de usuario debe tener al menos 3 caracteres", domain.ErrInvalidInput)
	}
	if err := domain.ValidatePasswordComplexity(password); err != nil {
		return domain.Technician{}, err
	}
	if len(specialty) < 3 {
		return domain.Technician{}, fmt.Errorf("%w: la especialidad debe tener al menos 3 caracteres", domain.ErrInvalidInput)
	}

	// 1. Password hashing with bcrypt OUTSIDE the database transaction
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.Technician{}, fmt.Errorf("error generando hash seguro: %w", err)
	}

	createdAt := t.now()
	userID := t.newID()
	techID := t.newID()

	user, err := domain.NewUserWithSecurity(userID, username, string(hash), fullName, domain.RoleTechnician, true, true, createdAt)
	if err != nil {
		return domain.Technician{}, err
	}

	tech, err := domain.NewTechnician(techID, userID, specialty, createdAt)
	if err != nil {
		return domain.Technician{}, err
	}

	// 2. Persist atomically with all-or-nothing rollback
	if err := t.technician.CreateWithAccount(ctx, user, tech); err != nil {
		return domain.Technician{}, err
	}

	return tech, nil
}

// SetAccess modifies the access state of a technician idempotently under canonical lock order.
func (t TechnicianUseCase) SetAccess(ctx context.Context, actorUserID, technicianID string, active bool) (domain.TechnicianWorkload, error) {
	return t.technician.UpdateAccessWithLock(ctx, actorUserID, technicianID, active)
}
