package service

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type mcpQueryStatsService struct {
	repo interfaces.MCPQueryStatsRepository
}

func NewMCPQueryStatsService(repo interfaces.MCPQueryStatsRepository) interfaces.MCPQueryStatsService {
	return &mcpQueryStatsService{repo: repo}
}

func (s *mcpQueryStatsService) RecordQuery(ctx context.Context, tenantID uint64, hit bool) error {
	return s.repo.Increment(ctx, tenantID, hit)
}

func (s *mcpQueryStatsService) GetStats(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantMCPQueryStatsResponse, error) {
	stats, err := s.repo.Get(ctx, tenantID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &types.TenantMCPQueryStatsResponse{TenantID: tenantID}, nil
	}
	if err != nil {
		return nil, err
	}

	hitRate := float64(0)
	if stats.QueryCount > 0 {
		hitRate = float64(stats.HitCount) / float64(stats.QueryCount)
	}

	return &types.TenantMCPQueryStatsResponse{
		TenantID:      tenantID,
		QueryCount:    stats.QueryCount,
		HitCount:      stats.HitCount,
		MissCount:     stats.QueryCount - stats.HitCount,
		HitRate:       hitRate,
		LastQueriedAt: stats.LastQueriedAt,
	}, nil
}
