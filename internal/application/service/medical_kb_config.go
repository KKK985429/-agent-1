package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// predefinedMedicalCategories defines the medical business entries.
// Each entry maps to two underlying WeKnora KBs (document + FAQ).
var predefinedMedicalCategories = []struct {
	Category    string
	DisplayName string
}{
	{Category: "symptom", DisplayName: "症状知识库"},
	{Category: "disease", DisplayName: "疾病知识库"},
	{Category: "drug", DisplayName: "药品知识库"},
	{Category: "lab", DisplayName: "检验检查知识库"},
}

type medicalKBConfigService struct {
	repo    interfaces.MedicalKnowledgeBaseConfigRepository
	kbSvc   interfaces.KnowledgeBaseService
	modelRepo interfaces.ModelRepository
}

func NewMedicalKnowledgeBaseConfigService(
	repo interfaces.MedicalKnowledgeBaseConfigRepository,
	kbSvc interfaces.KnowledgeBaseService,
	modelRepo interfaces.ModelRepository,
) interfaces.MedicalKnowledgeBaseConfigService {
	return &medicalKBConfigService{
		repo:      repo,
		kbSvc:     kbSvc,
		modelRepo: modelRepo,
	}
}

// getTenantID extracts tenant ID from context.
func (s *medicalKBConfigService) getTenantID(ctx context.Context) (uint64, error) {
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		return 0, fmt.Errorf("tenant ID not found in context")
	}
	return tenantID, nil
}

// findDefaultModelID returns the ID of the tenant's default model for the given type.
// Returns empty string if no default is configured.
func (s *medicalKBConfigService) findDefaultModelID(
	ctx context.Context, tenantID uint64, modelType types.ModelType,
) string {
	models, err := s.modelRepo.List(ctx, tenantID, modelType, "")
	if err != nil {
		logger.Warnf(ctx, "[MedicalKBConfig] Failed to list models of type %s: %v", modelType, err)
		return ""
	}
	for _, m := range models {
		if m.IsDefault {
			return m.ID
		}
	}
	// Fallback: use the first available model of the requested type.
	if len(models) > 0 {
		logger.Infof(ctx, "[MedicalKBConfig] No default %s model set; falling back to first available: %s", modelType, models[0].ID)
		return models[0].ID
	}
	return ""
}

// createDocumentKB creates a document-type knowledge base for a medical category.
func (s *medicalKBConfigService) createDocumentKB(
	ctx context.Context, tenantID uint64, category, displayName string,
	summaryModelID, embeddingModelID string,
) (*types.KnowledgeBase, error) {
	kb := &types.KnowledgeBase{
		Name:             fmt.Sprintf("%s - 文档", displayName),
		Type:             types.KnowledgeBaseTypeDocument,
		Description:      fmt.Sprintf("医疗%s - 文档（系统自动创建）", displayName),
		TenantID:         tenantID,
		SummaryModelID:   summaryModelID,
		EmbeddingModelID: embeddingModelID,
	}
	kb.EnsureDefaults()
	kb.IndexingStrategy.WikiEnabled = true // 开启 Wiki：上传文档后自动生成结构化百科页面

	logger.Infof(ctx, "[MedicalKBConfig] Creating document KB for %s (tenant=%d)", category, tenantID)
	created, err := s.kbSvc.CreateKnowledgeBase(ctx, kb)
	if err != nil {
		return nil, fmt.Errorf("create document KB for %s: %w", category, err)
	}
	logger.Infof(ctx, "[MedicalKBConfig] Created document KB %s for %s", created.ID, category)
	return created, nil
}

// createFAQKB creates a FAQ-type knowledge base for a medical category.
func (s *medicalKBConfigService) createFAQKB(
	ctx context.Context, tenantID uint64, category, displayName string,
	summaryModelID, embeddingModelID string,
) (*types.KnowledgeBase, error) {
	kb := &types.KnowledgeBase{
		Name:             fmt.Sprintf("%s - Q&A", displayName),
		Type:             types.KnowledgeBaseTypeFAQ,
		Description:      fmt.Sprintf("医疗%s - Q&A（系统自动创建）", displayName),
		TenantID:         tenantID,
		SummaryModelID:   summaryModelID,
		EmbeddingModelID: embeddingModelID,
		FAQConfig: &types.FAQConfig{
			IndexMode:         types.FAQIndexModeQuestionOnly,
			QuestionIndexMode: types.FAQQuestionIndexModeSeparate,
		},
	}
	kb.EnsureDefaults()

	logger.Infof(ctx, "[MedicalKBConfig] Creating FAQ KB for %s (tenant=%d)", category, tenantID)
	created, err := s.kbSvc.CreateKnowledgeBase(ctx, kb)
	if err != nil {
		return nil, fmt.Errorf("create FAQ KB for %s: %w", category, err)
	}
	logger.Infof(ctx, "[MedicalKBConfig] Created FAQ KB %s for %s", created.ID, category)
	return created, nil
}

// GetOrCreateConfig ensures each medical category has underlying KBs and returns config items.
func (s *medicalKBConfigService) GetOrCreateConfig(
	ctx context.Context,
) (*types.MedicalKBConfigResponse, error) {
	tenantID, err := s.getTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Find default models for the tenant (best-effort; may be empty).
	summaryModelID := s.findDefaultModelID(ctx, tenantID, types.ModelTypeKnowledgeQA)
	embeddingModelID := s.findDefaultModelID(ctx, tenantID, types.ModelTypeEmbedding)

	logger.Infof(ctx, "[MedicalKBConfig] Default models for tenant %d — summary: %s, embedding: %s",
		tenantID, summaryModelID, embeddingModelID)

	items := make([]types.MedicalKBConfigItem, 0, len(predefinedMedicalCategories))

	for _, cat := range predefinedMedicalCategories {
		existing, err := s.repo.GetByTenantAndCategory(ctx, tenantID, cat.Category)
		if err != nil {
			return nil, fmt.Errorf("query config for %s: %w", cat.Category, err)
		}

		var docKBID, faqKBID string

		if existing != nil {
			docKBID = existing.DocumentKBID
			faqKBID = existing.FAQKBID
		}

		// Auto-create document KB if missing.
		if docKBID == "" && summaryModelID != "" && embeddingModelID != "" {
			docKB, err := s.createDocumentKB(ctx, tenantID, cat.Category, cat.DisplayName,
				summaryModelID, embeddingModelID)
			if err != nil {
				logger.Warnf(ctx, "[MedicalKBConfig] Failed to auto-create document KB for %s: %v", cat.Category, err)
			} else {
				docKBID = docKB.ID
			}
		}

		// Auto-create FAQ KB if missing.
		if faqKBID == "" && summaryModelID != "" && embeddingModelID != "" {
			faqKB, err := s.createFAQKB(ctx, tenantID, cat.Category, cat.DisplayName,
				summaryModelID, embeddingModelID)
			if err != nil {
				logger.Warnf(ctx, "[MedicalKBConfig] Failed to auto-create FAQ KB for %s: %v", cat.Category, err)
			} else {
				faqKBID = faqKB.ID
			}
		}

		// Save/update the config row if any KB was created.
		if (existing == nil) || (existing.DocumentKBID != docKBID) || (existing.FAQKBID != faqKBID) {
			cfg := &types.MedicalKnowledgeBaseConfig{
				TenantID:     tenantID,
				Category:     cat.Category,
				DisplayName:  cat.DisplayName,
				DocumentKBID: docKBID,
				FAQKBID:      faqKBID,
			}
			if existing != nil {
				cfg.ID = existing.ID
				cfg.CreatedAt = existing.CreatedAt
			}
			if err := s.repo.Upsert(ctx, cfg); err != nil {
				logger.Warnf(ctx, "[MedicalKBConfig] Failed to upsert config for %s: %v", cat.Category, err)
			}
		}

		items = append(items, types.MedicalKBConfigItem{
			Key:          cat.Category,
			Name:         cat.DisplayName,
			DocumentKBID: docKBID,
			FAQKBID:      faqKBID,
		})
	}

	return &types.MedicalKBConfigResponse{Items: items}, nil
}
