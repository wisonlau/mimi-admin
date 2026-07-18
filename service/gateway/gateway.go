package main

import (
	"flag"
	"fmt"

	"gateway/internal/config"
	"gateway/internal/middleware/cors"
	"gateway/internal/middleware/hellologging"
	"gateway/internal/middleware/tokenlimiter"
	"gateway/internal/middleware/retry"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/netx"
	"github.com/zeromicro/go-zero/gateway"
)

var configFile = flag.String("f", "etc/gateway.yaml", "config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.ApplyResilience()

	// 初始化日志
	logx.MustSetup(c.Log)

	// 注册 etcd（网关自身）
	if len(c.Etcd.Hosts) > 0 && len(c.Etcd.Key) > 0 {
		pubListenOn := fmt.Sprintf("%s:%d", c.Host, c.Port)
		if c.Host == "0.0.0.0" {
			ip := netx.InternalIp()
			if ip != "" {
				pubListenOn = fmt.Sprintf("%s:%d", ip, c.Port)
			}
		}
		pubClient := discov.NewPublisher(c.Etcd.Hosts, c.Etcd.Key, pubListenOn)
		if err := pubClient.KeepAlive(); err != nil {
			logx.Error(err)
		}
		defer pubClient.Stop()
	}

	// 从 etcd 解析 HTTP 上游地址
	if err := resolveHttpTargets(c.Upstreams); err != nil {
		logx.Error(err)
	}

	// 构建上游配置列表（供各中间件使用）
	upstreamConfigs := c.ToUpstreamConfigs()

	// ------ 创建中间件 ------
	rateLimiters := tokenlimiter.BuildLimiters(upstreamConfigs, c.Redis.Address, c.Redis.Pass)
	rateLimitMiddleware := tokenlimiter.Middleware(rateLimiters)

	helloRouteSet := hellologging.BuildRouteSet(c.ToHelloLoggingRoutes())
	helloLoggingMiddleware := hellologging.Middleware(helloRouteSet)

	corsMiddleware := cors.Middleware(upstreamConfigs)
	retryMiddleware := retry.Middleware(upstreamConfigs)

	// 构建 GatewayConf 并自动为 CORS 路由添加 OPTIONS 映射
	gwConf := c.ToGatewayConf()
	addCorsOptionsRoutes(&gwConf, upstreamConfigs)

	// 洋葱模型：由外到内 cors → helloLogging → rateLimit → retry
	gw := gateway.MustNewServer(
		gwConf,
		gateway.WithMiddleware(
			corsMiddleware,
			helloLoggingMiddleware,
			rateLimitMiddleware,
			retryMiddleware,
		),
	)
	defer gw.Stop()

	gw.Start()
}

// resolveHttpTargets 遍历所有上游，对有 Etcd 配置的 HTTP 上游从 etcd 解析目标地址，
// 如果上游没有配置 Http 块，则自动创建。
func resolveHttpTargets(upstreams []config.UpstreamExt) error {
	for i := range upstreams {
		up := &upstreams[i]
		if up.Grpc != nil || up.Http == nil || len(up.Http.Etcd.Hosts) == 0 || len(up.Http.Etcd.Key) == 0 {
			continue
		}

		sub, err := discov.NewSubscriber(up.Http.Etcd.Hosts, up.Http.Etcd.Key)
		if err != nil {
			return fmt.Errorf("failed to subscribe to etcd key %q: %w", up.Http.Etcd.Key, err)
		}

		values := sub.Values()
		sub.Close()
		if len(values) == 0 {
			return fmt.Errorf("no value found for etcd key %q", up.Http.Etcd.Key)
		}

		up.Http.Target = values[0]
		logx.Infof("upstream %q resolved target from etcd: %s", up.Name, up.Http.Target)
	}
	return nil
}

// addCorsOptionsRoutes 为有 CORS 配置的路由自动添加 OPTIONS 预检路由
func addCorsOptionsRoutes(gwConf *gateway.GatewayConf, upstreamConfigs []tokenlimiter.UpstreamConfig) {
	for i, up := range upstreamConfigs {
		if cors.FromEntries(up.Middlewares) == nil {
			hasChild := false
			for _, m := range up.Mappings {
				if cors.FromEntries(m.Middlewares) != nil {
					hasChild = true
					gwConf.Upstreams[i].Mappings = append(
						gwConf.Upstreams[i].Mappings,
						gateway.RouteMapping{
							Method: "options",
							Path:   m.Path,
						},
					)
				}
			}
			if !hasChild {
				continue
			}
		} else {
			for _, m := range up.Mappings {
				gwConf.Upstreams[i].Mappings = append(
					gwConf.Upstreams[i].Mappings,
					gateway.RouteMapping{
						Method: "options",
						Path:   m.Path,
					},
				)
			}
		}
	}
}