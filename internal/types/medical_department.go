package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MedicalDepartment represents a hospital department in the healthcare add-on.
// Departments are tenant-scoped and independent of WeKnora knowledge bases.
type MedicalDepartment struct {
	// Unique identifier (UUID)
	ID string `json:"id"             gorm:"type:varchar(36);primaryKey"`
	// Tenant ID for data isolation
	TenantID uint64 `json:"tenant_id"      gorm:"type:bigint;not null;index:idx_medical_departments_tenant_id"`
	// Department name (required)
	Name string `json:"name"           gorm:"type:varchar(128);not null"`
	// Department code, unique per tenant (required, manual input)
	Code string `json:"code"           gorm:"type:varchar(64);not null"`
	// Hospital campus/area (required)
	HospitalArea string `json:"hospital_area"  gorm:"type:varchar(128);not null"`
	// Enable/disable status
	Enabled bool `json:"enabled"        gorm:"default:true"`
	// User ID who created this record
	CreatedBy string `json:"created_by"     gorm:"type:varchar(36)"`
	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`
	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete timestamp
	DeletedAt gorm.DeletedAt `json:"deleted_at"     gorm:"index:idx_medical_departments_deleted_at"`
}

// BeforeCreate populates the UUID primary key on insert.
func (d *MedicalDepartment) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// CreateMedicalDepartmentRequest is the request body for creating a department.
type CreateMedicalDepartmentRequest struct {
	Name         string `json:"name"          binding:"required"`
	Code         string `json:"code"          binding:"required"`
	HospitalArea string `json:"hospital_area" binding:"required"`
	Enabled      *bool  `json:"enabled"`
}

// UpdateMedicalDepartmentRequest is the request body for updating a department.
type UpdateMedicalDepartmentRequest struct {
	Name         *string `json:"name"`
	Code         *string `json:"code"`
	HospitalArea *string `json:"hospital_area"`
	Enabled      *bool   `json:"enabled"`
}

// ListMedicalDepartmentRequest is the query parameters for listing departments.
type ListMedicalDepartmentRequest struct {
	Keyword string `form:"keyword"`
	Enabled *bool  `form:"enabled"`
	Page    int    `form:"page"`
	PageSize int   `form:"page_size"`
}

// MedicalDepartmentResponse is the API response for a single department.
type MedicalDepartmentResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	HospitalArea string     `json:"hospital_area"`
	Enabled      bool       `json:"enabled"`
	CreatedBy    string     `json:"created_by"`
	CreatedByName string    `json:"created_by_name"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// MedicalDepartmentListResponse is the paginated list response.
type MedicalDepartmentListResponse struct {
	List     []*MedicalDepartmentResponse `json:"list"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}
