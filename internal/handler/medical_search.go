package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	// Needed for swaggo @Failure type references
	_ "github.com/Tencent/WeKnora/internal/errors"
)

type MedicalSearchHandler struct {
	service *service.MedicalSearchService
}

func NewMedicalSearchHandler(service *service.MedicalSearchService) *MedicalSearchHandler {
	return &MedicalSearchHandler{service: service}
}

// Search godoc
// @Summary      医疗三路统一检索
// @Description  并行检索文档（HybridSearch）、FAQ（语义检索）、Wiki（全文搜索），统一格式合并返回，按 Wiki > FAQ > Document 排序
// @Description  返回格式: {"success": true, "data": {"results": [...], "doc_raw": [...], "faq_raw": [...], "wiki_raw": [...]}}
// @Description  每个结果包含: source(document/faq/wiki), score, title, content, kb_id
// @Tags         医疗-检索
// @Accept       json
// @Produce      json
// @Param        request body types.MedicalSearchRequest true "检索请求，categories 必填（symptom/disease/drug/lab），query 必填"
// @Success      200     {object} map[string]interface{}  "统一检索结果，data 为 MedicalSearchResponse"
// @Failure      400     {object} errors.AppError         "参数不合法（categories 为空 / query 为空）"
// @Failure      401     {object} errors.AppError         "未登录"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/search [post]
func (h *MedicalSearchHandler) Search(c *gin.Context) {
	ctx := c.Request.Context()

	var req types.MedicalSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "请求参数不合法", "details": err.Error()}})
		return
	}

	if len(req.Categories) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "至少选择一个知识库分类"}})
		return
	}

	result, err := h.service.Search(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
