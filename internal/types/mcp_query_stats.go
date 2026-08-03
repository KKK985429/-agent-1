package types

import "time"

// TenantMCPQueryStats stores aggregate MCP hybrid-search usage for one tenant.
type TenantMCPQueryStats struct {
	TenantID      uint64     `json:"tenant_id" gorm:"primaryKey"`
	QueryCount    int64      `json:"query_count"`
	HitCount      int64      `json:"hit_count"`
	LastQueriedAt *time.Time `json:"last_queried_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (TenantMCPQueryStats) TableName() string {
	return "tenant_mcp_query_stats"
}

// TenantMCPQueryStatsResponse is the tenant-scoped statistics API response.
type TenantMCPQueryStatsResponse struct {
	TenantID      uint64     `json:"tenant_id"`
	QueryCount    int64      `json:"query_count"`
	HitCount      int64      `json:"hit_count"`
	MissCount     int64      `json:"miss_count"`
	HitRate       float64    `json:"hit_rate"`
	LastQueriedAt *time.Time `json:"last_queried_at,omitempty"`
}
