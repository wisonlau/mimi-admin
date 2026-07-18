package hellologging

import (
	"net/http"
	"strings"
	"time"

	"gateway/internal/middleware/tokenlimiter"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

// RouteEntry 带 HelloLogging 标记的路由
type RouteEntry struct {
	Method string
	Path   string
}

// RouteSet 记录哪些路由开启了 HelloLogging
type RouteSet map[string]bool

// BuildRouteSet 从路由列表构建开启集合
func BuildRouteSet(routes []RouteEntry) RouteSet {
	rs := make(RouteSet)
	for _, r := range routes {
		rs[strings.ToLower(r.Method+":"+r.Path)] = true
	}
	return rs
}

// Middleware 返回请求日志中间件，打印 hello in / hello out
func Middleware(routeSet RouteSet) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := strings.ToLower(r.Method + ":" + r.URL.Path)
			if !routeSet[key] {
				next(w, r)
				return
			}

			logx.Infow("hello in",
				logx.LogField{Key: "method", Value: r.Method},
				logx.LogField{Key: "path", Value: r.URL.Path},
			)

			start := time.Now()
			lrw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next(lrw, r)

			logx.Infow("hello out",
				logx.LogField{Key: "method", Value: r.Method},
				logx.LogField{Key: "path", Value: r.URL.Path},
				logx.LogField{Key: "status", Value: http.StatusText(lrw.statusCode)},
				logx.LogField{Key: "cost", Value: time.Since(start).String()},
			)
		}
	}
}

// HasHelloLogging 检查是否配置了 HelloLogging
func HasHelloLogging(entries tokenlimiter.MiddlewaresConf) bool {
	return tokenlimiter.HasMiddleware(entries, MiddlewareName)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
