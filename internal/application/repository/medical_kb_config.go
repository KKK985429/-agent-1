package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type medicalKBConfigRepo struct {
	db *gorm.DB
}

func NewMedicalKnowledgeBaseConfigRepository(db *gorm.DB) interfaces.MedicalKnowledgeBaseConfigRepository {
	return &medicalKBConfigRepo{db: db}
}

// GetByTenantAndCategory retrieves one config row; returns nil, nil if not found.
func (r *medicalKBConfigRepo) GetByTenantAndCategory(
	ctx context.Context, tenantID uint64, category string,
) (*types.MedicalKnowledgeBaseConfig, error) {
	var cfg types.MedicalKnowledgeBaseConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND category = ?", tenantID, category).
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

// ListByTenant retrieves all config rows for a tenant.
func (r *medicalKBConfigRepo) ListByTenant(
	ctx context.Context, tenantID uint64,
) ([]*types.MedicalKnowledgeBaseConfig, error) {
	var configs []*types.MedicalKnowledgeBaseConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&configs).Error
	return configs, err
}

// Upsert inserts or updates on conflict (tenant_id, category).
func (r *medicalKBConfigRepo) Upsert(
	ctx context.Context, config *types.MedicalKnowledgeBaseConfig,
) error {
	if config.ID == "" {
		// Let BeforeCreate generate UUID.
		// But GORM Upsert with OnConflict needs a primary key value.
		// We use a workaround: find existing by tenant+category, then create or update.
		existing, err := r.GetByTenantAndCategory(ctx, config.TenantID, config.Category)
		if err != nil {
			return err
		}
		if existing != nil {
			config.ID = existing.ID
			config.CreatedAt = existing.CreatedAt
		}
	}
	config.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "category"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"display_name", "document_kb_id", "faq_kb_id", "updated_at",
			}),
		}).
		Create(config).Error
}
