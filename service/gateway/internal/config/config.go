package config

import (
	"fmt"
	"time"

	"gateway/internal/middleware/hellologging"
	"gateway/internal/middleware/tokenlimiter"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/gateway"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type (
	// RouteMappingExt 扩展路由映射，Middlewares 为数组 [{name: config}, ...]
	RouteMappingExt struct {
		Method      string                    `json:",optional"`
		Path        string
		RpcPath     string                    `json:",optional"`
		Middlewares tokenlimiter.MiddlewaresConf `json:",optional"`
	}

	// HttpClientConfExt 扩展 HTTP 上游配置，增加 etcd 服务发现支持
	HttpClientConfExt struct {
		Target  string          `json:",optional"`
		Prefix  string          `json:",optional"`
		Timeout string          `json:",optional"` // 支持 "5s"、"500ms"、"3000"（纯数字毫秒）
		Etcd    discov.EtcdConf `json:",optional"`
	}

	// UpstreamExt 扩展上游配置
	UpstreamExt struct {
		Name        string                      `json:",optional"`
		Grpc        *zrpc.RpcClientConf         `json:",optional"`
		Http        *HttpClientConfExt           `json:",optional=!grpc"`
		ProtoSets   []string                    `json:",optional"`
		Middlewares tokenlimiter.MiddlewaresConf  `json:",optional"`
		Mappings    []RouteMappingExt           `json:",optional"`
	}

	// GlobalRedisConf 全局 Redis 配置
	GlobalRedisConf struct {
		Address string `json:",optional"`
		Pass    string `json:",optional"`
	}

	// ResilienceConf 服务韧性组件，可选分组，不写则走 go-zero 默认值
	ResilienceConf struct {
		CpuThreshold int64  `json:",optional"`
		Shedding     *bool  `json:",optional"`
		Breaker      *bool  `json:",optional"`
		Timeout      *bool  `json:",optional"`
		Recover      *bool  `json:",optional"`
		Trace        *bool  `json:",optional"`
		Log          *bool  `json:",optional"`
		Prometheus   *bool  `json:",optional"`
		MaxConns     *bool  `json:",optional"`
	}

	// Config 网关配置
	Config struct {
		rest.RestConf
		Etcd       discov.EtcdConf  `json:",optional"`
		Resilience *ResilienceConf  `json:",optional"`
		Upstreams  []UpstreamExt
		Redis      GlobalRedisConf `json:",optional"`
	}
)

// ApplyResilience 将 Resilience 配置应用到 RestConf
func (c *Config) ApplyResilience() {
	if c.Resilience == nil {
		return
	}
	r := c.Resilience
	if r.CpuThreshold > 0 {
		c.RestConf.CpuThreshold = r.CpuThreshold
	}
	if r.Shedding != nil {
		c.RestConf.Middlewares.Shedding = *r.Shedding
	}
	if r.Breaker != nil {
		c.RestConf.Middlewares.Breaker = *r.Breaker
	}
	if r.Timeout != nil {
		c.RestConf.Middlewares.Timeout = *r.Timeout
	}
	if r.Recover != nil {
		c.RestConf.Middlewares.Recover = *r.Recover
	}
	if r.Trace != nil {
		c.RestConf.Middlewares.Trace = *r.Trace
	}
	if r.Log != nil {
		c.RestConf.Middlewares.Log = *r.Log
	}
	if r.Prometheus != nil {
		c.RestConf.Middlewares.Prometheus = *r.Prometheus
	}
	if r.MaxConns != nil {
		c.RestConf.Middlewares.MaxConns = *r.MaxConns
	}
}

// ToGatewayConf 转换为 gateway.GatewayConf
func (c *Config) ToGatewayConf() gateway.GatewayConf {
	gc := gateway.GatewayConf{
		RestConf: c.RestConf,
	}
	for _, up := range c.Upstreams {
		gUp := gateway.Upstream{
			Name:      up.Name,
			Grpc:      up.Grpc,
			ProtoSets: up.ProtoSets,
		}
		if up.Http != nil {
			gUp.Http = &gateway.HttpClientConf{
				Target:  up.Http.Target,
				Prefix:  up.Http.Prefix,
				Timeout: resolveTimeout(up.Http.Timeout),
			}
		}
		for _, m := range up.Mappings {
			gUp.Mappings = append(gUp.Mappings, gateway.RouteMapping{
				Method:  m.Method,
				Path:    m.Path,
				RpcPath: m.RpcPath,
			})
		}
		gc.Upstreams = append(gc.Upstreams, gUp)
	}
	return gc
}

// resolveTimeout 将时间配置转为毫秒，支持 "5s"、"500ms"、"3000"（纯数字毫秒）三种格式
func resolveTimeout(s string) int64 {
	if s == "" {
		return 3000
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d.Milliseconds()
	}
	var ms int64
	if n, _ := fmt.Sscanf(s, "%d", &ms); n == 1 {
		return ms
	}
	return 3000
}

// ToUpstreamConfigs 转换为 tokenlimiter.UpstreamConfig 列表
func (c *Config) ToUpstreamConfigs() []tokenlimiter.UpstreamConfig {
	var result []tokenlimiter.UpstreamConfig
	for _, up := range c.Upstreams {
		uc := tokenlimiter.UpstreamConfig{
			Middlewares: up.Middlewares,
		}
		for _, m := range up.Mappings {
			uc.Mappings = append(uc.Mappings, tokenlimiter.MappingRateLimit{
				Method:      m.Method,
				Path:        m.Path,
				Middlewares: m.Middlewares,
			})
		}
		result = append(result, uc)
	}
	return result
}

// ToHelloLoggingRoutes 提取开启了 HelloLogging 的路由列表
func (c *Config) ToHelloLoggingRoutes() []hellologging.RouteEntry {
	var routes []hellologging.RouteEntry
	for _, up := range c.Upstreams {
		upEnabled := hellologging.HasHelloLogging(up.Middlewares)
		for _, m := range up.Mappings {
			enabled := upEnabled || hellologging.HasHelloLogging(m.Middlewares)
			if enabled {
				routes = append(routes, hellologging.RouteEntry{
					Method: m.Method,
					Path:   m.Path,
				})
			}
		}
	}
	return routes
}