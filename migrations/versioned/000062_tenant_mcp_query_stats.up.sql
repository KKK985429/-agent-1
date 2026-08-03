CREATE TABLE IF NOT EXISTS tenant_mcp_query_stats (
    tenant_id       BIGINT PRIMARY KEY,
    query_count     BIGINT NOT NULL DEFAULT 0,
    hit_count       BIGINT NOT NULL DEFAULT 0,
    last_queried_at TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_tenant_mcp_query_count_nonnegative CHECK (query_count >= 0),
    CONSTRAINT chk_tenant_mcp_hit_count_valid CHECK (hit_count >= 0 AND hit_count <= query_count)
);
