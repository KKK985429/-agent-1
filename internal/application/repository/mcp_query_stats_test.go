package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMCPQueryStatsTestRepository(t *testing.T) (*mcpQueryStatsRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:mcp-query-stats-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantMCPQueryStats{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	return &mcpQueryStatsRepository{db: db}, db
}

func TestMCPQueryStatsRepositoryIncrement(t *testing.T) {
	repo, _ := newMCPQueryStatsTestRepository(t)
	ctx := context.Background()

	require.NoError(t, repo.Increment(ctx, 42, false))
	require.NoError(t, repo.Increment(ctx, 42, true))

	stats, err := repo.Get(ctx, 42)
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.QueryCount)
	require.Equal(t, int64(1), stats.HitCount)
	require.NotNil(t, stats.LastQueriedAt)
}

func TestMCPQueryStatsRepositoryIncrementConcurrent(t *testing.T) {
	repo, _ := newMCPQueryStatsTestRepository(t)
	ctx := context.Background()

	const calls = 20
	var wg sync.WaitGroup
	errCh := make(chan error, calls)
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func(hit bool) {
			defer wg.Done()
			errCh <- repo.Increment(ctx, 7, hit)
		}(i%2 == 0)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	stats, err := repo.Get(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, int64(calls), stats.QueryCount)
	require.Equal(t, int64(calls/2), stats.HitCount)
}
