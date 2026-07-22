# WeKnora API 接入说明

本文档用于外部系统、外置 Agent、前后端服务接入 WeKnora 进行测试。

## 1. 服务地址

后端 REST API：

```text
http://172.16.1.203:18080/api/v1
```

MCP SSE 服务：

```text
http://172.16.1.203:18082/sse
```

前端后台页面：

```text
http://172.16.1.203:8088
```

## 2. 认证方式

外部系统推荐使用租户 API Key 调用：

```http
X-API-Key: sk-xxxx
Content-Type: application/json
```

示例：

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  http://172.16.1.203:18080/api/v1/knowledge-bases
```

前端用户态也可以使用登录接口返回的 JWT：

```http
Authorization: Bearer <token>
```

一般外部系统、Agent、服务端对接，用 `X-API-Key` 即可。

## 3. 常用 REST API

### 3.1 登录

```http
POST /api/v1/auth/login
```

```bash
curl -s \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }' \
  http://172.16.1.203:18080/api/v1/auth/login
```

返回中会包含：

```json
{
  "token": "JWT",
  "refresh_token": "refresh_token",
  "tenant": {
    "api_key": "sk-xxxx"
  }
}
```

### 3.2 查询知识库列表

```http
GET /api/v1/knowledge-bases
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  http://172.16.1.203:18080/api/v1/knowledge-bases
```

### 3.3 查询知识库详情

```http
GET /api/v1/knowledge-bases/{kb_id}
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  http://172.16.1.203:18080/api/v1/knowledge-bases/知识库ID
```

### 3.4 创建知识库

```http
POST /api/v1/knowledge-bases
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试知识库",
    "description": "用于接口接入测试",
    "type": "document",
    "chunking_config": {
      "chunk_size": 1000,
      "chunk_overlap": 200,
      "separators": ["."],
      "enable_multimodal": true
    }
  }' \
  http://172.16.1.203:18080/api/v1/knowledge-bases
```

### 3.5 上传文件到知识库

```http
POST /api/v1/knowledge-bases/{kb_id}/knowledge/file
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -F "file=@/path/to/file.pdf" \
  -F "enable_multimodel=true" \
  http://172.16.1.203:18080/api/v1/knowledge-bases/知识库ID/knowledge/file
```

### 3.6 查询知识文件列表

```http
GET /api/v1/knowledge-bases/{kb_id}/knowledge?page=1&page_size=20
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  "http://172.16.1.203:18080/api/v1/knowledge-bases/知识库ID/knowledge?page=1&page_size=20"
```

### 3.7 查询知识文件详情

```http
GET /api/v1/knowledge/{knowledge_id}
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  http://172.16.1.203:18080/api/v1/knowledge/知识文件ID
```

### 3.8 文件预览

```http
GET /api/v1/knowledge/{knowledge_id}/preview
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  http://172.16.1.203:18080/api/v1/knowledge/知识文件ID/preview
```

### 3.9 文件下载

```http
GET /api/v1/knowledge/{knowledge_id}/download
```

```bash
curl -L \
  -H "X-API-Key: sk-你的租户key" \
  http://172.16.1.203:18080/api/v1/knowledge/知识文件ID/download
```

### 3.10 混合检索

```http
GET /api/v1/knowledge-bases/{kb_id}/hybrid-search
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -X GET \
  -d '{
    "query_text": "高血压有哪些注意事项",
    "vector_threshold": 0.5,
    "keyword_threshold": 0.3,
    "match_count": 5
  }' \
  http://172.16.1.203:18080/api/v1/knowledge-bases/知识库ID/hybrid-search
```

### 3.11 创建会话

```http
POST /api/v1/sessions
```

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试会话",
    "description": "API 测试"
  }' \
  http://172.16.1.203:18080/api/v1/sessions
```

### 3.12 知识库问答

```http
POST /api/v1/knowledge-chat/{session_id}
```

该接口返回 SSE 流。

```bash
curl -N \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "高血压有哪些注意事项",
    "knowledge_base_ids": ["知识库ID"]
  }' \
  http://172.16.1.203:18080/api/v1/knowledge-chat/会话ID
```

### 3.13 Agent 问答

```http
POST /api/v1/agent-chat/{session_id}
```

该接口返回 SSE 流。

```bash
curl -N \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "帮我总结这个知识库里的内容",
    "agent_enabled": true,
    "agent_id": "agent_id",
    "knowledge_base_ids": ["知识库ID"]
  }' \
  http://172.16.1.203:18080/api/v1/agent-chat/会话ID
```

## 4. FAQ API

FAQ 知识库使用下面这些接口：

```text
GET    /api/v1/knowledge-bases/{kb_id}/faq/entries
POST   /api/v1/knowledge-bases/{kb_id}/faq/entry
POST   /api/v1/knowledge-bases/{kb_id}/faq/entries
GET    /api/v1/knowledge-bases/{kb_id}/faq/entries/{entry_id}
PUT    /api/v1/knowledge-bases/{kb_id}/faq/entries/{entry_id}
DELETE /api/v1/knowledge-bases/{kb_id}/faq/entries
POST   /api/v1/knowledge-bases/{kb_id}/faq/search
GET    /api/v1/faq/import/progress/{task_id}
```

FAQ 搜索示例：

```bash
curl -s \
  -H "X-API-Key: sk-你的租户key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "高血压怎么用药",
    "limit": 5
  }' \
  http://172.16.1.203:18080/api/v1/knowledge-bases/知识库ID/faq/search
```

## 5. MCP SSE 接入

MCP 服务地址：

```text
http://172.16.1.203:18082/sse
```

MCP 服务用于外置 Agent 工具调用。

当前精简后的 MCP 只建议暴露：

```text
get_knowledge_base
hybrid_search
```

### 5.1 MCP 配置示例

```json
{
  "name": "weknora-server",
  "namespaceId": "public",
  "protocol": "mcp-sse",
  "frontProtocol": "mcp-sse",
  "description": "WeKnora 知识库 MCP，仅支持知识库详情查询和混合检索",
  "version": "1.0.0",
  "enabled": true,
  "status": "active",
  "endpointSpecification": "{\"type\":\"url\",\"url\":\"http://172.16.1.203:18082/sse\"}",
  "toolSpecification": "{\"tools\":[{\"name\":\"get_knowledge_base\",\"description\":\"根据知识库ID查询知识库详情\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"kb_id\":{\"type\":\"string\",\"description\":\"知识库ID\"}},\"required\":[\"kb_id\"]}},{\"name\":\"hybrid_search\",\"description\":\"根据知识库ID和用户问题执行混合检索，返回召回结果\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"kb_id\":{\"type\":\"string\",\"description\":\"真实知识库ID，由外部系统或业务配置传入\"},\"query\":{\"type\":\"string\",\"description\":\"检索问题或关键词\"},\"vector_threshold\":{\"type\":\"number\",\"description\":\"向量相似度阈值，默认0.5\"},\"keyword_threshold\":{\"type\":\"number\",\"description\":\"关键词匹配阈值，默认0.3\"},\"match_count\":{\"type\":\"integer\",\"description\":\"返回结果数量，默认5\"}},\"required\":[\"kb_id\",\"query\"]}}]}",
  "resourceSpecification": null
}
```

### 5.2 MCP 工具说明

#### get_knowledge_base

根据知识库 ID 查询知识库详情。

参数：

```json
{
  "kb_id": "知识库ID"
}
```

底层对应 REST API：

```http
GET /api/v1/knowledge-bases/{kb_id}
```

#### hybrid_search

根据知识库 ID 和用户问题执行混合检索，返回召回结果。

参数：

```json
{
  "kb_id": "知识库ID",
  "query": "用户问题",
  "vector_threshold": 0.5,
  "keyword_threshold": 0.3,
  "match_count": 5
}
```

底层对应 REST API：

```http
GET /api/v1/knowledge-bases/{kb_id}/hybrid-search
```

## 6. 全量 API 模块

WeKnora API 按功能分为以下模块：

| 模块 | 说明 | 典型接口 |
| --- | --- | --- |
| 认证 | 注册、登录、JWT、OIDC | `/auth/register`、`/auth/login`、`/auth/me` |
| 租户 | 租户、成员、邀请、审计 | `/tenants`、`/tenants/{id}/members` |
| 知识库 | 创建、列表、详情、更新、删除、复制 | `/knowledge-bases`、`/knowledge-bases/{id}` |
| 知识文件 | 上传、URL 导入、列表、详情、下载、预览 | `/knowledge-bases/{id}/knowledge/file`、`/knowledge/{id}` |
| 检索 | 混合搜索、知识搜索 | `/knowledge-bases/{id}/hybrid-search`、`/knowledge-search` |
| FAQ | FAQ 列表、新增、导入、搜索、删除 | `/knowledge-bases/{id}/faq/entries`、`/knowledge-bases/{id}/faq/search` |
| 会话 | 创建会话、查会话、删会话 | `/sessions`、`/sessions/{id}` |
| 聊天 | 知识库问答、Agent 问答 | `/knowledge-chat/{session_id}`、`/agent-chat/{session_id}` |
| Agent | 创建、列表、详情、更新、复制、删除 | `/agents`、`/agents/{id}` |
| 模型 | 模型配置、模型列表、模型凭证 | `/models`、`/models/{id}`、`/models/providers` |
| 标签 | 知识库标签管理 | `/knowledge-bases/{id}/tags` |
| 分块 | Chunk 列表、删除 | `/knowledge/{id}/chunks` |
| MCP 服务 | MCP 服务配置管理 | `/mcp-services` |
| 网络搜索 | Web Search Provider 管理 | `/web-search/providers` |
| 向量库 | 向量存储配置 | `/vector-stores` |
| 组织 | 组织、成员、共享资源 | `/organizations` |
| Skills | Agent 技能列表 | `/skills` |
| 系统 | 系统配置、解析引擎、平台设置 | `/system/*` |
| 初始化 | 初始化模型、Ollama、知识库配置 | `/initialization/*` |

## 7. 接口文档位置

仓库内 Markdown 文档：

```text
docs/api/README.md
docs/api/*.md
```

Swagger 文件：

```text
docs/swagger.json
docs/swagger.yaml
```

如果服务不是 release 模式，可以访问 Swagger UI：

```text
http://172.16.1.203:18080/swagger/index.html
```

生产环境通常是 `GIN_MODE=release`，Swagger UI 可能关闭。这种情况下以仓库内 `docs/api`、`docs/swagger.yaml`、`docs/swagger.json` 为准。

## 8. 测试建议

建议按下面顺序测试：

1. 用 `X-API-Key` 调 `/knowledge-bases`，确认认证可用。
2. 从知识库列表里拿一个 `kb_id`。
3. 调 `/knowledge-bases/{kb_id}`，确认知识库详情可查。
4. 上传一个测试文件到 `/knowledge-bases/{kb_id}/knowledge/file`。
5. 调 `/knowledge-bases/{kb_id}/knowledge`，确认文件列表可查。
6. 文件解析完成后，调 `/knowledge-bases/{kb_id}/hybrid-search` 测检索。
7. 如需问答，先创建 session，再调 `/knowledge-chat/{session_id}` 或 `/agent-chat/{session_id}`。

