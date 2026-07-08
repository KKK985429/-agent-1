package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

type MedicalSearchHandler struct {
	service *service.MedicalSearchService
}

func NewMedicalSearchHandler(service *service.MedicalSearchService) *MedicalSearchHandler {
	return &MedicalSearchHandler{service: service}
}

// Search godoc
// @Summary      医疗三路统一检索
// @Description 并行检索文档、FAQ、Wiki，统一格式合并返回
// @Tags         医疗
// @Accept       json
// @Produce      json
// @Param        request body types.MedicalSearchRequest true "检索请求"
// @Success      200 {object} map[string]interface{} "统一检索结果"
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
