package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

type stubMCPQueryStatsRepository struct {
	stats *types.TenantMCPQueryStats
	err   error
}

func (r *stubMCPQueryStatsRepository) Increment(context.Context, uint64, bool) error {
	return r.err
}

func (r *stubMCPQueryStatsRepository) Get(context.Context, uint64) (*types.TenantMCPQueryStats, error) {
	return r.stats, r.err
}

func TestMCPQueryStatsServiceGetStats(t *testing.T) {
	now := time.Now()
	svc := NewMCPQueryStatsService(&stubMCPQueryStatsRepository{
		stats: &types.TenantMCPQueryStats{
			TenantID:      8,
			QueryCount:    10,
			HitCount:      7,
			LastQueriedAt: &now,
		},
	})

	stats, err := svc.GetStats(context.Background(), 8)
	require.NoError(t, err)
	require.Equal(t, int64(10), stats.QueryCount)
	require.Equal(t, int64(7), stats.HitCount)
	require.Equal(t, int64(3), stats.MissCount)
	require.InDelta(t, 0.7, stats.HitRate, 0.000001)
}
