# mimi-admin

go-zero 微服务框架项目，提供高性能、可扩展的后台管理解决方案。

## 架构

```
                         ┌─────────────────────┐
                         │     API 网关         │
                         │   (localhost:8888)   │
                         └──────┬──────┬───────┘
                    ┌───────────┘      └───────────┐
               ┌────┴────┐                   ┌────┴────┐
               │ HTTP 转  │                   │ HTTP 转  │
               │ HTTP    │                   │ gRPC    │
               └────┬────┘                   └────┬────┘
          ┌─────────┼──────────┐        ┌─────────┼──────────┐
     ┌────┴───┐┌───┴────┐┌────┴───┐  ┌─┴───┐┌───┴───┐┌────┴──┐
     │admin-  ││product ││ order  │  │admin-││product││ order │
     │core-api││-api    ││-api    │  │core- ││-rpc   ││-rpc   │
     │:8801   ││:8811   ││:8821   │  │rpc   ││:8812  ││:8822  │
     └────────┘└────────┘└────────┘  │:8802  ││       ││       │
                                     └──────┘└───────┘└───────┘
                                            etcd
                                      (127.0.0.1:2379)
```

## 目录结构

```
mimi-admin/
├── service/
│   ├── gateway/                    # API 网关 (Port: 8888)
│   │   ├── gateway.go              # 网关入口
│   │   ├── etc/gateway.yaml        # 网关配置
│   │   └── internal/
│   │       ├── config/config.go    # 网关配置结构体
│   │       └── middleware/          # 中间件
│   │           ├── cors/           #   跨域中间件
│   │           ├── hellologging/   #   请求日志中间件
│   │           ├── tokenlimiter/    #   限流器中间件
│   │           └── retry/          #   重试中间件
│   ├── admin-core/                 # 后台基础服务
│   │   ├── api/                    # HTTP 服务 (:8801)
│   │   │   ├── api.go
│   │   │   └── internal/
│   │   │       ├── config/config.go
│   │   │       ├── handler/        # 路由处理器
│   │   │       ├── logic/          # 业务逻辑
│   │   │       ├── svc/            # 服务上下文
│   │   │       └── types/types.go
│   │   └── rpc/                    # gRPC 服务 (:8802)
│   │       ├── rpc.go
│   │       └── internal/
│   │           ├── config/config.go
│   │           ├── logic/          # 业务逻辑
│   │           ├── server/         # gRPC 服务注册
│   │           └── svc/
│   ├── product/                    # product 服务
│   │   ├── api/                    # HTTP 服务 (:8811)
│   │   └── rpc/                    # gRPC 服务 (:8812)
│   └── order/                      # order 服务
│       ├── api/                    # HTTP 服务 (:8821)
│       └── rpc/                    # gRPC 服务 (:8822)
├── common/                         # 公共工具
├── deploy/
│   ├── start.sh                    # 一键启动脚本
│   └── k8s/
└── README.md
```

## 端口汇总

| 服务 | HTTP 端口 | gRPC 端口 |
|------|-----------|-----------|
| admin-core | 8801 | 8802 |
| product | 8811 | 8812 |
| order | 8821 | 8822 |
| API 网关 | 8888 | - |

## 技术栈

- **框架**: [go-zero](https://github.com/zeromicro/go-zero) v1.10.2
- **服务发现**: etcd (127.0.0.1:2379)
- **协议**: HTTP (REST) + gRPC
- **数据库**: MySQL (各服务独立配置)
- **缓存**: Redis (各服务 + 网关限流器)

## 服务注册

所有服务启动时自动注册到 etcd：

| 服务 | etcd Key | 注册方式 |
|------|----------|----------|
| admin-core-api | `admin-core.api` | ✅ 手动注册（`discov.NewPublisher`） |
| admin-core-rpc | `admin-core.rpc` | ✅ 自动注册（`zrpc` 内置） |
| product-api | `product.api` | ✅ 手动注册 |
| product-rpc | `product.rpc` | ✅ 自动注册 |
| order-api | `order.api` | ✅ 手动注册 |
| order-rpc | `order.rpc` | ✅ 自动注册 |
| gateway | `gateway` | ✅ 手动注册 |

## 接口列表

### 网关入口（`localhost:8888`）

网关提供 HTTP → HTTP 和 HTTP → gRPC 两种转发模式。

#### admin-core

| 方法 | 路径 | 转发类型 | 说明 |
|------|------|----------|------|
| GET | `/api/admin-core/ping` | HTTP → HTTP | 健康检查 |
| POST | `/api/admin-core/hello` | HTTP → HTTP | 打招呼，`Body: {"name":"xxx"}` |
| POST | `/api/admin-core/test_retry` | HTTP → HTTP | 50% 概率返回 503，测试重试 |
| GET | `/api/admin-core/hi/:name` | HTTP → HTTP | 单段路径，如 `/hi/a`、`/hi/b` |
| GET | `/api/admin-core/hi/f/ff` | HTTP → HTTP | 多段路径 |
| GET | `/grpc/admin-core/ping` | HTTP → gRPC | 健康检查 |
| POST | `/grpc/admin-core/hello` | HTTP → gRPC | 打招呼，`Body: {"name":"xxx"}` |
| POST | `/grpc/admin-core/testerr` | HTTP → gRPC | 测试错误（Trailers 含 `code-bin` + `message-bin`） |

#### product

| 方法 | 路径 | 转发类型 | 说明 |
|------|------|----------|------|
| GET | `/api/product/ping` | HTTP → HTTP | 健康检查 |
| POST | `/api/product/hello` | HTTP → HTTP | 打招呼，`Body: {"name":"xxx"}` |
| GET | `/grpc/product/ping` | HTTP → gRPC | 健康检查 |
| POST | `/grpc/product/hello` | HTTP → gRPC | 打招呼，`Body: {"name":"xxx"}` |

#### order

| 方法 | 路径 | 转发类型 | 说明 |
|------|------|----------|------|
| GET | `/api/order/ping` | HTTP → HTTP | 健康检查 |
| POST | `/api/order/hello` | HTTP → HTTP | 打招呼，`Body: {"name":"xxx"}` |
| GET | `/grpc/order/ping` | HTTP → gRPC | 健康检查 |
| POST | `/grpc/order/hello` | HTTP → gRPC | 打招呼，`Body: {"name":"xxx"}` |

### 直接服务入口（跳过网关）

#### admin-core-api（`localhost:8801`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin-core/ping` | 健康检查 |
| POST | `/api/admin-core/hello` | 打招呼，`Body: {"name":"xxx"}` |
| POST | `/api/admin-core/test_retry` | 50% 概率返回 503 |
| GET | `/api/admin-core/hi/:name` | 动态路径参数，如 `/hi/a` |
| GET | `/api/admin-core/hi/f/ff` | 多段路径 |

#### admin-core-rpc（`localhost:8802`）

| 方法 | 参数 | 说明 |
|------|------|------|
| `rpc.Rpc/Ping` | `{"ping":"xxx"}` | 健康检查 |
| `rpc.Rpc/Hello` | `{"name":"xxx"}` | 打招呼 |
| `rpc.Rpc/Testerr` | `{"ping":"xxx"}` | 测试错误返回（Trailers 含 `code-bin` + `message-bin`） |

#### product-api（`localhost:8811`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/product/ping` | 健康检查 |
| POST | `/api/product/hello` | 打招呼，`Body: {"name":"xxx"}` |

#### product-rpc（`localhost:8812`）

| 方法 | 参数 | 说明 |
|------|------|------|
| `rpc.Rpc/Ping` | `{"ping":"xxx"}` | 健康检查 |
| `rpc.Rpc/Hello` | `{"name":"xxx"}` | 打招呼 |

#### order-api（`localhost:8821`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/order/ping` | 健康检查 |
| POST | `/api/order/hello` | 打招呼，`Body: {"name":"xxx"}` |

#### order-rpc（`localhost:8822`）

| 方法 | 参数 | 说明 |
|------|------|------|
| `rpc.Rpc/Ping` | `{"ping":"xxx"}` | 健康检查 |
| `rpc.Rpc/Hello` | `{"name":"xxx"}` | 打招呼 |

## 网关配置

### 上游配置方式

支持三种上游配置，分别对应不同发现方式：

#### etcd 发现（HTTP）

```yaml
- Name: admin-core-http
    Http:
      Timeout: 5s
      Etcd:
        Hosts:
          - 127.0.0.1:2379
        Key: admin-core.api
    Mappings:
      - Method: get
        Path: /api/admin-core/ping
```

#### etcd 发现（gRPC）

```yaml
- Name: admin-core-grpc
    Grpc:
      Timeout: 5000           # gRPC 超时为毫秒
      Etcd:
        Hosts:
          - 127.0.0.1:2379
        Key: admin-core.rpc
    ProtoSets:
      - admin-core.pb
    Mappings:
      - Method: get
        Path: /grpc/admin-core/ping
        RpcPath: rpc.Rpc/Ping
```

#### 硬编码地址

```yaml
- Name: order-http
    Http:
      Target: localhost:8821
      Timeout: 5s
```

### 超时时间

| 类型 | 配置格式 | 支持写法 |
|------|----------|----------|
| HTTP | 字符串 | `5s`、`500ms`、`1.5s`、`3000`（毫秒） |
| gRPC | 数字（毫秒） | `5000`（go-zero 原生限制） |

### 路由映射

HTTP → HTTP 转发：`Path` 路径直接透传

```yaml
Mappings:
  - Method: get
    Path: /api/admin-core/ping   # 客户端访问路径
```

HTTP → gRPC 转发：`Path` 映射到 `RpcPath`

```yaml
Mappings:
  - Method: get
    Path: /grpc/admin-core/ping        # 客户端访问路径
    RpcPath: rpc.Rpc/Ping              # 映射的 gRPC 方法
```

### 路由中间件

支持按路由粒度和上游粒度配置中间件：

```yaml
# 上游级：对该上游所有路由生效
- Name: admin-core-http
    Middlewares:
      - TokenLimiter: ...
    Mappings:
      - Method: get
        Path: /api/admin-core/ping
        # 未指定，继承上游配置

# 路由级：仅对该路由生效，覆盖上游级
    Mappings:
      - Method: get
        Path: /api/admin-core/ping
        Middlewares:
          - TokenLimiter: ...
```

## 中间件

### 执行顺序（洋葱模型）

```
CORS → HelloLogging → TokenLimiter → Retry → Backend
```

| 中间件 | 层 | 职责 |
|--------|-----|------|
| CORS | 最外层 | 跨域头、OPTIONS 预检 |
| HelloLogging | 二 | hello in/out 请求日志 |
| TokenLimiter | 三 | 请求限流 |
| Retry | 最内层 | 失败重试 |

### CORS 跨域

```yaml
- Cors:
    allowCredentials: true
    allowHeaders:
      - "Content-Type, Authorization"
    allowOrigins:
      - ".google.com"
    allowMethods:
      - "GET"
      - "POST"
      - "OPTIONS"
```

- 支持 `allowHeaders` 逗号分隔字符串或数组
- 支持 `allowOrigins` 域名后缀通配（如 `.google.com` 匹配 `xxx.google.com`）
- 自动为有 CORS 配置的路由添加 `OPTIONS` 预检路由

### HelloLogging 请求日志

```yaml
- HelloLogging:        # 无需额外配置，存在即开启
```

输出：

```
hello in:  GET /api/admin-core/ping
hello out: GET /api/admin-core/ping → 200
```

### TokenLimiter 限流器

```yaml
- TokenLimiter:
    Redis:
      Host: 127.0.0.1:6379
      Pass: ""
    Type: path          # path | node | ip
    Rate: 100           # 每秒允许请求数
    Burst: 200          # 最大突发请求数
```

限流粒度：
- `path` — 按路径独立计数
- `node` — 按节点共享计数
- `ip` — 按客户端 IP 计数

### Retry 重试器

```yaml
- Retry:
    attempts: 3
    perTryTimeout: 0.1s
    conditions:
      - byStatusCode: '502-504'
      - byHeader:
          name: 'Grpc-Status'
          value: '14'
```

条件类型：
- `byStatusCode` — 匹配状态码范围（如 `502-504`、`500`、`5xx`）
- `byHeader` — 匹配响应头

## 启动方式

### 前置条件

```bash
# etcd
brew install etcd
brew services start etcd

# 或手动启动
etcd
```

### 一键启动

```bash
bash deploy/start.sh
```

按 `Ctrl+C` 停止所有服务。

### 逐个启动

```bash
# 启动 admin-core
cd service/admin-core/rpc && go run rpc.go -f etc/admin-core-rpc.yaml
cd service/admin-core/api && go run api.go -f etc/admin-core-api.yaml

# 启动 product
cd service/product/rpc && go run rpc.go -f etc/rpc.yaml
cd service/product/api && go run api.go -f etc/api-api.yaml

# 启动 order
cd service/order/rpc && go run rpc.go -f etc/order-rpc.yaml
cd service/order/api && go run api.go -f etc/order-api.yaml

# 启动网关
cd service/gateway && go run gateway.go -f etc/gateway.yaml
```

## 测试

```bash
# === HTTP 转发 ===
curl http://localhost:8888/api/admin-core/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/api/admin-core/hello
curl -X POST http://localhost:8888/api/admin-core/test_retry

curl http://localhost:8888/api/product/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/api/product/hello

curl http://localhost:8888/api/order/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/api/order/hello

# === gRPC 转发 ===
curl http://localhost:8888/grpc/admin-core/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/grpc/admin-core/hello
curl -X POST -H "Content-Type: application/json" \
  -d '{"ping":"test"}' http://localhost:8888/grpc/admin-core/testerr

curl http://localhost:8888/grpc/product/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/grpc/product/hello

curl http://localhost:8888/grpc/order/ping
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"world"}' http://localhost:8888/grpc/order/hello

# === 通配路径 ===
curl http://localhost:8888/api/admin-core/hi/a
curl http://localhost:8888/api/admin-core/hi/f/ff

# === grpcurl 直调 RPC ===
grpcurl -plaintext -d '{"ping":"test"}' localhost:8802 rpc.Rpc/Testerr
grpcurl -v -plaintext -d '{"ping":"test"}' localhost:8802 rpc.Rpc/Testerr  # 查看 Trailers

# === 检查 etcd 注册 ===
curl -s http://127.0.0.1:2379/v3/kv/range \
  -X POST -d '{"key":"AA==","range_end":"AA=="}' \
  -H "Content-Type: application/json" | \
  python3 -c "
import sys,json,base64
data = json.load(sys.stdin)
for kv in data.get('kvs', []):
    key = base64.b64decode(kv['key']).decode()
    val = base64.b64decode(kv['value']).decode()
    print(f'  {key}  →  {val}')
"
```

## 开发：添加新中间件

1. 在 `service/gateway/internal/middleware/` 下新建目录
2. 创建 `config.go`，定义 `MiddlewareName`、配置结构体、`FromEntries()`
3. 创建 `xxx.go`，实现 `Middleware()` 函数
4. 在 `gateway.go` 中引入并注册

```go
// gateway.go
import "gateway/internal/middleware/myplugin"

myMw := myplugin.Middleware(upstreamConfigs)

gw := gateway.MustNewServer(
    gwConf,
    gateway.WithMiddleware(
        corsMiddleware,
        helloLoggingMiddleware,
        rateLimitMiddleware,
        retryMiddleware,
        myMw,
    ),
)
```

## 协议说明

| 转发模式 | 入口 | 出口 | 说明 |
|----------|------|------|------|
| HTTP → HTTP | REST API | REST API | 路径透传，适合现有 HTTP 服务 |
| HTTP → gRPC | REST API | gRPC | 自动 JSON ↔ protobuf 转换 |
| gRPC → gRPC | ❌ | ❌ | 网关基于 HTTP 协议，不支持 gRPC 入口 |