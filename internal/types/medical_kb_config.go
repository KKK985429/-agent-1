package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MedicalKnowledgeBaseConfig maps a medical business category to its underlying
// WeKnora knowledge bases. Each medical entry (symptom / disease / drug / lab)
// needs two KBs: one document-type for files and one FAQ-type for Q&A.
type MedicalKnowledgeBaseConfig struct {
	ID           string    `json:"id"              gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64    `json:"tenant_id"       gorm:"type:bigint;not null;index:idx_mkbc_tenant_id"`
	Category     string    `json:"category"        gorm:"type:varchar(32);not null"`
	DisplayName  string    `json:"display_name"    gorm:"type:varchar(128);not null"`
	DocumentKBID string    `json:"document_kb_id"  gorm:"column:document_kb_id;type:varchar(36)"`
	FAQKBID      string    `json:"faq_kb_id"       gorm:"column:faq_kb_id;type:varchar(36)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate populates the UUID primary key on insert.
func (c *MedicalKnowledgeBaseConfig) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// MedicalKBConfigItem is the API response item for a single medical KB entry.
type MedicalKBConfigItem struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	DocumentKBID    string `json:"document_kb_id"`
	FAQKBID         string `json:"faq_kb_id"`
	LatestUpdatedAt string `json:"latest_updated_at"`
}

// MedicalKBConfigResponse is the top-level response for GET .../knowledge-base-config.
type MedicalKBConfigResponse struct {
	Items []MedicalKBConfigItem `json:"items"`
}
