package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type MCPQueryStatsHandler struct {
	service interfaces.MCPQueryStatsService
}

func NewMCPQueryStatsHandler(service interfaces.MCPQueryStatsService) *MCPQueryStatsHandler {
	return &MCPQueryStatsHandler{service: service}
}

// GetStats godoc
// @Summary      获取租户 MCP 查询统计
// @Description  获取当前租户通过 MCP 执行混合检索的查询次数和命中次数
// @Tags         MCP服务
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  errors.AppError
// @Failure      500  {object}  errors.AppError
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /mcp-query-stats [get]
func (h *MCPQueryStatsHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(errors.NewUnauthorizedError("tenant ID not found"))
		return
	}

	stats, err := h.service.GetStats(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError("failed to get MCP query statistics"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}
