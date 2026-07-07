-- Migration 000060: Create medical_departments table
--
-- This table stores medical department information for the healthcare
-- knowledge base add-on. Departments are tenant-scoped and have no
-- relationship with WeKnora knowledge bases or their content.
--
-- Fields:
--   id            - UUID primary key
--   tenant_id     - Tenant isolation (matches WeKnora tenant model)
--   name          - Department name (required)
--   code          - Department code, unique per tenant (required, manual input)
--   hospital_area - Hospital campus/area (required)
--   enabled       - Enable/disable status (default true)
--   created_by    - User ID who created this record
--   created_at    - Creation timestamp
--   updated_at    - Last update timestamp
--   deleted_at    - Soft delete timestamp (NULL = active)

CREATE TABLE IF NOT EXISTS medical_departments (
    id              VARCHAR(36)  PRIMARY KEY,
    tenant_id       BIGINT       NOT NULL,
    name            VARCHAR(128) NOT NULL,
    code            VARCHAR(64)  NOT NULL,
    hospital_area   VARCHAR(128) NOT NULL,
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by      VARCHAR(36),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP    NULL
);

-- Indexes
CREATE INDEX idx_medical_departments_tenant_id  ON medical_departments(tenant_id);
CREATE INDEX idx_medical_departments_enabled    ON medical_departments(enabled);
CREATE INDEX idx_medical_departments_deleted_at ON medical_departments(deleted_at);

-- Unique constraint: one active department per (tenant, code).
-- Soft-deleted records do not conflict, allowing re-creation with the same code.
CREATE UNIQUE INDEX idx_medical_departments_tenant_code_unique
    ON medical_departments(tenant_id, code)
    WHERE deleted_at IS NULL;
