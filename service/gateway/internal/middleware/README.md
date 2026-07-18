# Middleware 中间件开发规范

## 目录结构

每个中间件一个独立目录，命名采用小写英文：

```
internal/middleware/
├── middleware.go            # 包声明
├── cors/                    # 跨域中间件
│   ├── config.go            # 配置类型 + MiddlewareName + FromEntries
│   └── cors.go              # 中间件实现
├── hellologging/            # 请求日志中间件
│   ├── config.go
│   └── hellologging.go
├── TokenLimiter/             # 限流器中间件
│   ├── config.go
│   └── TokenLimiter.go
└── retry/                   # 重试中间件
    ├── config.go
    └── retry.go
```

## 规范要求

### 1. 每个中间件一个目录

目录名即中间件名（小写），所有代码放在该目录下。

### 2. 必须定义 `MiddlewareName`

不管用不用，必须定义。其值对应 `gateway.yaml` 中 `Middlewares` 数组元素里的 key 名称：

```go
// config.go
package cors

const MiddlewareName = "Cors"
```

```yaml
# gateway.yaml
Middlewares:
  - Cors:   # ← MiddlewareName 的值
```

### 3. 对外统一暴露 `Middleware()` 函数

每个中间件对外暴露 `Middleware()`，返回 `rest.Middleware`：

```go
func Middleware(upstreams []TokenLimiter.UpstreamConfig) rest.Middleware {
    // ... 预处理上游配置
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // 前置处理
            next(w, r)
            // 后置处理
        }
    }
}
```

### 4. 配置解析 `FromEntries()`

每个中间件实现 `FromEntries(entries TokenLimiter.MiddlewaresConf) *XXXConf`，从 `Middlewares` 数组中提取自己的配置：

```go
func FromEntries(entries TokenLimiter.MiddlewaresConf) *CorsConf {
    for _, m := range entries {
        raw, ok := m[MiddlewareName]
        if !ok || raw == nil {
            continue
        }
        b, _ := json.Marshal(raw)
        var cfg CorsConf
        json.Unmarshal(b, &cfg)
        return &cfg
    }
    return nil
}
```

### 5. 构建函数统一命名 `BuildXxx`

- `BuildLimiters` — 构建限流器
- `BuildRouteSet` — 构建日志路由集合
- `BuildRouteConfigs` — 构建 CORS 路由配置
- `BuildRetryConfigs` — 构建重试路由配置

### 6. 配置文件放在各自目录

每个中间件的配置类型和常量放在该目录的 `config.go` 中。

### 7. 洋葱模型执行顺序

中间件在网关中按以下顺序组合（由外到内）：

```go
// gateway.go
gw := gateway.MustNewServer(
    gwConf,
    gateway.WithMiddleware(
        corsMiddleware,         // ① 最外层：跨域
        helloLoggingMiddleware, // ② 请求日志
        rateLimitMiddleware,    // ③ 限流
        retryMiddleware,        // ④ 最内层：重试
    ),
)
```

| 中间件 | 层 | 职责 |
|--------|-----|------|
| **CORS** | 最外层 | 跨域头处理、OPTIONS 预检 |
| **HelloLogging** | 二 | hello in/out 请求日志 |
| **TokenLimit** | 三 | 请求限流（只计首次请求） |
| **Retry** | 最内层 | 失败重试（重试时不重复计数） |

## 中间件详情

### CORS 跨域中间件

配置项：

```yaml
- Cors:
    allowCredentials: true              # 是否允许携带凭证
    allowHeaders:
      - "Content-Type, Authorization"   # 支持字符串或数组格式
    allowOrigins:
      - ".google.com"                   # 支持域名后缀通配
    allowMethods:
      - "GET"
      - "POST"
```

特性：
- 支持 `allowHeaders` 使用逗号分隔字符串或数组
- 支持 `allowOrigins` 通配符匹配（如 `.google.com` 匹配 `xxx.google.com`）
- 自动为有 CORS 配置的路由添加 `OPTIONS` 预检路由

### HelloLogging 请求日志中间件

配置项：

```yaml
- HelloLogging:    # 无需额外配置，存在即开启
```

输出：

```
hello in:  GET /api/admin-core/ping
hello out: GET /api/admin-core/ping → 200
```

### TokenLimit 限流器中间件

配置项：

```yaml
- TokenLimiter:
    Redis:
      Host: 127.0.0.1:6379
      Pass: ""
    Type: path    # 限流粒度：path | node | ip
    Rate: 100     # 每秒允许请求数
    Burst: 200    # 最大突发请求数
```

限流类型：
- `path` — 按路径限流（各路径独立计数）
- `node` — 按节点限流（同一服务所有路径共享）
- `ip` — 按客户端 IP 限流

配置作用域（上游级 vs 路由级）：

```yaml
# 上游级：对该上游下所有路由生效
- Name: admin-core-http
    Middlewares:
      - TokenLimiter: ...
    Mappings:
      - Method: get
        Path: /api/admin-core/ping
        # 未指定，继承上游配置

# 路由级：仅对该路由生效，覆盖上游级配置
- Name: admin-core-http
    Mappings:
      - Method: get
        Path: /api/admin-core/ping
        Middlewares:
          - TokenLimiter: ...
```

### Retry 重试中间件

配置项：

```yaml
- Retry:
    attempts: 3                     # 重试次数（含首次请求）
    perTryTimeout: 0.1s             # 每次尝试的超时时间
    conditions:                     # 重试条件（任一匹配即触发）
      - byStatusCode: '502-504'     # 按状态码范围
      - byHeader:                   # 按响应头
          name: 'Grpc-Status'
          value: '14'
```

条件类型：
- `byStatusCode` — 匹配状态码范围（如 `500-504`、`502`、`4xx`、`5xx`）
- `byHeader` — 匹配响应头键值

## 添加新中间件

1. 在 `internal/middleware/` 下新建目录，如 `myplugin/`
2. 创建 `config.go`，定义 `MiddlewareName`、配置结构体、`FromEntries()`
3. 创建 `myplugin.go`，实现 `Middleware()` 函数
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
        myMw,           // ← 注册新中间件
    ),
)
```