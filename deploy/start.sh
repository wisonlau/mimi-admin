#!/bin/bash
# mimi-admin 一键启动脚本
# 支持按需启动特定服务或全部启动
#
# 用法:
#   bash deploy/start.sh                    # 启动所有服务
#   bash deploy/start.sh admin-core         # 启动 admin-core 的 api + rpc
#   bash deploy/start.sh admin-core api     # 仅启动 admin-core 的 api
#   bash deploy/start.sh admin-core rpc     # 仅启动 admin-core 的 rpc
#   bash deploy/start.sh product api        # 仅启动 product 的 api
#   bash deploy/start.sh order rpc          # 仅启动 order 的 rpc
#   bash deploy/start.sh gateway            # 仅启动网关
#
# 使用 Ctrl+C 停止所有已启动的服务


ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE_DIR="$ROOT_DIR/service"

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 记录 PID
PIDS=()

cleanup() {
	echo -e "
${YELLOW}正在停止所有服务...${NC}"
	for pid in "${PIDS[@]}"; do
		kill "$pid" 2>/dev/null || true
	done
	wait
	echo -e "${GREEN}所有服务已停止${NC}"
	exit 0
}

trap cleanup SIGINT SIGTERM

start_service() {
	local name="$1"
	local dir="$2"
	local cmd="$3"
	local port="$4"

	# 清理旧进程
	local old_pids
	old_pids=$(lsof -ti ":$port" -P 2>/dev/null || true)
	if [ -n "$old_pids" ]; then
		echo -e "  ${YELLOW}端口 $port 被旧进程占用,正在清理...${NC}"
		kill $old_pids 2>/dev/null
		# 等待端口释放（TCP TIME_WAIT）
		for j in $(seq 1 10); do
			if ! lsof -ti ":$port" -P 2>/dev/null | grep -q . 2>/dev/null; then
				break
			fi
			sleep 0.5
		done
	fi

	echo -e "${YELLOW}正在启动 $name (${port})...${NC}"

	cd "$dir"
	go mod tidy 2>/dev/null

	eval "$cmd" > "/tmp/mimi-$name.log" 2>&1 &
	local pid=$!
	PIDS+=("$pid")

	for i in $(seq 1 60); do
		if lsof -i ":$port" -P 2>/dev/null | grep -q LISTEN 2>/dev/null; then
			echo -e "  ${GREEN}✓ $name 已启动 (pid=$pid, port=$port)${NC}"
			return 0
		fi
		sleep 0.5
	done

	echo -e "  ${RED}✗ $name 启动超时，请检查日志: /tmp/mimi-$name.log${NC}"
	return 1
}

# ========== 服务定义 ==========

start_admin_core_rpc()   { start_service "admin-core-rpc"   "$SERVICE_DIR/admin-core/rpc" "go run rpc.go -f etc/admin-core-rpc.yaml" "8802"; }
start_admin_core_api()   { start_service "admin-core-api"   "$SERVICE_DIR/admin-core/api" "go run api.go -f etc/admin-core-api.yaml" "8801"; }
start_product_rpc()      { start_service "product-rpc"      "$SERVICE_DIR/product/rpc"     "go run rpc.go -f etc/rpc.yaml" "8812"; }
start_product_api()      { start_service "product-api"      "$SERVICE_DIR/product/api"     "go run api.go -f etc/api-api.yaml" "8811"; }
start_order_rpc()        { start_service "order-rpc"        "$SERVICE_DIR/order/rpc"       "go run rpc.go -f etc/order-rpc.yaml" "8822"; }
start_order_api()        { start_service "order-api"        "$SERVICE_DIR/order/api"       "go run api.go -f etc/order-api.yaml" "8821"; }
start_gateway()          { start_service "gateway"           "$SERVICE_DIR/gateway"         "go run gateway.go -f etc/gateway.yaml" "8888"; }

start_all() {
	start_admin_core_rpc
	start_admin_core_api
	start_product_rpc
	start_product_api
	start_order_rpc
	start_order_api
	start_gateway
}

# ========== 服务名 -> 函数映射 ==========

start_service_group() {
	local service="$1"

	case "$service" in
	admin-core)
		start_admin_core_rpc
		start_admin_core_api
		;;
	product)
		start_product_rpc
		start_product_api
		;;
	order)
		start_order_rpc
		start_order_api
		;;
	gateway)
		start_gateway
		;;
	*)
		echo -e "${RED}未知服务: $service${NC}"
		echo "可用服务: admin-core, product, order, gateway"
		exit 1
		;;
	esac
}

start_service_single() {
	local service="$1"
	local mode="$2"

	case "$service:$mode" in
	admin-core:rpc) start_admin_core_rpc ;;
	admin-core:api) start_admin_core_api ;;
	product:rpc)    start_product_rpc    ;;
	product:api)    start_product_api    ;;
	order:rpc)      start_order_rpc      ;;
	order:api)      start_order_api      ;;
	gateway:api)    start_gateway        ;;
	gateway:rpc)
		echo -e "${RED}gateway 没有 rpc 服务${NC}"
		exit 1
		;;
	*)
		echo -e "${RED}未知组合: $service $mode${NC}"
		echo "可用服务: admin-core, product, order, gateway"
		echo "可用模式: api, rpc"
		exit 1
		;;
	esac
}

# ========== 主逻辑 ==========

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  mimi-admin 启动脚本${NC}"
echo -e "${GREEN}========================================${NC}"

case $# in
	0)
		start_all
		;;
	1)
		start_service_group "$1"
		;;
	2)
		start_service_single "$1" "$2"
		;;
	*)
		echo "用法:"
		echo "  bash deploy/start.sh                        # 启动所有服务"
		echo "  bash deploy/start.sh <service>              # 启动某个服务的 api+rpc"
		echo "  bash deploy/start.sh <service> api|rpc      # 启动某个服务的指定模式"
		echo ""
		echo "服务: admin-core, product, order, gateway"
		exit 1
		;;
esac

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  已启动 ${#PIDS[@]} 个服务${NC}"
echo -e "${GREEN}  网关地址: http://localhost:8888${NC}"
echo -e "${GREEN}  按 Ctrl+C 停止所有服务${NC}"
echo -e "${GREEN}========================================${NC}"

wait
