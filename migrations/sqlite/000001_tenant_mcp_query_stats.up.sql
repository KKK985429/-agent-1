CREATE TABLE IF NOT EXISTS tenant_mcp_query_stats (
    tenant_id       INTEGER PRIMARY KEY,
    query_count     INTEGER NOT NULL DEFAULT 0 CHECK (query_count >= 0),
    hit_count       INTEGER NOT NULL DEFAULT 0 CHECK (hit_count >= 0 AND hit_count <= query_count),
    last_queried_at DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
