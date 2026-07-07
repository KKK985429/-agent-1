package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
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
// @Tags         医疗-知识库配置
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "配置列表"
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
