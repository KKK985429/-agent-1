package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

// MedicalKnowledgeBaseConfigRepository defines persistence operations for
// medical knowledge base configs.
type MedicalKnowledgeBaseConfigRepository interface {
	// GetByTenantAndCategory retrieves one config row; returns nil, nil if not found.
	GetByTenantAndCategory(ctx context.Context, tenantID uint64, category string) (*types.MedicalKnowledgeBaseConfig, error)
	// ListByTenant retrieves all config rows for a tenant, ordered by created_at.
	ListByTenant(ctx context.Context, tenantID uint64) ([]*types.MedicalKnowledgeBaseConfig, error)
	// Upsert inserts or updates a config row (on conflict tenant_id + category).
	Upsert(ctx context.Context, config *types.MedicalKnowledgeBaseConfig) error
	// GetLatestContentUpdatedAt returns the latest content update time across
	// the document KB's knowledges and FAQ KB's FAQ chunks.
	GetLatestContentUpdatedAt(ctx context.Context, tenantID uint64, documentKBID, faqKBID string) (*time.Time, error)
}

// MedicalKnowledgeBaseConfigService defines business logic for medical KB configs.
type MedicalKnowledgeBaseConfigService interface {
	// GetOrCreateConfig ensures medical KB config rows exist for the tenant.
	// Each category is checked; missing document_kb_id or faq_kb_id rows
	// trigger auto-creation of the underlying WeKnora KB using the tenant's
	// default models. Returns the full config items for the frontend.
	GetOrCreateConfig(ctx context.Context) (*types.MedicalKBConfigResponse, error)
}
