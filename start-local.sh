#!/bin/bash
# ============================================================
# WeKnora 本地开发启动脚本
# 用法: ./start-local.sh
# ============================================================

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="/tmp/weknora-logs"
mkdir -p "$LOG_DIR"

echo "========================================="
echo "  WeKnora 本地开发环境启动"
echo "========================================="

# ── 1. 确保 Docker 基础设施运行 ──
echo ""
echo "[1/4] 检查 Docker 基础设施..."

if ! docker ps --format '{{.Names}}' | grep -q "WeKnora-postgres-dev"; then
    echo "  → 启动 PostgreSQL（端口 5433）和 Redis..."
    cd "$PROJECT_DIR"
    docker compose -f docker-compose.dev.yml up -d postgres redis 2>&1
    sleep 3
    echo "  ✅ PostgreSQL 已启动"
else
    echo "  ✅ PostgreSQL 已在运行"
fi

# ── 2. 编译后端（如有修改） ──
echo ""
echo "[2/4] 编译后端..."

export PATH="/opt/homebrew/bin:$PATH"
export GOPROXY=https://goproxy.cn,direct

cd "$PROJECT_DIR"
if [ ! -f WeKnora ] || [ "$PROJECT_DIR/cmd/server" -nt "$PROJECT_DIR/WeKnora" ] || [ "$PROJECT_DIR/internal" -nt "$PROJECT_DIR/WeKnora" ]; then
    echo "  → 源码有更新，重新编译..."
    go build -o WeKnora ./cmd/server
    echo "  ✅ 编译完成"
else
    echo "  ✅ 二进制已是最新，跳过编译"
fi

# ── 3. 启动后端 ──
echo ""
echo "[3/4] 启动后端..."

# 先停掉旧进程
if lsof -ti:8080 > /dev/null 2>&1; then
    echo "  → 停止旧后端进程..."
    kill $(lsof -ti:8080) 2>/dev/null
    sleep 2
fi

DB_DRIVER=postgres \
DB_HOST=localhost \
DB_PORT=5433 \
DB_USER=postgres \
DB_PASSWORD='postgres123!@#' \
DB_NAME=WeKnora \
RETRIEVE_DRIVER=postgres \
STORAGE_TYPE=local \
LOCAL_STORAGE_BASE_DIR=$PROJECT_DIR/data/files \
TENANT_AES_KEY=weknorarag-api-key-secret-secret \
SYSTEM_AES_KEY=weknora-system-aes-key-32bytes!! \
JWT_SECRET=weknora-jwt-secret \
GIN_MODE=debug \
TZ=Asia/Shanghai \
WEKNORA_LANGUAGE=zh-CN \
AUTO_RECOVER_DIRTY=true \
	DOCREADER_ADDR=localhost:50051 \
	DOCREADER_TRANSPORT=grpc \
SSRF_WHITELIST_EXTRA=172.16.1.250,searxng,qdrant,milvus,weaviate,doris-fe \
nohup "$PROJECT_DIR/WeKnora" > "$LOG_DIR/backend.log" 2>&1 &

echo "  → 后端 PID: $!"
echo "  → 日志: $LOG_DIR/backend.log"

# 等待后端就绪
echo "  → 等待后端启动..."
for i in $(seq 1 30); do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "  ✅ 后端已就绪 (http://localhost:8080)"
        break
    fi
    sleep 1
done

# ── 4. 启动前端 ──
echo ""
echo "[4/4] 启动前端..."

if lsof -ti:5173 > /dev/null 2>&1; then
    echo "  → 停止旧前端进程..."
    kill $(lsof -ti:5173) 2>/dev/null
    sleep 1
fi

cd "$PROJECT_DIR/frontend"
nohup npm run dev > "$LOG_DIR/frontend.log" 2>&1 &

echo "  → 前端 PID: $!"
echo "  → 日志: $LOG_DIR/frontend.log"
echo "  → 等待前端启动..."
sleep 3

echo ""
echo "========================================="
echo "  🚀 启动完成！"
echo ""
echo "  前端:  http://localhost:5173"
echo "  后端:  http://localhost:8080"
echo "  Swagger: http://localhost:8080/swagger/index.html"
echo ""
echo "  查看日志:"
echo "    tail -f $LOG_DIR/backend.log"
echo "    tail -f $LOG_DIR/frontend.log"
echo ""
echo "  停止服务:"
echo "    kill \$(lsof -ti:8080) \$(lsof -ti:5173)"
echo "========================================="
