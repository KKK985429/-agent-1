package handler

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mcpStatsKBServiceStub struct {
	interfaces.KnowledgeBaseService
	results []*types.SearchResult
}

func (s *mcpStatsKBServiceStub) GetKnowledgeBaseByID(context.Context, string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: "kb-1", TenantID: 9}, nil
}

func (s *mcpStatsKBServiceStub) HybridSearch(
	context.Context,
	string,
	types.SearchParams,
) ([]*types.SearchResult, error) {
	return s.results, nil
}

type mcpStatsServiceStub struct {
	interfaces.MCPQueryStatsService
	calls    int
	tenantID uint64
	hit      bool
}

func (s *mcpStatsServiceStub) RecordQuery(_ context.Context, tenantID uint64, hit bool) error {
	s.calls++
	s.tenantID = tenantID
	s.hit = hit
	return nil
}

func newMCPSourceTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/v1/knowledge-bases/kb/hybrid-search", nil)
	return c
}

func TestIsTrustedMCPRequest(t *testing.T) {
	t.Setenv(mcpInternalTokenEnv, "server-secret")

	t.Run("ordinary API request", func(t *testing.T) {
		trusted, err := isTrustedMCPRequest(newMCPSourceTestContext())
		require.NoError(t, err)
		require.False(t, trusted)
	})

	t.Run("valid MCP credentials", func(t *testing.T) {
		c := newMCPSourceTestContext()
		c.Request.Header.Set(mcpSourceHeader, "mcp")
		c.Request.Header.Set(mcpInternalTokenHeader, "server-secret")
		trusted, err := isTrustedMCPRequest(c)
		require.NoError(t, err)
		require.True(t, trusted)
	})

	t.Run("invalid MCP credentials", func(t *testing.T) {
		c := newMCPSourceTestContext()
		c.Request.Header.Set(mcpSourceHeader, "mcp")
		c.Request.Header.Set(mcpInternalTokenHeader, "wrong-secret")
		trusted, err := isTrustedMCPRequest(c)
		require.Error(t, err)
		require.False(t, trusted)
	})
}

func TestHybridSearchRecordsTrustedMCPQuery(t *testing.T) {
	t.Setenv(mcpInternalTokenEnv, "server-secret")
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name    string
		results []*types.SearchResult
		hit     bool
	}{
		{name: "hit", results: []*types.SearchResult{{ID: "chunk-1"}}, hit: true},
		{name: "miss", results: []*types.SearchResult{}, hit: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(
				"POST",
				"/api/v1/knowledge-bases/kb-1/hybrid-search",
				bytes.NewBufferString(`{"query_text":"test","match_count":5}`),
			)
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.Header.Set(mcpSourceHeader, "mcp")
			c.Request.Header.Set(mcpInternalTokenHeader, "server-secret")
			c.Set(types.TenantIDContextKey.String(), uint64(9))
			c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

			stats := &mcpStatsServiceStub{}
			h := &KnowledgeBaseHandler{
				service:       &mcpStatsKBServiceStub{results: tc.results},
				mcpQueryStats: stats,
			}
			h.HybridSearch(c)

			require.Equal(t, 200, w.Code)
			require.Equal(t, 1, stats.calls)
			require.Equal(t, uint64(9), stats.tenantID)
			require.Equal(t, tc.hit, stats.hit)
		})
	}
}
