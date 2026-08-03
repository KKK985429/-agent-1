package repository

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mcpQueryStatsRepository struct {
	db *gorm.DB
}

func NewMCPQueryStatsRepository(db *gorm.DB) interfaces.MCPQueryStatsRepository {
	return &mcpQueryStatsRepository{db: db}
}

// Increment atomically creates or increments the tenant's counters.
func (r *mcpQueryStatsRepository) Increment(ctx context.Context, tenantID uint64, hit bool) error {
	now := time.Now()
	hitIncrement := int64(0)
	if hit {
		hitIncrement = 1
	}

	row := &types.TenantMCPQueryStats{
		TenantID:      tenantID,
		QueryCount:    1,
		HitCount:      hitIncrement,
		LastQueriedAt: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"query_count":     gorm.Expr("tenant_mcp_query_stats.query_count + ?", 1),
			"hit_count":       gorm.Expr("tenant_mcp_query_stats.hit_count + ?", hitIncrement),
			"last_queried_at": now,
			"updated_at":      now,
		}),
	}).Create(row).Error
}

func (r *mcpQueryStatsRepository) Get(ctx context.Context, tenantID uint64) (*types.TenantMCPQueryStats, error) {
	var stats types.TenantMCPQueryStats
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
