package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// MedicalDepartmentRepository defines persistence operations for medical departments.
type MedicalDepartmentRepository interface {
	// Create inserts a new medical department.
	Create(ctx context.Context, dept *types.MedicalDepartment) error
	// GetByID retrieves a department by its ID (tenant-scoped).
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.MedicalDepartment, error)
	// List retrieves departments with optional keyword and enabled filters (tenant-scoped).
	List(ctx context.Context, tenantID uint64, keyword string, enabled *bool, page, pageSize int) ([]*types.MedicalDepartment, int64, error)
	// Update saves changes to an existing department.
	Update(ctx context.Context, dept *types.MedicalDepartment) error
	// Delete soft-deletes a department (tenant-scoped).
	Delete(ctx context.Context, tenantID uint64, id string) error
	// ExistsByCode checks whether a department with the given code already exists
	// for the tenant (excluding soft-deleted records and optionally the given excludeID).
	ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID string) (bool, error)
}

// MedicalDepartmentService defines business logic for medical departments.
type MedicalDepartmentService interface {
	// CreateDepartment creates a new department.
	CreateDepartment(ctx context.Context, req *types.CreateMedicalDepartmentRequest, userID string) (*types.MedicalDepartmentResponse, error)
	// GetDepartment retrieves a single department by ID.
	GetDepartment(ctx context.Context, id string) (*types.MedicalDepartmentResponse, error)
	// ListDepartments lists departments with filters and pagination.
	ListDepartments(ctx context.Context, req *types.ListMedicalDepartmentRequest) (*types.MedicalDepartmentListResponse, error)
	// UpdateDepartment updates an existing department.
	UpdateDepartment(ctx context.Context, id string, req *types.UpdateMedicalDepartmentRequest) (*types.MedicalDepartmentResponse, error)
	// DeleteDepartment soft-deletes a department.
	DeleteDepartment(ctx context.Context, id string) error
	// ListHospitalAreas returns the configured hospital area options.
	ListHospitalAreas(ctx context.Context) []HospitalAreaOption
}

// HospitalAreaOption represents a selectable hospital area.
type HospitalAreaOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
