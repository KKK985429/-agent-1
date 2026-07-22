package types

// MedicalSearchRequest 三路统一检索请求
type MedicalSearchRequest struct {
	Categories       []string `json:"categories"        binding:"required"` // 医疗分类: symptom/disease/drug/lab，可多选
	Query            string   `json:"query"              binding:"required"` // 查询文本
	DocMatchCount    int      `json:"doc_match_count"`                      // 文档路返回条数，默认 10
	FAQMatchCount    int      `json:"faq_match_count"`                      // FAQ 路返回条数，默认 5
	WikiLimit        int      `json:"wiki_limit"`                           // Wiki 路返回条数，默认 5
	VectorThreshold  float64  `json:"vector_threshold"`                     // 向量阈值，默认 0.15
	KeywordThreshold float64  `json:"keyword_threshold"`                    // 关键词阈值，默认 0.3
}

// MedicalSearchSource 检索来源
type MedicalSearchSource string

const (
	MedicalSourceDocument MedicalSearchSource = "document"
	MedicalSourceFAQ      MedicalSearchSource = "faq"
	MedicalSourceWiki     MedicalSearchSource = "wiki"
)

// UnifiedMedicalSearchResult 统一检索结果
type UnifiedMedicalSearchResult struct {
	Source    MedicalSearchSource `json:"source"`     // document / faq / wiki
	Score     float64             `json:"score"`      // 归一化到 [0,1] 的相关性分数
	Title     string              `json:"title"`      // 标题/问题
	Content   string              `json:"content"`    // 内容/答案
	Summary   string              `json:"summary"`    // 摘要
	KBID      string              `json:"kb_id"`      // 所属知识库 ID

	// Document 专属
	ChunkType   string `json:"chunk_type,omitempty"`   // text / summary / table_column
	KnowledgeID string `json:"knowledge_id,omitempty"`  // 源文件 ID
	FileName    string `json:"file_name,omitempty"`     // 源文件名

	// FAQ 专属
	FAQEntryID       int      `json:"faq_entry_id,omitempty"`
	Answers          []string `json:"answers,omitempty"`
	SimilarQuestions []string `json:"similar_questions,omitempty"`

	// Wiki 专属
	WikiSlug     string   `json:"wiki_slug,omitempty"`
	WikiPageType string   `json:"wiki_page_type,omitempty"` // entity / concept / summary
	WikiAliases  []string `json:"wiki_aliases,omitempty"`
	WikiInLinks  []string `json:"wiki_in_links,omitempty"`
}

// MedicalSearchResponse 统一检索响应
type MedicalSearchResponse struct {
	Results []*UnifiedMedicalSearchResult `json:"results"`

	// 三路原始结果（调试用，保留原始分数）
	DocRaw  []*SearchResult `json:"doc_raw,omitempty"`
	FAQRaw  []*FAQEntry     `json:"faq_raw,omitempty"`
	WikiRaw []*WikiPage     `json:"wiki_raw,omitempty"`
}
