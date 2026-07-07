-- Migration 000061: Create medical_knowledge_base_configs table
--
-- Stores the mapping between medical business entries (symptom / disease / drug / lab)
-- and their underlying WeKnora knowledge bases. Each medical entry needs two KBs:
--   - document KB  (for file upload / file list / file detail)
--   - faq KB       (for Q&A create / list / import / detail)
--
-- The GET /api/v1/medical/knowledge-base-config endpoint lazily creates missing KBs
-- on first access per tenant, using the tenant's default KnowledgeQA and Embedding
-- models. Subsequent requests return the saved config without re-creating.

CREATE TABLE IF NOT EXISTS medical_knowledge_base_configs (
    id             VARCHAR(36)  PRIMARY KEY,
    tenant_id      BIGINT       NOT NULL,
    category       VARCHAR(32)  NOT NULL,   -- symptom / disease / drug / lab
    display_name   VARCHAR(128) NOT NULL,   -- 症状知识库 / 疾病知识库 / 药品知识库 / 检验检查知识库
    document_kb_id VARCHAR(36)  NULL,       -- WeKnora document-type KB ID (auto-created or manually bound)
    faq_kb_id      VARCHAR(36)  NULL,       -- WeKnora FAQ-type KB ID (auto-created or manually bound)
    created_at     TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- One config row per tenant per category
CREATE UNIQUE INDEX idx_medical_kb_configs_tenant_category
    ON medical_knowledge_base_configs(tenant_id, category);
