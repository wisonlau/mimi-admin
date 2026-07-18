package retry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gateway/internal/middleware/tokenlimiter"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

// Middleware 返回重试中间件
func Middleware(upstreams []tokenlimiter.UpstreamConfig) rest.Middleware {
	routeCfgs := buildRetryConfigs(upstreams)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := strings.ToLower(r.Method + ":" + r.URL.Path)
			cfg, ok := routeCfgs[key]
			if !ok {
				next(w, r)
				return
			}

			// 缓冲请求体，支持多次读取
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body.Close()
			}

			// 重试逻辑
			var lastResp *bufferedResponseWriter
			for attempt := 1; attempt <= cfg.Attempts; attempt++ {
				if attempt > 1 {
					logx.Infow("重试请求",
						logx.LogField{Key: "attempt", Value: attempt},
						logx.LogField{Key: "path", Value: r.URL.Path},
					)
				}

				// 每次重试创建新的请求体
				if bodyBytes != nil {
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}

				// 用缓冲 ResponseWriter 捕获响应
				lrw := &bufferedResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
				next(lrw, r)

				// 检查是否需要重试
				if attempt < cfg.Attempts && shouldRetry(lrw, cfg) {
					lastResp = lrw
					continue
				}

				// 最后一次尝试或成功，写出响应
				writeResponse(w, lrw)
				return
			}

			// 所有重试都失败，写出最后一次的响应
			if lastResp != nil {
				writeResponse(w, lastResp)
			}
		}
	}
}

// shouldRetry 根据条件判断是否需要重试
func shouldRetry(lrw *bufferedResponseWriter, cfg *RetryConf) bool {
	for _, cond := range cfg.Conditions {
		// byStatusCode
		if raw, ok := cond["byStatusCode"]; ok {
			rangeStr, ok := raw.(string)
			if !ok {
				continue
			}
			if matchStatusCode(lrw.statusCode, rangeStr) {
				return true
			}
		}
		// byHeader
		if raw, ok := cond["byHeader"]; ok {
			headerMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := headerMap["name"].(string)
			val, _ := headerMap["value"].(string)
			if name != "" && lrw.Header().Get(name) == val {
				return true
			}
		}
	}
	return false
}

// matchStatusCode 检查状态码是否匹配范围表达式（如 "502-504"）
func matchStatusCode(code int, rangeStr string) bool {
	parts := strings.SplitN(rangeStr, "-", 2)
	if len(parts) != 2 {
		return false
	}
	start, end := 0, 0
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return false
		}
		start = start*10 + int(c-'0')
	}
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return false
		}
		end = end*10 + int(c-'0')
	}
	return code >= start && code <= end
}

// bufferedResponseWriter 缓冲响应，用于判断是否重试
type bufferedResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	headers    http.Header
	written    bool
}

func (w *bufferedResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *bufferedResponseWriter) WriteHeader(code int) {
	if w.written {
		return
	}
	w.written = true
	w.statusCode = code
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(data)
}

// writeResponse 将缓冲的响应写出到真正的 ResponseWriter
func writeResponse(w http.ResponseWriter, lrw *bufferedResponseWriter) {
	for key, values := range lrw.headers {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(lrw.statusCode)
	if lrw.body.Len() > 0 {
		w.Write(lrw.body.Bytes())
	}
}

// buildRetryConfigs 遍历所有上游配置，提取有 Retry 的路由
func buildRetryConfigs(upstreams []tokenlimiter.UpstreamConfig) map[string]*RetryConf {
	cfgs := make(map[string]*RetryConf)

	for _, up := range upstreams {
		upCfg := FromEntries(up.Middlewares)

		for _, m := range up.Mappings {
			var cfg *RetryConf
			if mc := FromEntries(m.Middlewares); mc != nil {
				cfg = mc
			} else if upCfg != nil {
				cfg = upCfg
			} else {
				continue
			}

			key := strings.ToLower(m.Method + ":" + m.Path)
			cfgs[key] = cfg
		}
	}
	return cfgs
}

// FromEntries 从中间件列表中提取 Retry 配置
func FromEntries(entries tokenlimiter.MiddlewaresConf) *RetryConf {
	for _, m := range entries {
		raw, ok := m[MiddlewareName]
		if !ok || raw == nil {
			continue
		}
		b, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var cfg RetryConf
		if err := json.Unmarshal(b, &cfg); err != nil {
			logx.Errorw("Retry 配置解析失败",
				logx.LogField{Key: "error", Value: err.Error()},
			)
			continue
		}
		return &cfg
	}
	return nil
}
