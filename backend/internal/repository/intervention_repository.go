package repository

import (
	"context"
	"database/sql"
	"time"

	"workshop/internal/domain"
)

const interventionColumn = "id, service_order_id, technician_id, description, labor_hour_count, performed_at, created_at"

// InterventionRepository persists the work executed on a vehicle together with
// the parts it consumed.
type InterventionRepository struct {
	database *sql.DB
	timeout  time.Duration
}

// NewInterventionRepository wires the intervention adapter.
func NewInterventionRepository(database *sql.DB, timeout time.Duration) InterventionRepository {
	return InterventionRepository{database: database, timeout: timeout}
}

// Save writes the intervention and its parts in one transaction, so an
// intervention is never stored without the parts it declared.
func (r InterventionRepository) Save(ctx context.Context, intervention domain.Intervention) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	transaction, err := r.database.BeginTx(queryCtx, nil)
	if err != nil {
		return translate(err)
	}
	defer func() { _ = transaction.Rollback() }()

	if _, err := transaction.ExecContext(
		queryCtx,
		"INSERT INTO intervention ("+interventionColumn+") VALUES (?, ?, ?, ?, ?, ?, ?)",
		intervention.ID, intervention.ServiceOrderID, intervention.TechnicianID,
		intervention.Description, intervention.LaborHourCount,
		intervention.PerformedAt, intervention.CreatedAt,
	); err != nil {
		return translate(err)
	}
	for _, part := range intervention.Part {
		if _, err := transaction.ExecContext(
			queryCtx,
			"INSERT INTO part_usage (id, intervention_id, part_name, quantity, created_at) VALUES (?, ?, ?, ?, ?)",
			part.ID, intervention.ID, part.PartName, part.Quantity, part.CreatedAt,
		); err != nil {
			return translate(err)
		}
	}
	return translate(transaction.Commit())
}

// FindByID reads one intervention with its parts.
func (r InterventionRepository) FindByID(ctx context.Context, id string) (domain.Intervention, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var intervention domain.Intervention
	err := r.database.QueryRowContext(
		queryCtx, "SELECT "+interventionColumn+" FROM intervention WHERE id = ?", id,
	).Scan(
		&intervention.ID, &intervention.ServiceOrderID, &intervention.TechnicianID,
		&intervention.Description, &intervention.LaborHourCount,
		&intervention.PerformedAt, &intervention.CreatedAt,
	)
	if err != nil {
		return domain.Intervention{}, translate(err)
	}
	part, err := r.listPart(queryCtx, intervention.ID)
	if err != nil {
		return domain.Intervention{}, err
	}
	intervention.Part = part
	return intervention, nil
}

// ListByServiceOrder reads the interventions of an order, oldest first, with their warranty coverage if issued.
func (r InterventionRepository) ListByServiceOrder(ctx context.Context, serviceOrderID string) ([]domain.Intervention, error) {
	return r.list(
		ctx,
		"SELECT i.id, i.service_order_id, i.technician_id, i.description, i.labor_hour_count, "+
			"i.performed_at, i.created_at, w.id, w.warranty_kind, w.coverage_month_count, w.expiration_date "+
			"FROM intervention i "+
			"LEFT JOIN warranty w ON w.intervention_id = i.id "+
			"WHERE i.service_order_id = ? ORDER BY i.performed_at",
		serviceOrderID,
	)
}

// ListByVehicle reads every intervention of a vehicle for the timeline.
func (r InterventionRepository) ListByVehicle(ctx context.Context, vehicleID string) ([]domain.Intervention, error) {
	return r.list(
		ctx,
		"SELECT i.id, i.service_order_id, i.technician_id, i.description, i.labor_hour_count, "+
			"i.performed_at, i.created_at, w.id, w.warranty_kind, w.coverage_month_count, w.expiration_date "+
			"FROM intervention i "+
			"JOIN service_order so ON so.id = i.service_order_id "+
			"LEFT JOIN warranty w ON w.intervention_id = i.id "+
			"WHERE so.vehicle_id = ? ORDER BY i.performed_at",
		vehicleID,
	)
}

func (r InterventionRepository) list(ctx context.Context, query, argument string) ([]domain.Intervention, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.database.QueryContext(queryCtx, query, argument)
	if err != nil {
		return nil, translate(err)
	}
	defer func() { _ = rows.Close() }()

	listed := make([]domain.Intervention, 0)
	now := time.Now()
	for rows.Next() {
		var intervention domain.Intervention
		var warrantyID, warrantyKind sql.NullString
		var coverageMonths sql.NullInt32
		var expirationDate sql.NullTime
		if err := rows.Scan(
			&intervention.ID, &intervention.ServiceOrderID, &intervention.TechnicianID,
			&intervention.Description, &intervention.LaborHourCount,
			&intervention.PerformedAt, &intervention.CreatedAt,
			&warrantyID, &warrantyKind, &coverageMonths, &expirationDate,
		); err != nil {
			return nil, translate(err)
		}
		if warrantyID.Valid && warrantyID.String != "" {
			valid := expirationDate.Valid && !expirationDate.Time.Before(now)
			intervention.Warranty = &domain.InterventionWarranty{
				ID:                 warrantyID.String,
				Valid:              valid,
				Kind:               warrantyKind.String,
				CoverageMonthCount: int(coverageMonths.Int32),
				ExpirationDate:     expirationDate.Time,
			}
		}
		listed = append(listed, intervention)
	}
	if err := rows.Err(); err != nil {
		return nil, translate(err)
	}
	for index := range listed {
		part, err := r.listPart(queryCtx, listed[index].ID)
		if err != nil {
			return nil, err
		}
		listed[index].Part = part
	}
	return listed, nil
}

func (r InterventionRepository) listPart(ctx context.Context, interventionID string) ([]domain.PartUsage, error) {
	rows, err := r.database.QueryContext(
		ctx,
		"SELECT id, intervention_id, part_name, quantity, created_at FROM part_usage "+
			"WHERE intervention_id = ? ORDER BY created_at",
		interventionID,
	)
	if err != nil {
		return nil, translate(err)
	}
	defer func() { _ = rows.Close() }()

	listed := make([]domain.PartUsage, 0)
	for rows.Next() {
		var part domain.PartUsage
		if err := rows.Scan(
			&part.ID, &part.InterventionID, &part.PartName, &part.Quantity, &part.CreatedAt,
		); err != nil {
			return nil, translate(err)
		}
		listed = append(listed, part)
	}
	return listed, translate(rows.Err())
}
