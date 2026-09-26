-- Consolidated MySQL initialization script
-- Automatically executed on container first boot or cloud database setup.

CREATE DATABASE IF NOT EXISTS workshop;
USE workshop;

-- 0001_create_user
CREATE TABLE IF NOT EXISTS `user` (
  id CHAR(36) NOT NULL,
  username VARCHAR(80) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(40) NOT NULL,
  full_name VARCHAR(160) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  requires_password_change TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_user PRIMARY KEY (id),
  CONSTRAINT uq_user_username UNIQUE (username),
  CONSTRAINT ck_user_role CHECK (role IN ('ADMINISTRATOR', 'TECHNICIAN'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0002_create_customer
CREATE TABLE IF NOT EXISTS customer (
  id CHAR(36) NOT NULL,
  full_name VARCHAR(160) NOT NULL,
  document_number VARCHAR(40) NOT NULL,
  phone VARCHAR(40) NOT NULL,
  email VARCHAR(160) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_customer PRIMARY KEY (id),
  CONSTRAINT uq_customer_document_number UNIQUE (document_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0003_create_vehicle
CREATE TABLE IF NOT EXISTS vehicle (
  id CHAR(36) NOT NULL,
  customer_id CHAR(36) NOT NULL,
  plate VARCHAR(16) NOT NULL,
  vin VARCHAR(32) NOT NULL,
  brand VARCHAR(80) NOT NULL,
  model VARCHAR(80) NOT NULL,
  model_year SMALLINT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_vehicle PRIMARY KEY (id),
  CONSTRAINT uq_vehicle_plate UNIQUE (plate),
  CONSTRAINT uq_vehicle_vin UNIQUE (vin),
  CONSTRAINT ck_vehicle_model_year CHECK (model_year > 1900),
  CONSTRAINT fk_vehicle_customer FOREIGN KEY (customer_id)
    REFERENCES customer (id) ON DELETE RESTRICT,
  INDEX ix_vehicle_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0004_create_technician
CREATE TABLE IF NOT EXISTS technician (
  id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  specialty VARCHAR(120) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_technician PRIMARY KEY (id),
  CONSTRAINT uq_technician_user_id UNIQUE (user_id),
  CONSTRAINT fk_technician_user FOREIGN KEY (user_id)
    REFERENCES `user` (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0005_create_service_order
CREATE TABLE IF NOT EXISTS service_order (
  id CHAR(36) NOT NULL,
  order_number VARCHAR(32) NOT NULL,
  vehicle_id CHAR(36) NOT NULL,
  reported_failure TEXT NOT NULL,
  status VARCHAR(40) NOT NULL,
  received_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT pk_service_order PRIMARY KEY (id),
  CONSTRAINT uq_service_order_order_number UNIQUE (order_number),
  CONSTRAINT ck_service_order_status CHECK (
    status IN ('RECEIVED', 'IN_DIAGNOSIS', 'IN_REPAIR', 'READY', 'DELIVERED')
  ),
  CONSTRAINT fk_service_order_vehicle FOREIGN KEY (vehicle_id)
    REFERENCES vehicle (id) ON DELETE RESTRICT,
  INDEX ix_service_order_vehicle_id (vehicle_id),
  INDEX ix_service_order_status (status),
  INDEX ix_service_order_received_at (received_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0006_create_assignment
CREATE TABLE IF NOT EXISTS assignment (
  id CHAR(36) NOT NULL,
  service_order_id CHAR(36) NOT NULL,
  technician_id CHAR(36) NOT NULL,
  is_active TINYINT(1) NOT NULL,
  active_marker CHAR(36) NULL,
  active_order_marker CHAR(36) NULL,
  assigned_at DATETIME NOT NULL,
  released_at DATETIME NULL,
  CONSTRAINT pk_assignment PRIMARY KEY (id),
  CONSTRAINT uq_assignment_active_marker UNIQUE (active_marker),
  CONSTRAINT uq_assignment_active_order_marker UNIQUE (active_order_marker),
  CONSTRAINT ck_assignment_active_marker CHECK (
    (is_active = 1 AND active_marker IS NOT NULL AND active_marker = technician_id)
    OR (is_active = 0 AND active_marker IS NULL)
  ),
  CONSTRAINT ck_assignment_active_order_marker CHECK (
    (is_active = 1 AND active_order_marker IS NOT NULL AND active_order_marker = service_order_id)
    OR (is_active = 0 AND active_order_marker IS NULL)
  ),
  CONSTRAINT fk_assignment_service_order FOREIGN KEY (service_order_id)
    REFERENCES service_order (id) ON DELETE CASCADE,
  CONSTRAINT fk_assignment_technician FOREIGN KEY (technician_id)
    REFERENCES technician (id) ON DELETE RESTRICT,
  INDEX ix_assignment_service_order_id (service_order_id),
  INDEX ix_assignment_technician_id (technician_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0007_create_diagnostic
CREATE TABLE IF NOT EXISTS diagnostic (
  id CHAR(36) NOT NULL,
  service_order_id CHAR(36) NOT NULL,
  technician_id CHAR(36) NOT NULL,
  finding TEXT NOT NULL,
  component_to_repair TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_diagnostic PRIMARY KEY (id),
  CONSTRAINT fk_diagnostic_service_order FOREIGN KEY (service_order_id)
    REFERENCES service_order (id) ON DELETE CASCADE,
  CONSTRAINT fk_diagnostic_technician FOREIGN KEY (technician_id)
    REFERENCES technician (id) ON DELETE RESTRICT,
  INDEX ix_diagnostic_service_order_id (service_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0008_create_intervention
CREATE TABLE IF NOT EXISTS intervention (
  id CHAR(36) NOT NULL,
  service_order_id CHAR(36) NOT NULL,
  technician_id CHAR(36) NOT NULL,
  description TEXT NOT NULL,
  labor_hour_count DECIMAL(6,2) NOT NULL,
  performed_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_intervention PRIMARY KEY (id),
  CONSTRAINT ck_intervention_labor_hour_count CHECK (labor_hour_count > 0),
  CONSTRAINT fk_intervention_service_order FOREIGN KEY (service_order_id)
    REFERENCES service_order (id) ON DELETE CASCADE,
  CONSTRAINT fk_intervention_technician FOREIGN KEY (technician_id)
    REFERENCES technician (id) ON DELETE RESTRICT,
  INDEX ix_intervention_service_order_id (service_order_id),
  INDEX ix_intervention_performed_at (performed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0009_create_part_usage
CREATE TABLE IF NOT EXISTS part_usage (
  id CHAR(36) NOT NULL,
  intervention_id CHAR(36) NOT NULL,
  part_name VARCHAR(160) NOT NULL,
  quantity INT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_part_usage PRIMARY KEY (id),
  CONSTRAINT ck_part_usage_quantity CHECK (quantity > 0),
  CONSTRAINT fk_part_usage_intervention FOREIGN KEY (intervention_id)
    REFERENCES intervention (id) ON DELETE CASCADE,
  INDEX ix_part_usage_intervention_id (intervention_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0010_create_warranty
CREATE TABLE IF NOT EXISTS warranty (
  id CHAR(36) NOT NULL,
  intervention_id CHAR(36) NOT NULL,
  warranty_kind VARCHAR(20) NOT NULL,
  coverage_month_count INT NOT NULL,
  issued_at DATETIME NOT NULL,
  expiration_date DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_warranty PRIMARY KEY (id),
  CONSTRAINT ck_warranty_kind CHECK (warranty_kind IN ('LABOR', 'PART')),
  CONSTRAINT ck_warranty_coverage_month_count CHECK (coverage_month_count > 0),
  CONSTRAINT fk_warranty_intervention FOREIGN KEY (intervention_id)
    REFERENCES intervention (id) ON DELETE RESTRICT,
  INDEX ix_warranty_intervention_id (intervention_id),
  INDEX ix_warranty_expiration_date (expiration_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 0011_create_status_transition
CREATE TABLE IF NOT EXISTS status_transition (
  id CHAR(36) NOT NULL,
  service_order_id CHAR(36) NOT NULL,
  from_status VARCHAR(40) NOT NULL,
  to_status VARCHAR(40) NOT NULL,
  changed_by_user_id CHAR(36) NOT NULL,
  changed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT pk_status_transition PRIMARY KEY (id),
  CONSTRAINT fk_status_transition_service_order FOREIGN KEY (service_order_id)
    REFERENCES service_order (id) ON DELETE CASCADE,
  CONSTRAINT fk_status_transition_user FOREIGN KEY (changed_by_user_id)
    REFERENCES `user` (id) ON DELETE RESTRICT,
  INDEX ix_status_transition_service_order_id (service_order_id, changed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Seed bootstrap (Initial users: admin, jperez, lramirez with exact credentials)
SET @admin_username = 'admin';
SET @admin_full_name = 'Administrador del taller';
SET @admin_password_hash = '$2a$10$w1Ogd7P9BQMWaBZzPpU8tuptXLL0LlxCSRQLZ3b9UKimMklLiJ5Um';
SET @technician_one_username = 'jperez';
SET @technician_one_full_name = 'Juan Perez';
SET @technician_one_password_hash = '$2a$10$pGTrsi0lGtajtxLEmkSmeuyvlJomL1KTVawt0Wi/eDF8i2sgFD.Zm';
SET @technician_one_specialty = 'Motor y transmision';
SET @technician_two_username = 'lramirez';
SET @technician_two_full_name = 'Laura Ramirez';
SET @technician_two_password_hash = '$2a$10$Aks9UH5ENBk/DwyN3aUu4OX4WJO91umqXuKPSescDv/lZuEI8BcIG';
SET @technician_two_specialty = 'Frenos y suspension';

INSERT INTO `user` (id, username, password_hash, role, full_name, created_at)
VALUES (
  '11111111-1111-4111-8111-111111111111',
  @admin_username,
  @admin_password_hash,
  'ADMINISTRATOR',
  @admin_full_name,
  CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE
  password_hash = @admin_password_hash,
  full_name = @admin_full_name;

INSERT INTO `user` (id, username, password_hash, role, full_name, created_at)
VALUES (
  '22222222-2222-4222-8222-222222222222',
  @technician_one_username,
  @technician_one_password_hash,
  'TECHNICIAN',
  @technician_one_full_name,
  CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE
  password_hash = @technician_one_password_hash,
  full_name = @technician_one_full_name;

INSERT INTO `user` (id, username, password_hash, role, full_name, created_at)
VALUES (
  '33333333-3333-4333-8333-333333333333',
  @technician_two_username,
  @technician_two_password_hash,
  'TECHNICIAN',
  @technician_two_full_name,
  CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE
  password_hash = @technician_two_password_hash,
  full_name = @technician_two_full_name;

INSERT INTO technician (id, user_id, specialty, created_at)
VALUES (
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1',
  '22222222-2222-4222-8222-222222222222',
  @technician_one_specialty,
  CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE specialty = @technician_one_specialty;

INSERT INTO technician (id, user_id, specialty, created_at)
VALUES (
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2',
  '33333333-3333-4333-8333-333333333333',
  @technician_two_specialty,
  CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE specialty = @technician_two_specialty;
