package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// MCPQueryStatsRepository persists tenant-level MCP query counters.
type MCPQueryStatsRepository interface {
	Increment(ctx context.Context, tenantID uint64, hit bool) error
	Get(ctx context.Context, tenantID uint64) (*types.TenantMCPQueryStats, error)
}

// MCPQueryStatsService records and reads tenant-level MCP query counters.
type MCPQueryStatsService interface {
	RecordQuery(ctx context.Context, tenantID uint64, hit bool) error
	GetStats(ctx context.Context, tenantID uint64) (*types.TenantMCPQueryStatsResponse, error)
}
