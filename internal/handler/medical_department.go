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
// @Description  分页查询科室列表，支持按名称/编号模糊搜索、按启用状态筛选
// @Description  返回格式: {"success": true, "data": {"list": [...], "total": N, "page": 1, "page_size": 20}}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        keyword   query     string  false  "搜索关键词（科室名称/编号）"
// @Param        enabled   query     bool    false  "启用状态筛选（true/false），不传则查全部"
// @Param        page      query     int     false  "页码，默认 1"
// @Param        page_size query     int     false  "每页数量，默认 20"
// @Success      200       {object}  map[string]interface{}  "科室分页列表，data 为 MedicalDepartmentListResponse"
// @Failure      401       {object}  errors.AppError         "未登录"
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
// @Description  根据 ID 获取单个科室信息，用于编辑前回显
// @Description  返回格式: {"success": true, "data": {"id": "...", "name": "呼吸科", "code": "HX001", "hospital_area": "中心院区", "enabled": true, "created_by": "...", "created_at": "...", "updated_at": "..."}}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "科室 ID（UUID）"
// @Success      200  {object}  map[string]interface{}  "科室详情，data 为 MedicalDepartmentResponse"
// @Failure      404  {object}  errors.AppError         "科室不存在"
// @Failure      401  {object}  errors.AppError         "未登录"
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
// @Description  创建新的医疗科室，同一租户下科室编号（code）不能重复
// @Description  请求体: {"name": "呼吸科", "code": "HX001", "hospital_area": "中心院区", "enabled": true}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        request  body      types.CreateMedicalDepartmentRequest  true  "科室信息，name/code/hospital_area 必填"
// @Success      200      {object}  map[string]interface{}                "创建的科室，data 为 MedicalDepartmentResponse"
// @Failure      400      {object}  errors.AppError                       "参数校验失败（必填项为空 / 编号已存在）"
// @Failure      401      {object}  errors.AppError                       "未登录或无权限（需要 Contributor+）"
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
// @Description  更新科室信息，只传需要修改的字段即可；同一租户下编号不能和其他科室重复
// @Description  请求体: {"name": "呼吸科", "code": "HX001", "hospital_area": "中心院区", "enabled": true}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id       path      string                               true  "科室 ID（UUID）"
// @Param        request  body      types.UpdateMedicalDepartmentRequest  true  "需更新的字段，nil 表示不修改"
// @Success      200      {object}  map[string]interface{}                "更新后的科室，data 为 MedicalDepartmentResponse"
// @Failure      400      {object}  errors.AppError                       "参数校验失败"
// @Failure      404      {object}  errors.AppError                       "科室不存在"
// @Failure      409      {object}  errors.AppError                       "编号已被其他科室占用"
// @Failure      401      {object}  errors.AppError                       "未登录或无权限（需要 Contributor+）"
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
// @Description  软删除科室（标记 deleted_at 而非物理删除）
// @Description  返回格式: {"success": true}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "科室 ID（UUID）"
// @Success      200  {object}  map[string]interface{}  "删除成功，无 data 字段"
// @Failure      404  {object}  errors.AppError         "科室不存在"
// @Failure      401  {object}  errors.AppError         "未登录或无权限（需要 Contributor+）"
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
// @Description  获取院区下拉选项列表
// @Description  返回格式: {"success": true, "data": [{"label": "中心院区", "value": "中心院区"}, ...]}
// @Tags         医疗-科室管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "院区选项列表，data 为 [{label, value}, ...]"
// @Failure      401  {object}  errors.AppError         "未登录"
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
