package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// medicalDepartmentRepository implements MedicalDepartmentRepository using GORM.
type medicalDepartmentRepository struct {
	db *gorm.DB
}

// NewMedicalDepartmentRepository creates a new medical department repository.
func NewMedicalDepartmentRepository(db *gorm.DB) interfaces.MedicalDepartmentRepository {
	return &medicalDepartmentRepository{db: db}
}

// Create inserts a new medical department record.
func (r *medicalDepartmentRepository) Create(ctx context.Context, dept *types.MedicalDepartment) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

// GetByID retrieves a department by its ID (tenant-scoped, excluding soft-deleted).
func (r *medicalDepartmentRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.MedicalDepartment, error) {
	var dept types.MedicalDepartment
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&dept).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

// List retrieves departments with optional keyword (matches name or code) and enabled filter.
func (r *medicalDepartmentRepository) List(
	ctx context.Context,
	tenantID uint64,
	keyword string,
	enabled *bool,
	page, pageSize int,
) ([]*types.MedicalDepartment, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&types.MedicalDepartment{}).
		Where("tenant_id = ?", tenantID)

	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if enabled != nil {
		baseQuery = baseQuery.Where("enabled = ?", *enabled)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var depts []*types.MedicalDepartment
	if err := baseQuery.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&depts).Error; err != nil {
		return nil, 0, err
	}

	return depts, total, nil
}

// Update saves changes to an existing department.
func (r *medicalDepartmentRepository) Update(ctx context.Context, dept *types.MedicalDepartment) error {
	return r.db.WithContext(ctx).Save(dept).Error
}

// Delete soft-deletes a department (tenant-scoped).
// Since MedicalDepartment has gorm.DeletedAt, GORM sets deleted_at instead of
// actually removing the row.
func (r *medicalDepartmentRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&types.MedicalDepartment{}).Error
}

// ExistsByCode checks whether a department with the given code already exists
// for the tenant (excluding soft-deleted records). If excludeID is non-empty,
// that record is excluded from the check (used during update to allow keeping
// the same code).
func (r *medicalDepartmentRepository) ExistsByCode(
	ctx context.Context,
	tenantID uint64,
	code string,
	excludeID string,
) (bool, error) {
	query := r.db.WithContext(ctx).Model(&types.MedicalDepartment{}).
		Where("tenant_id = ? AND code = ?", tenantID, code)

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
