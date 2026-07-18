package cors

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"gateway/internal/middleware/tokenlimiter"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

// routeCorsConfig 按路由存储 CORS 配置
type routeCorsConfig struct {
	allowedOrigins map[string]bool
	allowedMethods string
	allowedHeaders string
}

// Middleware 返回 CORS 跨域中间件
func Middleware(upstreams []tokenlimiter.UpstreamConfig) rest.Middleware {
	routeCfgs := buildRouteConfigs(upstreams)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := strings.ToLower(r.Method + ":" + r.URL.Path)
			cfg, ok := routeCfgs[key]
			// OPTIONS 请求用该路径下任意已有方法匹配
			if !ok && r.Method == http.MethodOptions {
				cfg = findCorsForPath(r.URL.Path, routeCfgs)
				ok = cfg != nil
			}
			if !ok {
				next(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if isOriginAllowed(origin, cfg.allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", cfg.allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", cfg.allowedHeaders)
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}
}

// findCorsForPath 对于 OPTIONS 请求，查找路径对应的 CORS 配置
func findCorsForPath(path string, cfgs map[string]*routeCorsConfig) *routeCorsConfig {
	// 用常见的 HTTP 方法前缀去匹配
	for _, method := range []string{"get:", "post:", "put:", "delete:", "patch:"} {
		if cfg, ok := cfgs[method+strings.ToLower(path)]; ok {
			return cfg
		}
	}
	return nil
}

// buildRouteConfigs 遍历所有上游配置，提取有 Cors 的路由
func buildRouteConfigs(upstreams []tokenlimiter.UpstreamConfig) map[string]*routeCorsConfig {
	cfgs := make(map[string]*routeCorsConfig)
	var mu sync.Mutex

	for _, up := range upstreams {
		upCfg := FromEntries(up.Middlewares)

		for _, m := range up.Mappings {
			var cfg *CorsConf
			if mc := FromEntries(m.Middlewares); mc != nil {
				cfg = mc
			} else if upCfg != nil {
				cfg = upCfg
			} else {
				continue
			}

			key := strings.ToLower(m.Method + ":" + m.Path)
			mu.Lock()
			cfgs[key] = &routeCorsConfig{
				allowedOrigins: buildOriginSet(cfg.AllowOrigins),
				allowedMethods: buildMethodSet(cfg.AllowMethods),
				allowedHeaders: strings.Join(cfg.AllowHeaders, ", "),
			}
			mu.Unlock()
		}
	}
	return cfgs
}

// FromEntries 从中间件列表中提取 Cors 配置
func FromEntries(entries tokenlimiter.MiddlewaresConf) *CorsConf {
	for _, m := range entries {
		raw, ok := m[MiddlewareName]
		if !ok || raw == nil {
			continue
		}
		b, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var cfg CorsConf
		if err := json.Unmarshal(b, &cfg); err != nil {
			logx.Errorw("Cors 配置解析失败",
				logx.LogField{Key: "error", Value: err.Error()},
			)
			continue
		}
		return &cfg
	}
	return nil
}

func buildOriginSet(origins []string) map[string]bool {
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		set[strings.TrimSpace(o)] = true
	}
	return set
}

func buildMethodSet(methods []string) string {
	return strings.Join(methods, ", ")
}

func isOriginAllowed(origin string, allowed map[string]bool) bool {
	if len(allowed) == 0 || origin == "" {
		return false
	}
	if allowed[origin] {
		return true
	}
	for pattern := range allowed {
		if strings.HasPrefix(pattern, ".") && strings.HasSuffix(origin, pattern) {
			return true
		}
	}
	return false
}
