package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// MedicalSearchService 医疗三路统一检索服务
type MedicalSearchService struct {
	kbConfigRepo interfaces.MedicalKnowledgeBaseConfigRepository
	kbService    interfaces.KnowledgeBaseService
	kgService    interfaces.KnowledgeService
	wikiService  interfaces.WikiPageService
}

func NewMedicalSearchService(
	kbConfigRepo interfaces.MedicalKnowledgeBaseConfigRepository,
	kbService interfaces.KnowledgeBaseService,
	kgService interfaces.KnowledgeService,
	wikiService interfaces.WikiPageService,
) *MedicalSearchService {
	return &MedicalSearchService{
		kbConfigRepo: kbConfigRepo,
		kbService:    kbService,
		kgService:    kgService,
		wikiService:  wikiService,
	}
}

func getTenantIDFromCtx(ctx context.Context) (uint64, error) {
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		return 0, fmt.Errorf("无法获取租户信息")
	}
	return tenantID, nil
}

// Search 执行三路统一检索
func (s *MedicalSearchService) Search(ctx context.Context, req *types.MedicalSearchRequest) (*types.MedicalSearchResponse, error) {
	tenantID, err := getTenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	// 默认值
	if req.DocMatchCount <= 0 { req.DocMatchCount = 10 }
	if req.FAQMatchCount <= 0 { req.FAQMatchCount = 5 }
	if req.WikiLimit <= 0      { req.WikiLimit = 5 }
	if req.VectorThreshold <= 0  { req.VectorThreshold = 0.15 }
	if req.KeywordThreshold <= 0 { req.KeywordThreshold = 0.3 }

	// 1. 收集选中分类的 doc_kb_ids 和 faq_kb_ids
	var docKBIDs, faqKBIDs []string
	for _, cat := range req.Categories {
		cfg, err := s.kbConfigRepo.GetByTenantAndCategory(ctx, tenantID, cat)
		if err != nil || cfg == nil { continue }
		if cfg.DocumentKBID != "" { docKBIDs = append(docKBIDs, cfg.DocumentKBID) }
		if cfg.FAQKBID != ""      { faqKBIDs  = append(faqKBIDs, cfg.FAQKBID) }
	}

	if len(docKBIDs) == 0 && len(faqKBIDs) == 0 {
		return &types.MedicalSearchResponse{Results: []*types.UnifiedMedicalSearchResult{}}, nil
	}

	// 2. 并行三路检索
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		docResults []*types.SearchResult
		faqEntries []*types.FAQEntry
		wikiPages  []*types.WikiPage
	)

	// 路一：文档 HybridSearch
	if len(docKBIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			params := types.SearchParams{
				QueryText:        req.Query,
				MatchCount:       req.DocMatchCount,
				KnowledgeBaseIDs: docKBIDs,
				VectorThreshold:  req.VectorThreshold,
				KeywordThreshold: req.KeywordThreshold,
			}
			results, err := s.kbService.HybridSearch(ctx, docKBIDs[0], params)
			if err != nil {
				logger.Warnf(ctx, "[MedicalSearch] doc HybridSearch failed: %v", err)
				return
			}
			mu.Lock()
			docResults = results
			mu.Unlock()
		}()
	}

	// 路二：FAQ 检索
	if len(faqKBIDs) > 0 {
		for _, faqKBID := range faqKBIDs {
			wg.Add(1)
			go func(kbID string) {
				defer wg.Done()
				faqReq := &types.FAQSearchRequest{
					QueryText:  req.Query,
					MatchCount: req.FAQMatchCount,
				}
				entries, err := s.kgService.SearchFAQEntries(ctx, kbID, faqReq)
				if err != nil {
					logger.Warnf(ctx, "[MedicalSearch] FAQ search failed for %s: %v", kbID, err)
					return
				}
				mu.Lock()
				faqEntries = append(faqEntries, entries...)
				mu.Unlock()
			}(faqKBID)
		}
	}

	// 路三：Wiki 检索
	if len(docKBIDs) > 0 {
		for _, docKBID := range docKBIDs {
			wg.Add(1)
			go func(kbID string) {
				defer wg.Done()
				pages, err := s.wikiService.SearchPages(ctx, kbID, req.Query, req.WikiLimit)
				if err != nil {
					logger.Warnf(ctx, "[MedicalSearch] wiki search failed for %s: %v", kbID, err)
					return
				}
				mu.Lock()
				wikiPages = append(wikiPages, pages...)
				mu.Unlock()
			}(docKBID)
		}
	}

	wg.Wait()

	// 3. 格式化 + 融合
	var unified []*types.UnifiedMedicalSearchResult

	for _, r := range docResults {
		unified = append(unified, convertDocResult(r))
	}

	for _, r := range faqEntries {
		unified = append(unified, convertFAQEntry(r))
	}

	for _, p := range wikiPages {
		unified = append(unified, convertWikiPage(p))
	}

	// 4. 排序：Wiki > FAQ > Document，同源按 score 降序
	sort.SliceStable(unified, func(i, j int) bool {
		pi, pj := sourcePriority(unified[i].Source), sourcePriority(unified[j].Source)
		if pi != pj { return pi < pj }
		return unified[i].Score > unified[j].Score
	})

	// 去重：Wiki 和 Document 可能内容重叠，简单按 title 去重（保留高优先级）
	seen := make(map[string]bool)
	var deduped []*types.UnifiedMedicalSearchResult
	for _, r := range unified {
		key := strings.ToLower(strings.TrimSpace(r.Title))
		if key == "" { key = r.Content[:min(50, len(r.Content))] }
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, r)
		}
	}

	return &types.MedicalSearchResponse{
		Results: deduped,
		DocRaw:  docResults,
		FAQRaw:  faqEntries,
		WikiRaw: wikiPages,
	}, nil
}

// ── 格式化转换 ──

func convertDocResult(r *types.SearchResult) *types.UnifiedMedicalSearchResult {
	content := r.Content
	if content == "" { content = r.MatchedContent }
	return &types.UnifiedMedicalSearchResult{
		Source:      types.MedicalSourceDocument,
		Score:       math.Round(r.Score*10000) / 10000,
		Title:       r.KnowledgeTitle,
		Content:     content,
		KBID:        r.KnowledgeBaseID,
		ChunkType:   r.ChunkType,
		KnowledgeID: r.KnowledgeID,
		FileName:    r.KnowledgeFilename,
	}
}

func convertFAQEntry(r *types.FAQEntry) *types.UnifiedMedicalSearchResult {
	content := strings.Join(r.Answers, "\n")
	if content == "" { content = r.StandardQuestion }
	return &types.UnifiedMedicalSearchResult{
		Source:           types.MedicalSourceFAQ,
		Score:            math.Round(r.Score*10000) / 10000,
		Title:            r.StandardQuestion,
		Content:          content,
		Summary:          r.StandardQuestion,
		KBID:             r.KnowledgeBaseID,
		FAQEntryID:       int(r.ID),
		Answers:          r.Answers,
		SimilarQuestions: r.SimilarQuestions,
	}
}

func convertWikiPage(p *types.WikiPage) *types.UnifiedMedicalSearchResult {
	return &types.UnifiedMedicalSearchResult{
		Source:       types.MedicalSourceWiki,
		Score:        1.0, // Wiki 无分数，默认为 1.0（LLM 精炼知识优先级最高）
		Title:        p.Title,
		Content:      p.Content,
		Summary:      p.Summary,
		KBID:         p.KnowledgeBaseID,
		WikiSlug:     p.Slug,
		WikiPageType: p.PageType,
		WikiAliases:  p.Aliases,
		WikiInLinks:  p.InLinks,
	}
}

// ── 分数处理 ──

func sourcePriority(s types.MedicalSearchSource) int {
	switch s {
	case types.MedicalSourceWiki:     return 0
	case types.MedicalSourceFAQ:      return 1
	case types.MedicalSourceDocument: return 2
	default:                          return 3
	}
}
