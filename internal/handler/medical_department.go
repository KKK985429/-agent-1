package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// MedicalDepartmentHandler handles HTTP requests for medical department management.
type MedicalDepartmentHandler struct {
	service interfaces.MedicalDepartmentService
}

// NewMedicalDepartmentHandler creates a new MedicalDepartmentHandler.
func NewMedicalDepartmentHandler(service interfaces.MedicalDepartmentService) *MedicalDepartmentHandler {
	return &MedicalDepartmentHandler{service: service}
}

// ListDepartments godoc
// @Summary      获取科室列表
// @Description  获取科室列表，支持搜索和筛选
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        keyword  query     string  false  "搜索关键词（科室名称/编号）"
// @Param        enabled  query     bool    false  "启用状态筛选"
// @Param        page     query     int     false  "页码"
// @Param        page_size query    int     false  "每页数量"
// @Success      200      {object}  map[string]interface{}  "科室列表"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/departments [get]
func (h *MedicalDepartmentHandler) ListDepartments(c *gin.Context) {
	ctx := c.Request.Context()

	var req types.ListMedicalDepartmentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error(ctx, "Failed to bind list departments query", err)
		c.Error(errors.NewBadRequestError("查询参数不合法").WithDetails(err.Error()))
		return
	}

	// Parse enabled filter — accept "true"/"false" as string, convert to bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		v := enabledStr == "true" || enabledStr == "1"
		req.Enabled = &v
	}

	result, err := h.service.ListDepartments(ctx, &req)
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

// GetDepartment godoc
// @Summary      获取科室详情
// @Description  根据ID获取单个科室信息
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "科室ID"
// @Success      200  {object}  map[string]interface{}  "科室详情"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/departments/{id} [get]
func (h *MedicalDepartmentHandler) GetDepartment(c *gin.Context) {
	ctx := c.Request.Context()
	id := secutils.SanitizeForLog(c.Param("id"))

	result, err := h.service.GetDepartment(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"id": id,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CreateDepartment godoc
// @Summary      新建科室
// @Description  创建新的医疗科室
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        request  body      types.CreateMedicalDepartmentRequest  true  "科室信息"
// @Success      200      {object}  map[string]interface{}  "创建的科室"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/departments [post]
func (h *MedicalDepartmentHandler) CreateDepartment(c *gin.Context) {
	ctx := c.Request.Context()

	var req types.CreateMedicalDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind create department payload", err)
		c.Error(errors.NewBadRequestError("请求参数不合法").WithDetails(err.Error()))
		return
	}

	// Get current user ID from context
	userID, _ := types.UserIDFromContext(ctx)

	result, err := h.service.CreateDepartment(ctx, &req, userID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"name": secutils.SanitizeForLog(req.Name),
			"code": secutils.SanitizeForLog(req.Code),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// UpdateDepartment godoc
// @Summary      编辑科室
// @Description  更新科室信息
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id       path      string                               true  "科室ID"
// @Param        request  body      types.UpdateMedicalDepartmentRequest  true  "科室更新信息"
// @Success      200      {object}  map[string]interface{}  "更新后的科室"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/departments/{id} [put]
func (h *MedicalDepartmentHandler) UpdateDepartment(c *gin.Context) {
	ctx := c.Request.Context()
	id := secutils.SanitizeForLog(c.Param("id"))

	var req types.UpdateMedicalDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind update department payload", err)
		c.Error(errors.NewBadRequestError("请求参数不合法").WithDetails(err.Error()))
		return
	}

	result, err := h.service.UpdateDepartment(ctx, id, &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"id": id,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// DeleteDepartment godoc
// @Summary      删除科室
// @Description  软删除科室
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "科室ID"
// @Success      200  {object}  map[string]interface{}  "删除结果"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/departments/{id} [delete]
func (h *MedicalDepartmentHandler) DeleteDepartment(c *gin.Context) {
	ctx := c.Request.Context()
	id := secutils.SanitizeForLog(c.Param("id"))

	if err := h.service.DeleteDepartment(ctx, id); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"id": id,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ListHospitalAreas godoc
// @Summary      获取院区列表
// @Description  获取院区下拉选项
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "院区列表"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /medical/hospital-areas [get]
func (h *MedicalDepartmentHandler) ListHospitalAreas(c *gin.Context) {
	ctx := c.Request.Context()

	areas := h.service.ListHospitalAreas(ctx)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    areas,
	})
}
