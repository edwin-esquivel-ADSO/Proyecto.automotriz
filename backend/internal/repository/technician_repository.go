package repository

import (
	"context"
	"database/sql"
	"time"

	"workshop/internal/domain"
)

const technicianColumn = "id, user_id, specialty, created_at"

// workloadSelect reads every technician with the order they currently hold and their account active status.
const workloadSelect = "SELECT t.id, t.user_id, t.specialty, t.created_at, u.full_name, u.is_active, " +
	"COALESCE(a.service_order_id, ''), COALESCE(so.order_number, ''), COALESCE(v.plate, ''), " +
	"CASE WHEN a.id IS NOT NULL AND so.status IS NOT NULL AND so.status != 'DELIVERED' THEN 1 ELSE 0 END AS is_busy " +
	"FROM technician t " +
	"JOIN `user` u ON u.id = t.user_id " +
	"LEFT JOIN assignment a ON a.technician_id = t.id AND a.is_active = 1 " +
	"LEFT JOIN service_order so ON so.id = a.service_order_id " +
	"LEFT JOIN vehicle v ON v.id = so.vehicle_id " +
	"ORDER BY u.full_name"

// TechnicianRepository reads the mechanic profiles and their workload.
type TechnicianRepository struct {
	database *sql.DB
	timeout  time.Duration
}

// NewTechnicianRepository wires the technician adapter.
func NewTechnicianRepository(database *sql.DB, timeout time.Duration) TechnicianRepository {
	return TechnicianRepository{database: database, timeout: timeout}
}

// ListWorkload returns every technician with the order they are working on.
func (r TechnicianRepository) ListWorkload(ctx context.Context) ([]domain.TechnicianWorkload, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.database.QueryContext(queryCtx, workloadSelect)
	if err != nil {
		return nil, translate(err)
	}
	defer func() { _ = rows.Close() }()

	listed := make([]domain.TechnicianWorkload, 0)
	for rows.Next() {
		var workload domain.TechnicianWorkload
		var isBusy, isActive int
		if err := rows.Scan(
			&workload.Technician.ID, &workload.Technician.UserID,
			&workload.Technician.Specialty, &workload.Technician.CreatedAt,
			&workload.FullName, &isActive, &workload.ActiveOrderID,
			&workload.ActiveOrderNumber, &workload.ActiveVehiclePlate,
			&isBusy,
		); err != nil {
			return nil, translate(err)
		}
		workload.IsActive = isActive == 1
		workload.Busy = isBusy == 1
		listed = append(listed, workload)
	}
	return listed, translate(rows.Err())
}

// CreateWithAccount persists a user authentication account and its technician profile in a single atomic transaction.
func (r TechnicianRepository) CreateWithAccount(ctx context.Context, user domain.User, tech domain.Technician) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	tx, err := r.database.BeginTx(queryCtx, nil)
	if err != nil {
		return translate(err)
	}
	defer func() { _ = tx.Rollback() }()

	isActiveVal := 0
	if user.IsActive {
		isActiveVal = 1
	}
	requiresPasswordChangeVal := 0
	if user.RequiresPasswordChange {
		requiresPasswordChangeVal = 1
	}

	_, err = tx.ExecContext(queryCtx,
		"INSERT INTO `user` (id, username, password_hash, role, full_name, is_active, requires_password_change, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		user.ID, user.Username, user.PasswordHash, string(user.Role), user.FullName, isActiveVal, requiresPasswordChangeVal, user.CreatedAt,
	)
	if err != nil {
		return translate(err)
	}

	_, err = tx.ExecContext(queryCtx,
		"INSERT INTO technician (id, user_id, specialty, created_at) VALUES (?, ?, ?, ?)",
		tech.ID, tech.UserID, tech.Specialty, tech.CreatedAt,
	)
	if err != nil {
		return translate(err)
	}

	return translate(tx.Commit())
}

// UpdateAccessWithLock acquires canonical locks in strict order (user -> technician) and idempotently mutates access.
// Enforces invariants BR-ACTIVE-07 (self-deactivation prevention) and BR-ACTIVE-07B (last active administrator protection).
func (r TechnicianRepository) UpdateAccessWithLock(ctx context.Context, actorUserID, technicianID string, active bool) (domain.TechnicianWorkload, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	tx, err := r.database.BeginTx(queryCtx, nil)
	if err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Canonical Lock Step 1: Lock user row associated with technician
	var targetUserID, role string
	var currentIsActive int
	err = tx.QueryRowContext(queryCtx,
		"SELECT u.id, u.role, u.is_active FROM `user` u JOIN technician t ON t.user_id = u.id WHERE t.id = ? FOR UPDATE",
		technicianID,
	).Scan(&targetUserID, &role, &currentIsActive)
	if err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}

	// Canonical Lock Step 2: Lock technician row
	var techID, specialty string
	var createdAt time.Time
	err = tx.QueryRowContext(queryCtx,
		"SELECT id, specialty, created_at FROM technician WHERE id = ? FOR UPDATE",
		technicianID,
	).Scan(&techID, &specialty, &createdAt)
	if err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}

	// Invariant BR-ACTIVE-07: Self-deactivation prevention
	if targetUserID == actorUserID {
		return domain.TechnicianWorkload{}, domain.ErrSelfDeactivation
	}

	// Invariant: only TECHNICIAN accounts can be managed through technician access endpoint
	if role != string(domain.RoleTechnician) {
		return domain.TechnicianWorkload{}, domain.ErrForbidden
	}

	// Invariant BR-ACTIVE-07B: If target was an administrator (general access service safeguard)
	if role == string(domain.RoleAdministrator) && !active {
		var activeAdmins int
		err = tx.QueryRowContext(queryCtx,
			"SELECT COUNT(*) FROM `user` WHERE role = 'ADMINISTRATOR' AND is_active = 1 FOR UPDATE",
		).Scan(&activeAdmins)
		if err != nil {
			return domain.TechnicianWorkload{}, translate(err)
		}
		if activeAdmins <= 1 {
			return domain.TechnicianWorkload{}, domain.ErrLastAdministrator
		}
	}

	// Idempotent state mutation
	newActiveVal := 0
	if active {
		newActiveVal = 1
	}
	_, err = tx.ExecContext(queryCtx, "UPDATE `user` SET is_active = ? WHERE id = ?", newActiveVal, targetUserID)
	if err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}

	if err := tx.Commit(); err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}

	return r.FindWorkloadByTechnicianID(ctx, technicianID)
}

// FindWorkloadByTechnicianID reads the single workload row for a given technician.
func (r TechnicianRepository) FindWorkloadByTechnicianID(ctx context.Context, technicianID string) (domain.TechnicianWorkload, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const singleWorkloadSelect = "SELECT t.id, t.user_id, t.specialty, t.created_at, u.full_name, u.is_active, " +
		"COALESCE(a.service_order_id, ''), COALESCE(so.order_number, ''), COALESCE(v.plate, ''), " +
		"CASE WHEN a.id IS NOT NULL AND so.status IS NOT NULL AND so.status != 'DELIVERED' THEN 1 ELSE 0 END AS is_busy " +
		"FROM technician t " +
		"JOIN `user` u ON u.id = t.user_id " +
		"LEFT JOIN assignment a ON a.technician_id = t.id AND a.is_active = 1 " +
		"LEFT JOIN service_order so ON so.id = a.service_order_id " +
		"LEFT JOIN vehicle v ON v.id = so.vehicle_id " +
		"WHERE t.id = ?"

	var workload domain.TechnicianWorkload
	var isBusy, isActive int
	err := r.database.QueryRowContext(queryCtx, singleWorkloadSelect, technicianID).Scan(
		&workload.Technician.ID, &workload.Technician.UserID,
		&workload.Technician.Specialty, &workload.Technician.CreatedAt,
		&workload.FullName, &isActive, &workload.ActiveOrderID,
		&workload.ActiveOrderNumber, &workload.ActiveVehiclePlate,
		&isBusy,
	)
	if err != nil {
		return domain.TechnicianWorkload{}, translate(err)
	}
	workload.IsActive = isActive == 1
	workload.Busy = isBusy == 1
	return workload, nil
}

// FindByID reads one mechanic profile.
func (r TechnicianRepository) FindByID(ctx context.Context, id string) (domain.Technician, error) {
	return r.findBy(ctx, "SELECT "+technicianColumn+" FROM technician WHERE id = ?", id)
}

// FindByUserID resolves the mechanic profile of a signed in user.
func (r TechnicianRepository) FindByUserID(ctx context.Context, userID string) (domain.Technician, error) {
	return r.findBy(ctx, "SELECT "+technicianColumn+" FROM technician WHERE user_id = ?", userID)
}

func (r TechnicianRepository) findBy(ctx context.Context, query string, argument any) (domain.Technician, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var technician domain.Technician
	err := r.database.QueryRowContext(queryCtx, query, argument).Scan(
		&technician.ID, &technician.UserID, &technician.Specialty, &technician.CreatedAt,
	)
	if err != nil {
		return domain.Technician{}, translate(err)
	}
	return technician, nil
}
