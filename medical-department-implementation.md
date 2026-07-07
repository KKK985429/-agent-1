# 科室管理功能 — 完整实现文档

## 目录

1. [整体架构](#1-整体架构)
2. [数据库设计](#2-数据库设计)
3. [文件清单](#3-文件清单)
4. [后端实现详解](#4-后端实现详解)
   - [4.1 数据库迁移](#41-数据库迁移)
   - [4.2 数据模型 Types](#42-数据模型-types)
   - [4.3 接口定义 Interfaces](#43-接口定义-interfaces)
   - [4.4 数据访问层 Repository](#44-数据访问层-repository)
   - [4.5 业务逻辑层 Service](#45-业务逻辑层-service)
   - [4.6 HTTP 处理层 Handler](#46-http-处理层-handler)
   - [4.7 路由注册 Router](#47-路由注册-router)
   - [4.8 依赖注入 Container](#48-依赖注入-container)
5. [前端实现详解](#5-前端实现详解)
   - [5.1 API 封装层](#51-api-封装层)
   - [5.2 科室列表页面](#52-科室列表页面)
   - [5.3 科室表单弹窗](#53-科室表单弹窗)
   - [5.4 路由注册](#54-路由注册)
   - [5.5 菜单注册](#55-菜单注册)
   - [5.6 国际化](#56-国际化)
6. [遇到问题与解决](#6-遇到问题与解决)
7. [完整请求链路](#7-完整请求链路)

---

## 1. 整体架构

科室管理是一个**独立的新增模块**，和 WeKnora 知识库系统没有任何耦合。它遵循 WeKnora 现有的三层架构模式：

```
用户浏览器 (Vue3 + TDesign)
    ↓ HTTP
Gin Router (路由注册 + RBAC 权限中间件)
    ↓
Handler (HTTP 请求处理：参数校验、JSON 序列化)
    ↓
Service (业务逻辑：租户隔离、编号唯一性校验)
    ↓
Repository (数据库操作：GORM ORM)
    ↓
PostgreSQL (medical_departments 表)
```

**核心设计原则：**
- 科室是独立的普通关系型数据库表，不和知识库内容关联
- 租户隔离：所有查询必须带 `tenant_id`
- 软删除：使用 GORM 的 `gorm.DeletedAt`，删除时只标记 `deleted_at`
- 编号唯一：同一租户下 `code` 不能重复（通过 partial unique index 保证）
- 复用 WeKnora 现有的 RBAC 权限体系（Viewer+ 查看，Contributor+ 编辑）

---

## 2. 数据库设计

### 表结构

```sql
CREATE TABLE IF NOT EXISTS medical_departments (
    id              VARCHAR(36)  PRIMARY KEY,        -- UUID 主键
    tenant_id       BIGINT       NOT NULL,            -- 租户 ID（多租户数据隔离）
    name            VARCHAR(128) NOT NULL,            -- 科室名称（如"呼吸科"）
    code            VARCHAR(64)  NOT NULL,            -- 科室编号（手动输入，同租户唯一）
    hospital_area   VARCHAR(128) NOT NULL,            -- 院区（如"中心院区"）
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE, -- 启用状态
    created_by      VARCHAR(36),                     -- 创建人用户 ID
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP    NULL                 -- 软删除时间戳
);
```

### 索引设计

```sql
-- 租户查询索引
CREATE INDEX idx_medical_departments_tenant_id  ON medical_departments(tenant_id);
-- 状态筛选索引
CREATE INDEX idx_medical_departments_enabled    ON medical_departments(enabled);
-- 软删除索引
CREATE INDEX idx_medical_departments_deleted_at ON medical_departments(deleted_at);
-- 唯一约束：同租户下活跃记录的 code 唯一
-- 使用 partial index（WHERE deleted_at IS NULL），
-- 这样被软删除的记录不会和新建记录冲突
CREATE UNIQUE INDEX idx_medical_departments_tenant_code_unique
    ON medical_departments(tenant_id, code)
    WHERE deleted_at IS NULL;
```

**为什么用 Partial Unique Index 而不是普通 UNIQUE？**

如果使用 `UNIQUE(tenant_id, code)`，那么当一个科室被软删除后（`deleted_at` 不为 NULL），同租户下无法再创建相同 code 的新科室。使用 `WHERE deleted_at IS NULL` 的 partial index 后，只有活跃记录（未被删除）才参与唯一约束，软删除的记录不会冲突。

---

## 3. 文件清单

### 新增文件（8 个）

| # | 文件路径 | 作用 |
|---|---------|------|
| 1 | `migrations/versioned/000060_medical_departments.up.sql` | 创建表的迁移 SQL |
| 2 | `migrations/versioned/000060_medical_departments.down.sql` | 回滚迁移 SQL |
| 3 | `internal/types/medical_department.go` | 数据模型 + 请求/响应结构体定义 |
| 4 | `internal/types/interfaces/medical_department.go` | Repository + Service 接口定义 |
| 5 | `internal/application/repository/medical_department.go` | 数据库操作实现（GORM） |
| 6 | `internal/application/service/medical_department.go` | 业务逻辑实现 |
| 7 | `internal/handler/medical_department.go` | HTTP 请求处理器 |
| 8 | `frontend/src/api/medical/department/index.ts` | 前端 API 封装 |
| 9 | `frontend/src/views/medical/department/DepartmentList.vue` | 科室列表页面 |
| 10 | `frontend/src/views/medical/department/DepartmentFormDialog.vue` | 新建/编辑弹窗 |

### 修改文件（7 个）

| # | 文件路径 | 作用 | 修改内容 |
|---|---------|------|---------|
| 1 | `internal/container/container.go` | 依赖注入容器 | 注册 Repository、Service、Handler |
| 2 | `internal/router/router.go` | 路由注册 | 添加路由参数 + 路由注册函数 + 调用 |
| 3 | `frontend/src/router/index.ts` | 前端路由 | 添加 `/platform/medical/departments` 路由 |
| 4 | `frontend/src/stores/menu.ts` | 左侧菜单 Store | 添加"知识库管理"菜单项 |
| 5 | `frontend/src/components/menu.vue` | 菜单组件 | 筛选逻辑 + 图标激活状态 |
| 6 | `frontend/src/i18n/locales/zh-CN.ts` | 中文翻译 | 添加 `medicalKB: "知识库管理"` |
| 7 | `frontend/src/i18n/locales/en-US.ts` | 英文翻译 | 添加 `medicalKB: 'Medical KB'` |

---

## 4. 后端实现详解

### 4.1 数据库迁移

**文件：`migrations/versioned/000060_medical_departments.up.sql`**

```sql
-- 迁移编号 000060，在现有最新迁移 000059 基础上递增
-- WeKnora 启动时会自动执行所有未执行的迁移（AUTO_MIGRATE=true）

CREATE TABLE IF NOT EXISTS medical_departments (
    id              VARCHAR(36)  PRIMARY KEY,         -- UUID，由 Go 代码在 BeforeCreate 中生成
    tenant_id       BIGINT       NOT NULL,            -- 租户隔离核心字段
    name            VARCHAR(128) NOT NULL,            -- 科室名称
    code            VARCHAR(64)  NOT NULL,            -- 科室编号（手动输入）
    hospital_area   VARCHAR(128) NOT NULL,            -- 院区
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE, -- 默认启用
    created_by      VARCHAR(36),                     -- 创建人 ID
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP    NULL                 -- 软删除
);

-- 查询优化索引
CREATE INDEX idx_medical_departments_tenant_id  ON medical_departments(tenant_id);
CREATE INDEX idx_medical_departments_enabled    ON medical_departments(enabled);
CREATE INDEX idx_medical_departments_deleted_at ON medical_departments(deleted_at);

-- 关键：部分唯一索引，只约束未删除的记录
CREATE UNIQUE INDEX idx_medical_departments_tenant_code_unique
    ON medical_departments(tenant_id, code)
    WHERE deleted_at IS NULL;
```

**文件：`migrations/versioned/000060_medical_departments.down.sql`**

```sql
DROP TABLE IF EXISTS medical_departments;
```

---

### 4.2 数据模型 (Types)

**文件：`internal/types/medical_department.go`**

这个文件定义了整个模块使用的所有数据结构。

```go
package types

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// MedicalDepartment 是科室的数据库模型
// GORM 会根据 struct tag 自动映射数据库字段
type MedicalDepartment struct {
    // ID 是 UUID 主键，BeforeCreate 钩子自动生成
    ID           string         `json:"id"             gorm:"type:varchar(36);primaryKey"`
    // TenantID 租户隔离 —— 所有查询都会带上这个条件
    TenantID     uint64         `json:"tenant_id"      gorm:"type:bigint;not null"`
    Name         string         `json:"name"           gorm:"type:varchar(128);not null"`
    Code         string         `json:"code"           gorm:"type:varchar(64);not null"`
    HospitalArea string         `json:"hospital_area"  gorm:"type:varchar(128);not null"`
    Enabled      bool           `json:"enabled"        gorm:"default:true"`
    CreatedBy    string         `json:"created_by"     gorm:"type:varchar(36)"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    // gorm.DeletedAt 是关键类型：GORM 会自动处理软删除逻辑
    // 查询时自动加 WHERE deleted_at IS NULL
    // 删除时自动 SET deleted_at = NOW() 而不是物理删除
    DeletedAt    gorm.DeletedAt `json:"deleted_at"     gorm:"index"`
}

// BeforeCreate 是 GORM 的生命周期钩子
// 在插入数据库之前自动调用，生成 UUID 主键
func (d *MedicalDepartment) BeforeCreate(tx *gorm.DB) error {
    if d.ID == "" {
        d.ID = uuid.New().String()
    }
    return nil
}

// CreateMedicalDepartmentRequest 新建科室的请求体
// binding:"required" 是 Gin 框架的校验 tag，自动验证字段非空
type CreateMedicalDepartmentRequest struct {
    Name         string `json:"name"          binding:"required"`
    Code         string `json:"code"          binding:"required"`
    HospitalArea string `json:"hospital_area" binding:"required"`
    Enabled      *bool  `json:"enabled"`                  // 指针类型，可选（默认 true）
}

// UpdateMedicalDepartmentRequest 编辑科室的请求体
// 所有字段都是指针类型 —— nil 表示"不修改这个字段"
type UpdateMedicalDepartmentRequest struct {
    Name         *string `json:"name"`
    Code         *string `json:"code"`
    HospitalArea *string `json:"hospital_area"`
    Enabled      *bool   `json:"enabled"`
}

// ListMedicalDepartmentRequest 列表查询参数
type ListMedicalDepartmentRequest struct {
    Keyword  string `form:"keyword"`                     // 搜索关键词（匹配 name 或 code）
    Enabled  *bool  `form:"enabled"`                     // 按启用状态筛选
    Page     int    `form:"page"`
    PageSize int    `form:"page_size"`
}

// MedicalDepartmentResponse 返回给前端的单个科室数据
type MedicalDepartmentResponse struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Code          string    `json:"code"`
    HospitalArea  string    `json:"hospital_area"`
    Enabled       bool      `json:"enabled"`
    CreatedBy     string    `json:"created_by"`
    CreatedByName string    `json:"created_by_name"`    // 创建人显示名（预留，当前未关联查询）
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

// MedicalDepartmentListResponse 分页列表响应
type MedicalDepartmentListResponse struct {
    List     []*MedicalDepartmentResponse `json:"list"`
    Total    int64                        `json:"total"`
    Page     int                          `json:"page"`
    PageSize int                          `json:"page_size"`
}
```

---

### 4.3 接口定义 (Interfaces)

**文件：`internal/types/interfaces/medical_department.go`**

这个文件定义了 Repository 和 Service 的接口，是 Go 依赖注入的核心——上层代码只依赖接口，不依赖具体实现。

```go
package interfaces

import (
    "context"
    "github.com/Tencent/WeKnora/internal/types"
)

// MedicalDepartmentRepository 数据访问接口
// 所有方法都接受 context.Context，用于传递租户信息和请求追踪
type MedicalDepartmentRepository interface {
    Create(ctx context.Context, dept *types.MedicalDepartment) error
    GetByID(ctx context.Context, tenantID uint64, id string) (*types.MedicalDepartment, error)
    // keyword 搜索名称或编号；enabled=nil 表示不过滤
    List(ctx context.Context, tenantID uint64, keyword string, enabled *bool, page, pageSize int) ([]*types.MedicalDepartment, int64, error)
    Update(ctx context.Context, dept *types.MedicalDepartment) error
    Delete(ctx context.Context, tenantID uint64, id string) error
    // excludeID 用于编辑时排除自身，防止"编号已存在"误判
    ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID string) (bool, error)
}

// MedicalDepartmentService 业务逻辑接口
type MedicalDepartmentService interface {
    CreateDepartment(ctx context.Context, req *types.CreateMedicalDepartmentRequest, userID string) (*types.MedicalDepartmentResponse, error)
    GetDepartment(ctx context.Context, id string) (*types.MedicalDepartmentResponse, error)
    ListDepartments(ctx context.Context, req *types.ListMedicalDepartmentRequest) (*types.MedicalDepartmentListResponse, error)
    UpdateDepartment(ctx context.Context, id string, req *types.UpdateMedicalDepartmentRequest) (*types.MedicalDepartmentResponse, error)
    DeleteDepartment(ctx context.Context, id string) error
    ListHospitalAreas(ctx context.Context) []HospitalAreaOption
}

// HospitalAreaOption 院区下拉选项
type HospitalAreaOption struct {
    Label string `json:"label"`
    Value string `json:"value"`
}
```

---

### 4.4 数据访问层 (Repository)

**文件：`internal/application/repository/medical_department.go`**

Repository 层负责所有数据库操作，使用 GORM 作为 ORM。

```go
package repository

import (
    "context"
    "gorm.io/gorm"
    "github.com/Tencent/WeKnora/internal/types"
    "github.com/Tencent/WeKnora/internal/types/interfaces"
)

// 私有 struct，通过接口暴露 —— Go 的封装模式
type medicalDepartmentRepository struct {
    db *gorm.DB
}

// 构造函数返回接口类型，方便依赖注入和单元测试 mock
func NewMedicalDepartmentRepository(db *gorm.DB) interfaces.MedicalDepartmentRepository {
    return &medicalDepartmentRepository{db: db}
}

// Create 插入新记录
func (r *medicalDepartmentRepository) Create(ctx context.Context, dept *types.MedicalDepartment) error {
    // WithContext 传递请求上下文（包含超时、取消信号、链路追踪 ID）
    return r.db.WithContext(ctx).Create(dept).Error
}

// GetByID 查询单条记录
// 关键：WHERE tenant_id = ? —— 租户隔离，只能查自己租户的数据
// GORM 自动加 WHERE deleted_at IS NULL（因为 MedicalDepartment 有 gorm.DeletedAt）
func (r *medicalDepartmentRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.MedicalDepartment, error) {
    var dept types.MedicalDepartment
    if err := r.db.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, id).
        First(&dept).Error; err != nil {
        return nil, err
    }
    return &dept, nil
}

// List 分页查询，支持关键词搜索和状态筛选
func (r *medicalDepartmentRepository) List(
    ctx context.Context,
    tenantID uint64,
    keyword string,
    enabled *bool,
    page, pageSize int,
) ([]*types.MedicalDepartment, int64, error) {
    // 基础查询：限定租户
    baseQuery := r.db.WithContext(ctx).Model(&types.MedicalDepartment{}).
        Where("tenant_id = ?", tenantID)

    // 关键词搜索：同时匹配 name 和 code（LIKE 模糊查询）
    if keyword != "" {
        like := "%" + keyword + "%"
        baseQuery = baseQuery.Where("name LIKE ? OR code LIKE ?", like, like)
    }

    // 状态筛选：仅当明确指定时才过滤
    if enabled != nil {
        baseQuery = baseQuery.Where("enabled = ?", *enabled)
    }

    // 先 Count 再 Find —— 两次查询
    var total int64
    if err := baseQuery.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 分页计算
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 20 }
    offset := (page - 1) * pageSize

    var depts []*types.MedicalDepartment
    if err := baseQuery.
        Order("created_at DESC").                        // 按创建时间倒序
        Offset(offset).Limit(pageSize).
        Find(&depts).Error; err != nil {
        return nil, 0, err
    }

    return depts, total, nil
}

// Update 全量更新记录
func (r *medicalDepartmentRepository) Update(ctx context.Context, dept *types.MedicalDepartment) error {
    return r.db.WithContext(ctx).Save(dept).Error
}

// Delete 软删除
// 因为 MedicalDepartment.DeletedAt 是 gorm.DeletedAt 类型，
// GORM 会自动执行 UPDATE SET deleted_at=NOW() 而不是 DELETE FROM
func (r *medicalDepartmentRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
    return r.db.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, id).
        Delete(&types.MedicalDepartment{}).Error
}

// ExistsByCode 检查编号是否已存在
// excludeID 用于编辑场景：排除当前正在编辑的记录
func (r *medicalDepartmentRepository) ExistsByCode(
    ctx context.Context, tenantID uint64, code string, excludeID string,
) (bool, error) {
    query := r.db.WithContext(ctx).Model(&types.MedicalDepartment{}).
        Where("tenant_id = ? AND code = ?", tenantID, code)

    if excludeID != "" {
        query = query.Where("id != ?", excludeID)
    }

    var count int64
    if err := query.Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}
```

---

### 4.5 业务逻辑层 (Service)

**文件：`internal/application/service/medical_department.go`**

Service 层包含所有业务规则：参数校验、编号唯一性、租户隔离。

```go
package service

import (
    "context"
    "fmt"
    "github.com/Tencent/WeKnora/internal/types"
    "github.com/Tencent/WeKnora/internal/types/interfaces"
)

// 院区配置（硬编码的字典项，后续可改为系统配置或独立表）
var hospitalAreas = []interfaces.HospitalAreaOption{
    {Label: "中心院区", Value: "中心院区"},
    {Label: "东院区",   Value: "东院区"},
    {Label: "西院区",   Value: "西院区"},
    {Label: "北院区",   Value: "北院区"},
    {Label: "南院区",   Value: "南院区"},
}

type medicalDepartmentService struct {
    repo interfaces.MedicalDepartmentRepository
}

func NewMedicalDepartmentService(repo interfaces.MedicalDepartmentRepository) interfaces.MedicalDepartmentService {
    return &medicalDepartmentService{repo: repo}
}

// getTenantID 从请求上下文中提取租户 ID
// 这是租户隔离的关键 —— 租户 ID 由 Auth 中间件注入到 context 中
func getTenantID(ctx context.Context) (uint64, error) {
    tenantID, ok := types.TenantIDFromContext(ctx)
    if !ok {
        return 0, fmt.Errorf("无法获取租户信息，请确认登录状态")
    }
    return tenantID, nil
}

func (s *medicalDepartmentService) CreateDepartment(
    ctx context.Context,
    req *types.CreateMedicalDepartmentRequest,
    userID string,
) (*types.MedicalDepartmentResponse, error) {
    // 1. 获取租户 ID（每个租户的数据完全隔离）
    tenantID, err := getTenantID(ctx)
    if err != nil { return nil, err }

    // 2. 参数校验
    if req.Name == "" { return nil, fmt.Errorf("科室名称不能为空") }
    if req.Code == "" { return nil, fmt.Errorf("科室编号不能为空") }
    if req.HospitalArea == "" { return nil, fmt.Errorf("院区不能为空") }

    // 3. 编号唯一性校验（同租户下不能重复）
    exists, err := s.repo.ExistsByCode(ctx, tenantID, req.Code, "")
    if err != nil { return nil, fmt.Errorf("检查科室编号失败: %w", err) }
    if exists { return nil, fmt.Errorf("科室编号 %s 已存在", req.Code) }

    // 4. 构建数据模型
    enabled := true
    if req.Enabled != nil { enabled = *req.Enabled }

    dept := &types.MedicalDepartment{
        TenantID:     tenantID,        // 关键：写入正确的租户 ID
        Name:         req.Name,
        Code:         req.Code,
        HospitalArea: req.HospitalArea,
        Enabled:      enabled,
        CreatedBy:    userID,
    }

    // 5. 持久化
    if err := s.repo.Create(ctx, dept); err != nil {
        return nil, fmt.Errorf("创建科室失败: %w", err)
    }

    return s.toResponse(dept, ""), nil
}

// UpdateDepartment 编辑科室
func (s *medicalDepartmentService) UpdateDepartment(
    ctx context.Context, id string, req *types.UpdateMedicalDepartmentRequest,
) (*types.MedicalDepartmentResponse, error) {
    tenantID, err := getTenantID(ctx)
    if err != nil { return nil, err }

    // 1. 先查出原记录
    dept, err := s.repo.GetByID(ctx, tenantID, id)
    if err != nil { return nil, fmt.Errorf("科室不存在: %w", err) }

    // 2. 按需更新字段（nil = 不修改）
    if req.Name != nil {
        if *req.Name == "" { return nil, fmt.Errorf("科室名称不能为空") }
        dept.Name = *req.Name
    }
    if req.Code != nil {
        if *req.Code == "" { return nil, fmt.Errorf("科室编号不能为空") }
        // 唯一性校验时排除自身
        exists, err := s.repo.ExistsByCode(ctx, tenantID, *req.Code, id)
        if err != nil { return nil, fmt.Errorf("检查科室编号失败: %w", err) }
        if exists { return nil, fmt.Errorf("科室编号 %s 已存在", *req.Code) }
        dept.Code = *req.Code
    }
    if req.HospitalArea != nil {
        if *req.HospitalArea == "" { return nil, fmt.Errorf("院区不能为空") }
        dept.HospitalArea = *req.HospitalArea
    }
    if req.Enabled != nil {
        dept.Enabled = *req.Enabled
    }

    // 3. 保存更新
    if err := s.repo.Update(ctx, dept); err != nil {
        return nil, fmt.Errorf("更新科室失败: %w", err)
    }

    return s.toResponse(dept, ""), nil
}

// 其他方法（ListDepartments、DeleteDepartment、ListHospitalAreas）
// 结构类似，此处省略完整代码，详见源文件

// toResponse 将数据库模型转换为 API 响应 DTO
func (s *medicalDepartmentService) toResponse(
    dept *types.MedicalDepartment, createdByName string,
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
```

---

### 4.6 HTTP 处理层 (Handler)

**文件：`internal/handler/medical_department.go`**

Handler 负责：读取 HTTP 请求 → 绑定参数 → 调用 Service → 返回 JSON。

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/Tencent/WeKnora/internal/types"
    "github.com/Tencent/WeKnora/internal/types/interfaces"
)

type MedicalDepartmentHandler struct {
    service interfaces.MedicalDepartmentService
}

func NewMedicalDepartmentHandler(service interfaces.MedicalDepartmentService) *MedicalDepartmentHandler {
    return &MedicalDepartmentHandler{service: service}
}

// ListDepartments 科室列表
func (h *MedicalDepartmentHandler) ListDepartments(c *gin.Context) {
    ctx := c.Request.Context()  // 从 gin.Context 获取标准 context.Context

    var req types.ListMedicalDepartmentRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.Error(errors.NewBadRequestError("查询参数不合法"))
        return
    }

    // 处理 enabled 参数（URL query string 传的是字符串 "true"/"false"）
    if enabledStr := c.Query("enabled"); enabledStr != "" {
        v := enabledStr == "true" || enabledStr == "1"
        req.Enabled = &v
    }

    result, err := h.service.ListDepartments(ctx, &req)
    if err != nil {
        c.Error(err)    // Gin 错误处理中间件会统一格式化为 JSON
        return
    }

    // 统一响应格式：{"success": true, "data": {...}}
    c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// CreateDepartment 新建科室
func (h *MedicalDepartmentHandler) CreateDepartment(c *gin.Context) {
    ctx := c.Request.Context()

    var req types.CreateMedicalDepartmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(errors.NewBadRequestError("请求参数不合法").WithDetails(err.Error()))
        return
    }

    // 从 context 获取当前登录用户 ID（由 Auth 中间件注入）
    userID, _ := types.UserIDFromContext(ctx)

    result, err := h.service.CreateDepartment(ctx, &req, userID)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// UpdateDepartment 编辑科室
func (h *MedicalDepartmentHandler) UpdateDepartment(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    var req types.UpdateMedicalDepartmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(errors.NewBadRequestError("请求参数不合法"))
        return
    }

    result, err := h.service.UpdateDepartment(ctx, id, &req)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// DeleteDepartment 软删除科室
func (h *MedicalDepartmentHandler) DeleteDepartment(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    if err := h.service.DeleteDepartment(ctx, id); err != nil {
        c.Error(err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListHospitalAreas 院区下拉
func (h *MedicalDepartmentHandler) ListHospitalAreas(c *gin.Context) {
    ctx := c.Request.Context()
    areas := h.service.ListHospitalAreas(ctx)
    c.JSON(http.StatusOK, gin.H{"success": true, "data": areas})
}
```

---

### 4.7 路由注册 (Router)

**文件：`internal/router/router.go`（修改）**

添加了三处：

**A. RouterParams 结构体添加字段（第 84 行）**

```go
type RouterParams struct {
    dig.In
    // ... 现有字段 ...
    WikiPageHandler              *handler.WikiPageHandler
    MedicalDepartmentHandler     *handler.MedicalDepartmentHandler  // ← 新增
}
```

**B. NewRouter 函数中调用注册（第 205 行）**

```go
RegisterWikiPageRoutes(v1, params.WikiPageHandler, rbacGuards)
RegisterMedicalDepartmentRoutes(v1, params.MedicalDepartmentHandler, rbacGuards) // ← 新增
RegisterChunkerDebugRoutes(v1, rbacGuards)
```

**C. 路由注册函数（文件末尾新增）**

```go
// RegisterMedicalDepartmentRoutes 注册科室管理路由
// 所有路由都在 /api/v1/medical/ 前缀下
// 权限：Viewer+ 查看，Contributor+ 新建/编辑/删除
func RegisterMedicalDepartmentRoutes(
    r *gin.RouterGroup,
    handler *handler.MedicalDepartmentHandler,
    g *rbacGuards,
) {
    if handler == nil { return }

    medical := r.Group("/medical")
    {
        // g.Viewer()    → 最低角色：观察者（只读）
        // g.Contributor() → 最低角色：贡献者（可写）
        medical.GET("/departments", g.Viewer(), handler.ListDepartments)
        medical.GET("/departments/:id", g.Viewer(), handler.GetDepartment)
        medical.POST("/departments", g.Contributor(), handler.CreateDepartment)
        medical.PUT("/departments/:id", g.Contributor(), handler.UpdateDepartment)
        medical.DELETE("/departments/:id", g.Contributor(), handler.DeleteDepartment)
        medical.GET("/hospital-areas", g.Viewer(), handler.ListHospitalAreas)
    }
}
```

**RBAC 权限中间件说明：**

- `g.Viewer()` — 检查用户角色是否 ≥ Viewer。所有已认证的租户成员都可以查看。
- `g.Contributor()` — 检查用户角色是否 ≥ Contributor。新建/编辑/删除需要此角色。
- 如果 RBAC 未启用（`enable_rbac=false`），中间件会自动放行所有请求。
- 中间件从前置的 Auth 中间件设置的 context 中读取租户 ID 和用户角色。

---

### 4.8 依赖注入 (Container)

**文件：`internal/container/container.go`（修改）**

添加了三行注册，顺序很重要：Repository → Service → Handler。

```go
// Repository 注册（第 167 行）
must(container.Provide(repository.NewMedicalDepartmentRepository))

// Service 注册（第 214 行）
must(container.Provide(service.NewMedicalDepartmentService))

// Handler 注册（第 344 行）
must(container.Provide(handler.NewMedicalDepartmentHandler))
```

`must()` 是一个 helper 函数：如果注册失败（比如类型不匹配），直接 panic 终止启动，避免带错误状态运行。

`dig` 容器会自动解析构造函数参数：当 `NewMedicalDepartmentService` 需要 `MedicalDepartmentRepository` 时，dig 会自动找到已注册的 `NewMedicalDepartmentRepository` 并注入。

---

## 5. 前端实现详解

### 5.1 API 封装层

**文件：`frontend/src/api/medical/department/index.ts`**（新增）

```typescript
// 复用 WeKnora 已有的 HTTP 请求工具
// get/post/put/del 内置了 JWT token 注入、错误处理、base URL 拼接
import { get, post, put, del } from '../../../utils/request'

// ── TypeScript 类型定义 ──
export interface MedicalDepartment {
    id: string
    name: string
    code: string
    hospital_area: string
    enabled: boolean
    created_by: string
    created_by_name: string
    created_at: string
    updated_at: string
}

export interface DepartmentPayload {
    name: string
    code: string
    hospital_area: string
    enabled: boolean
}

export interface HospitalArea {
    label: string
    value: string
}

// ── API 方法 ──
export function listDepartments(params?: {
    keyword?: string; enabled?: string; page?: number; page_size?: number
}) {
    const query = new URLSearchParams()
    if (params?.keyword) query.set('keyword', params.keyword)
    if (params?.enabled !== undefined && params.enabled !== '')
        query.set('enabled', params.enabled)
    if (params?.page) query.set('page', String(params.page))
    if (params?.page_size) query.set('page_size', String(params.page_size))
    const qs = query.toString()
    return get(qs ? `/api/v1/medical/departments?${qs}` : '/api/v1/medical/departments')
}

export function createDepartment(data: DepartmentPayload) {
    return post('/api/v1/medical/departments', data)
}

export function updateDepartment(id: string, data: Partial<DepartmentPayload>) {
    return put(`/api/v1/medical/departments/${id}`, data)
}

export function deleteDepartment(id: string) {
    return del(`/api/v1/medical/departments/${id}`)
}

export function listHospitalAreas() {
    return get('/api/v1/medical/hospital-areas')
}
```

**请求工具说明（`utils/request.ts`）：**
- `get/post/put/del` 基于 axios 封装
- 自动从 `localStorage` 读取 JWT token 并注入 `Authorization: Bearer xxx` 头
- 自动处理错误响应并弹出提示
- 自动拼接 `/api/v1` 基础路径（通过 Vite proxy 转发到后端）

---

### 5.2 科室列表页面

**文件：`frontend/src/views/medical/department/DepartmentList.vue`**（新增）

这是科室管理的核心页面，组织结构：

```
┌──────────────────────────────────────────────────┐
│ 页面标题 + 说明                                    │
├──────────────────────────────────────────────────┤
│ 4 个知识库卡片（科室/症状/疾病/药品）                  │
│ 选中 "科室知识库" 时显示科室列表                      │
├──────────────────────────────────────────────────┤
│ 搜索栏：输入框 + 状态下拉 + 搜索/重置按钮              │
│ 操作栏：新建科室按钮                                 │
├──────────────────────────────────────────────────┤
│ TablePlus 表格：科室名称/编号/院区/状态/创建人/时间/操作 │
│ 分页器                                             │
├──────────────────────────────────────────────────┤
│ DepartmentFormDialog（新建/编辑弹窗）                │
└──────────────────────────────────────────────────┘
```

核心逻辑：

```typescript
// 数据流
fetchData() → listDepartments(API) → tableData 更新 → 表格渲染

// 卡片切换
handleCardClick(key) → activeCard = key → 重新加载对应数据

// 搜索
handleSearch() → 重置分页 → fetchData()

// 新建
openCreateDialog() → dialogMode='create' → 弹窗打开 → 提交 → emit('saved') → fetchData()

// 编辑
openEditDialog(row) → dialogMode='edit' → 弹窗打开（回显数据） → 提交 → emit('saved') → fetchData()

// 启用/停用
toggleEnabled(row) → updateDepartment(id, {enabled: !row.enabled}) → fetchData()

// 删除
handleDelete(id) → deleteDepartment(id) → fetchData()
```

---

### 5.3 科室表单弹窗

**文件：`frontend/src/views/medical/department/DepartmentFormDialog.vue`**（新增）

设计决策：**新建和编辑使用同一个组件**，通过 `mode` prop 区分。

```typescript
// Props
props: {
    modelValue: boolean      // 控制弹窗显隐（v-model）
    mode: 'create' | 'edit'  // 模式
    department?: MedicalDepartment  // 编辑时传入原数据
}

// 核心逻辑
handleConfirm() {
    // 1. TDesign 表单校验
    const valid = await formRef.value?.validate()
    if (valid !== true) return

    // 2. 构建 payload
    const payload = { name, code, hospital_area, enabled }

    // 3. 调用 API
    if (isEdit) {
        await updateDepartment(department.id, payload)
    } else {
        await createDepartment(payload)
    }

    // 4. 关闭弹窗 + 通知父组件刷新
    visible.value = false
    emit('saved')
}
```

**表单校验规则：**
```typescript
const rules = {
    name:          [{ required: true, message: '请输入科室名称' }],
    code:          [{ required: true, message: '请输入科室编号' }],
    hospital_area: [{ required: true, message: '请选择院区' }],
    enabled:       [{ required: true, message: '请选择启用状态' }],
}
```

---

### 5.4 前端路由注册

**文件：`frontend/src/router/index.ts`（修改）**

```typescript
// 在 /platform 的 children 数组中添加
{
    path: "medical/departments",
    name: "medicalDepartments",
    component: () => import("../views/medical/department/DepartmentList.vue"),
    meta: { requiresInit: true, requiresAuth: true }
    // requiresInit: 系统必须已完成初始化（有租户数据）
    // requiresAuth: 用户必须已登录
}
```

`() => import(...)` 是 Vue Router 的**异步懒加载**语法，只有用户真正访问这个路由时才会加载该组件的 JS 代码。

---

### 5.5 菜单注册

**文件：`frontend/src/stores/menu.ts`（修改）**

```typescript
const menuArr = reactive<MenuItem[]>([
    // ... 现有菜单 ...
    { title: '', titleKey: 'menu.organizations', icon: 'organization', path: 'organizations' },
    // ↓ 新增
    { title: '', titleKey: 'menu.medicalKB', icon: 'zhishiku', path: 'medical/departments' },
    // ↑ 新增
    { title: '', titleKey: 'menu.settings', icon: 'setting', path: 'settings' },
])
```

- `titleKey: 'menu.medicalKB'` — 菜单文字从 i18n 文件读取，支持多语言
- `icon: 'zhishiku'` — 复用知识库图标（`assets/img/zhishiku.svg`）
- `path: 'medical/departments'` — 点击导航到 `/platform/medical/departments`

**文件：`frontend/src/components/menu.vue`（修改）**

修改了三个位置：

```typescript
// 1. topMenuItems 筛选 —— 让医疗知识库菜单出现在上半部分
item.path === 'medical/departments'

// 2. bottomMenuItems 排除 —— 不在下半部分（设置/退出）显示
item.path === 'medical/departments'

// 3. 图标激活状态 —— 在医疗科室页面时知识库图标变绿
knowledgeIcon.value = (kbActiveState.isKbActive || route.name === 'medicalDepartments')
    ? 'zhishiku-green.svg' : 'zhishiku.svg'
```

---

### 5.6 国际化

**文件：`frontend/src/i18n/locales/zh-CN.ts`**

```typescript
menu: {
    // ... 现有 ...
    medicalKB: "知识库管理",  // ← 新增
}
```

**文件：`frontend/src/i18n/locales/en-US.ts`**

```typescript
menu: {
    // ... 现有 ...
    medicalKB: 'Medical KB',  // ← 新增
}
```

---

## 6. 遇到问题与解决

### 问题 1：Go 依赖下载极慢/超时

**现象：** 第一次 `go build` 需要下载几百个依赖包，`proxy.golang.org`（Google 服务）在国内访问不稳定。

**解决：** 设置 Go 代理为国内镜像：
```bash
export GOPROXY=https://goproxy.cn,direct
```

---

### 问题 2：编译成功但缺少函数定义

**现象：** `go run cmd/server/main.go` 报错：
```
undefined: runStartupBootstrap
undefined: listenWithRetry
undefined: shutdownSignals
```

**原因：** `go run main.go` 只编译单个文件，不会自动包含同包的其他 `.go` 文件。而 `cmd/server/` 目录下有 `main.go`、`bootstrap.go`、`listen.go`、`signals_unix.go` 等多个文件。

**解决：** 使用 `go run ./cmd/server/` 或 `go build ./cmd/server`，编译整个 package。

---

### 问题 3：macOS 编译的二进制无法在 Docker Linux 容器中运行

**现象：** 将 macOS 上编译的二进制拷贝到 Docker 容器中运行，报错：
```
exec format error
```

**原因：** macOS 编译的二进制是 Mach-O 格式（`arm64`），Docker 容器运行的是 Linux，需要 ELF 格式。

**解决：** 使用交叉编译：
```bash
GOOS=linux GOARCH=arm64 go build -o WeKnora-linux ./cmd/server
```

（但本项目有 CGO 依赖如 DuckDB/sqlite-vec，交叉编译失败，最终改用本地运行。）

---

### 问题 4：数据库地址不通

**现象：** 本地 Go 后端连不上 Postgres，报 `connection timed out`。

**原因：** `docker-compose.yml` 中的 Postgres 只暴露在 Docker 内部网络（`172.19.0.x`），没有映射到宿主机端口。而宿主机本地 `localhost:5432` 是另一个项目的 Postgres。

**解决：** 使用 `docker-compose.dev.yml` 启动开发环境的 Postgres：
```bash
docker compose -f docker-compose.dev.yml up -d postgres
```
dev compose 中 Postgres 映射到宿主机 `5433` 端口（因为 `5432` 被 `industry_postgres` 占用）。

---

### 问题 5：数据库密码认证失败

**现象：** `password authentication failed for user "postgres"`。

**解决：** 通过 `docker compose -f docker-compose.dev.yml up -d` 使用正确的 dev 环境 Postgres（密码与 `.env` 中的 `postgres123!@#` 匹配）。

---

### 问题 6：端口 8080 被 Docker Desktop 占用

**现象：** Go 后端启动时 `bind: address already in use`。

**原因：** Docker Desktop 的 `com.docke` 进程会将已停止容器的 `-p 8080:8080` 端口映射保持一段时间。

**解决：** 杀掉占用端口的进程：
```bash
kill $(lsof -ti:8080)
# 或
sudo lsof -i :8080  # 找到 PID
kill -9 <PID>
```

---

### 问题 7：创建科室成功但 tenant_id 为 0

**现象：** API 创建科室返回成功，但数据库里 `tenant_id=0`，列表查询返回空。

**原因：** 最初在 Service 中使用了 `types.MustTenantIDFromContext(ctx)`，该函数从 `context.Context` 中读取租户 ID。但 Context 中的 tenant_id key 有两种设置方式：
- Auth 中间件通过 `c.Set(types.TenantIDContextKey.String(), tenantID)` 设置到 Gin Context（`c.Keys`）
- 同时通过 `c.Request = c.Request.WithContext(ctx)` 设置到 Request Context

`MustTenantIDFromContext` 从 Request Context 中读取，可能因为中间件执行顺序问题导致读到的是零值（`uint64(0)`），而 `Must` 版本在 key 存在但值为 0 时不会报错。

**解决：** 替换为自定义的 `getTenantID` 函数，使用 `TenantIDFromContext`（返回 `(uint64, bool)`），当 `ok=false` 时明确返回错误。同时确保 Service 在创建时**显式设置** `dept.TenantID = tenantID`（不依赖 GORM 默认值）。

---

### 问题 8：Router 语法错误 — 多余的闭合括号

**现象：** 编译报错 `syntax error: non-declaration statement outside function body`。

**原因：** 用 `sed` 脚本操作 `router.go` 时，在已有的 `RegisterMedicalDepartmentRoutes` 调用上重复添加了多行，导致函数定义外多了一个 `}`。

**解决：** 手动检查并删除多余的行。

---

### 问题 9：前端 Dev Server 进程退出

**现象：** Safari 打开 `localhost:5173` 显示无法连接。

**原因：** `nohup` 启动的前端进程可能因为父 shell 退出而被终止。

**解决：** 重新启动 `npm run dev`。

---

## 7. 完整请求链路

以"新建科室"为例，从前端到数据库的完整链路：

```
1. 用户在 DepartmentList.vue 点击 "新建科室"
   ↓
2. DepartmentFormDialog.vue 打开（mode='create'）
   用户填写表单 → 点击 "新建"
   ↓
3. frontend/src/api/medical/department/index.ts
   createDepartment(payload)
     → POST /api/v1/medical/departments
     → axios 自动注入 Authorization: Bearer <JWT_TOKEN>
   ↓
4. Vite Dev Server proxy（vite.config.ts）
   /api → http://localhost:8080
   ↓
5. Gin Router (router.go:1616)
   g.Contributor() 中间件 → 检查用户角色 ≥ Contributor
   ↓
6. MedicalDepartmentHandler.CreateDepartment (handler/medical_department.go:98)
   - c.ShouldBindJSON → 反序列化请求体
   - types.UserIDFromContext(ctx) → 获取当前用户 ID
   - service.CreateDepartment(ctx, req, userID)
   ↓
7. medicalDepartmentService.CreateDepartment (service/medical_department.go:49)
   - getTenantID(ctx) → 从 Context 提取 tenant_id=10000
   - 校验 name/code/hospital_area 非空
   - repo.ExistsByCode → 检查编号是否已存在
   - 构建 MedicalDepartment{TenantID: 10000, Name: "呼吸科", ...}
   - repo.Create(ctx, dept)
   ↓
8. medicalDepartmentRepository.Create (repository/medical_department.go:23)
   - GORM: INSERT INTO medical_departments (...) VALUES (...)
   - BeforeCreate 钩子 → 生成 UUID 主键
   ↓
9. PostgreSQL (WeKnora-postgres-dev:5433)
   写入 medical_departments 表
   tenant_id=10000, name="呼吸科", code="HX001", enabled=true
   ↓
10. 响应链路逆向返回
    Service → Handler → Gin → HTTP → axios → DepartmentFormDialog
    - 关闭弹窗
    - emit('saved')
    - DepartmentList.fetchData() → 刷新表格
```

---

## 总结

- **新增文件**：10 个（2 迁移 + 6 后端 + 1 前端 API + 2 前端页面）
- **修改文件**：7 个（2 后端注册 + 5 前端配置）
- **API 端点**：6 个（GET list/detail/areas, POST create, PUT update, DELETE delete）
- **数据库表**：1 个（`medical_departments`）
- **核心模式**：完全遵循 WeKnora 现有的三层架构（Handler → Service → Repository）
- **关键设计**：租户隔离、软删除、partial unique index、RBAC 权限、前后端分离
