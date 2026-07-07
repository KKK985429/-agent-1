package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// hospitalAreas is the configured list of hospital campuses/areas.
var hospitalAreas = []interfaces.HospitalAreaOption{
	{Label: "中心院区", Value: "中心院区"},
	{Label: "东院区", Value: "东院区"},
	{Label: "西院区", Value: "西院区"},
	{Label: "北院区", Value: "北院区"},
	{Label: "南院区", Value: "南院区"},
}

type medicalDepartmentService struct {
	repo interfaces.MedicalDepartmentRepository
}

func NewMedicalDepartmentService(repo interfaces.MedicalDepartmentRepository) interfaces.MedicalDepartmentService {
	return &medicalDepartmentService{repo: repo}
}

// getTenantID extracts tenant ID from context with proper error handling.
func getTenantID(ctx context.Context) (uint64, error) {
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		logger.Warnf(ctx, "[MedicalDepartment] Tenant ID not found in context")
		return 0, errors.NewUnauthorizedError("无法获取租户信息，请确认登录状态")
	}
	return tenantID, nil
}

// CreateDepartment creates a new department after validation.
func (s *medicalDepartmentService) CreateDepartment(
	ctx context.Context,
	req *types.CreateMedicalDepartmentRequest,
	userID string,
) (*types.MedicalDepartmentResponse, error) {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Validate required fields → 400 Bad Request
	if req.Name == "" {
		return nil, errors.NewBadRequestError("科室名称不能为空")
	}
	if req.Code == "" {
		return nil, errors.NewBadRequestError("科室编号不能为空")
	}
	if req.HospitalArea == "" {
		return nil, errors.NewBadRequestError("院区不能为空")
	}

	// Check code uniqueness → 409 Conflict
	exists, err := s.repo.ExistsByCode(ctx, tenantID, req.Code, "")
	if err != nil {
		return nil, fmt.Errorf("检查科室编号失败: %w", err)
	}
	if exists {
		return nil, errors.NewConflictError(fmt.Sprintf("科室编号 %s 已存在", req.Code))
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	dept := &types.MedicalDepartment{
		TenantID:     tenantID,
		Name:         req.Name,
		Code:         req.Code,
		HospitalArea: req.HospitalArea,
		Enabled:      enabled,
		CreatedBy:    userID,
	}

	if err := s.repo.Create(ctx, dept); err != nil {
		return nil, fmt.Errorf("创建科室失败: %w", err)
	}

	return s.toResponse(dept, ""), nil
}

// GetDepartment retrieves a single department by ID.
func (s *medicalDepartmentService) GetDepartment(
	ctx context.Context,
	id string,
) (*types.MedicalDepartmentResponse, error) {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return nil, err
	}

	dept, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, errors.NewNotFoundError("科室不存在")
	}

	return s.toResponse(dept, ""), nil
}

// ListDepartments lists departments with filters and pagination.
func (s *medicalDepartmentService) ListDepartments(
	ctx context.Context,
	req *types.ListMedicalDepartmentRequest,
) (*types.MedicalDepartmentListResponse, error) {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	depts, total, err := s.repo.List(ctx, tenantID, req.Keyword, req.Enabled, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询科室列表失败: %w", err)
	}

	responseList := make([]*types.MedicalDepartmentResponse, 0, len(depts))
	for _, d := range depts {
		responseList = append(responseList, s.toResponse(d, ""))
	}

	return &types.MedicalDepartmentListResponse{
		List:     responseList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateDepartment updates an existing department after validation.
func (s *medicalDepartmentService) UpdateDepartment(
	ctx context.Context,
	id string,
	req *types.UpdateMedicalDepartmentRequest,
) (*types.MedicalDepartmentResponse, error) {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return nil, err
	}

	dept, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, errors.NewNotFoundError("科室不存在")
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.NewBadRequestError("科室名称不能为空")
		}
		dept.Name = *req.Name
	}
	if req.Code != nil {
		if *req.Code == "" {
			return nil, errors.NewBadRequestError("科室编号不能为空")
		}
		exists, err := s.repo.ExistsByCode(ctx, tenantID, *req.Code, id)
		if err != nil {
			return nil, fmt.Errorf("检查科室编号失败: %w", err)
		}
		if exists {
			return nil, errors.NewConflictError(fmt.Sprintf("科室编号 %s 已存在", *req.Code))
		}
		dept.Code = *req.Code
	}
	if req.HospitalArea != nil {
		if *req.HospitalArea == "" {
			return nil, errors.NewBadRequestError("院区不能为空")
		}
		dept.HospitalArea = *req.HospitalArea
	}
	if req.Enabled != nil {
		dept.Enabled = *req.Enabled
	}

	if err := s.repo.Update(ctx, dept); err != nil {
		return nil, fmt.Errorf("更新科室失败: %w", err)
	}

	return s.toResponse(dept, ""), nil
}

// DeleteDepartment soft-deletes a department.
func (s *medicalDepartmentService) DeleteDepartment(
	ctx context.Context,
	id string,
) error {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return err
	}

	if _, err := s.repo.GetByID(ctx, tenantID, id); err != nil {
		return errors.NewNotFoundError("科室不存在")
	}

	return s.repo.Delete(ctx, tenantID, id)
}

// ListHospitalAreas returns the configured hospital area options.
func (s *medicalDepartmentService) ListHospitalAreas(
	ctx context.Context,
) []interfaces.HospitalAreaOption {
	return hospitalAreas
}

// toResponse converts a MedicalDepartment to a response DTO.
func (s *medicalDepartmentService) toResponse(
	dept *types.MedicalDepartment,
	createdByName string,
) *types.MedicalDepartmentResponse {
	return &types.MedicalDepartmentResponse{
		ID:            dept.ID,
		Name:          dept.Name,
		Code:          dept.Code,
		HospitalArea:  dept.HospitalArea,
		Enabled:       dept.Enabled,
		CreatedBy:     dept.CreatedBy,
		CreatedByName: createdByName,
		CreatedAt:     dept.CreatedAt,
		UpdatedAt:     dept.UpdatedAt,
	}
}
