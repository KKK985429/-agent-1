package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	// Needed for swaggo @Failure type references
	_ "github.com/Tencent/WeKnora/internal/errors"
)

// MedicalKBConfigHandler handles HTTP requests for medical knowledge base configuration.
type MedicalKBConfigHandler struct {
	service interfaces.MedicalKnowledgeBaseConfigService
}

// NewMedicalKBConfigHandler creates a new handler.
func NewMedicalKBConfigHandler(service interfaces.MedicalKnowledgeBaseConfigService) *MedicalKBConfigHandler {
	return &MedicalKBConfigHandler{service: service}
}

// GetOrCreateConfig godoc
// @Summary      获取医疗知识库配置
// @Description  返回当前租户的医疗知识库配置（症状/疾病/药品/检验检查）。首次访问时自动创建缺失的底层 WeKnora 知识库（document + FAQ），使用租户默认模型。
// @Description
// @Description  === 返回格式 ===
// @Description  {"success": true, "data": {"items": [
// @Description    {"key": "symptom",  "name": "症状知识库",     "document_kb_id": "abc-123", "faq_kb_id": "def-456"},
// @Description    {"key": "disease",  "name": "疾病知识库",     "document_kb_id": "ghi-789", "faq_kb_id": "jkl-012"},
// @Description    {"key": "drug",     "name": "药品知识库",     "document_kb_id": "mno-345", "faq_kb_id": "pqr-678"},
// @Description    {"key": "lab",      "name": "检验检查知识库",  "document_kb_id": "stu-901", "faq_kb_id": "vwx-234"}
// @Description  ]}}
// @Description
// @Description  === 对接方式：拿到配置后如何使用 ===
// @Description  1. 用户选择分类（如 symptom），取对应 document_kb_id 和 faq_kb_id
// @Description  2. 文件操作 → 使用 document_kb_id 调已有知识库接口：
// @Description     - 上传: POST /api/v1/knowledge-bases/{document_kb_id}/knowledge/file
// @Description     - 列表: GET  /api/v1/knowledge-bases/{document_kb_id}/knowledge
// @Description     - 详情: GET  /api/v1/knowledge/{knowledge_id}
// @Description     - 下载: GET  /api/v1/knowledge/{knowledge_id}/download
// @Description  3. Q&A 操作 → 使用 faq_kb_id 调已有 FAQ 接口：
// @Description     - 新建: POST /api/v1/knowledge-bases/{faq_kb_id}/faq/entry
// @Description     - 列表: GET  /api/v1/knowledge-bases/{faq_kb_id}/faq/entries
// @Description     - 详情: GET  /api/v1/knowledge-bases/{faq_kb_id}/faq/entries/{entry_id}
// @Description     - 更新: PUT  /api/v1/knowledge-bases/{faq_kb_id}/faq/entries/{entry_id}
// @Description     - 批量启停: PUT /api/v1/knowledge-bases/{faq_kb_id}/faq/entries/fields
// @Description     - 批量导入: POST /api/v1/knowledge-bases/{faq_kb_id}/faq/entries
// @Description     - 删除: DELETE /api/v1/knowledge-bases/{faq_kb_id}/faq/entries
// @Description     - 导入进度: GET /api/v1/faq/import/progress/{task_id}
// @Description  4. 统一检索 → 直接调 POST /api/v1/medical/search
// @Description
// @Description  === 认证方式 ===
// @Description  所有接口在 Authorization Header 中传 JWT Token:
// @Description    Authorization: Bearer <token>
// @Description  租户隔离由 Token 自动完成，无需手动传 tenant_id。
// @Description
// @Description  category 取值: symptom / disease / drug / lab
// @Tags         医疗-知识库配置
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "配置列表，data.items 为 MedicalKBConfigItem[]"
// @Failure      401  {object}  errors.AppError         "未登录"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/knowledge-base-config [get]
func (h *MedicalKBConfigHandler) GetOrCreateConfig(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := h.service.GetOrCreateConfig(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
