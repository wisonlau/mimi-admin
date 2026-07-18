package tokenlimiter

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

// BuildLimiters 为每个有限流配置的路由创建 TokenBucket 限流器
func BuildLimiters(upstreams []UpstreamConfig, defaultRedis string, defaultRedisPass string) map[string]*limit.TokenLimiter {
	limiters := make(map[string]*limit.TokenLimiter)
	var mu sync.Mutex

	for _, up := range upstreams {
		upRL := ExtractTokenLimiter(up.Middlewares)

		for _, m := range up.Mappings {
			var rl *TokenLimiterConf
			if mrl := ExtractTokenLimiter(m.Middlewares); mrl != nil {
				rl = mrl
			} else if upRL != nil {
				rl = upRL
			} else {
				continue
			}

			key := fmt.Sprintf("%s:%s", m.Method, m.Path)
			redisConf := resolveRedisConf(rl, defaultRedis, defaultRedisPass)

			rds, err := redis.NewRedis(redisConf)
			if err != nil {
				logx.Errorw("限流器 Redis 连接失败",
					logx.LogField{Key: "path", Value: key},
					logx.LogField{Key: "error", Value: err.Error()},
				)
				continue
			}

			limiter := limit.NewTokenLimiter(rl.Rate, rl.Burst, rds, "gateway:"+MiddlewareName+":"+key)
			mu.Lock()
			limiters[key] = limiter
			mu.Unlock()

			logx.Infow("路由限流器已启用",
				logx.LogField{Key: "route", Value: key},
				logx.LogField{Key: "rate", Value: fmt.Sprintf("%d", rl.Rate)},
				logx.LogField{Key: "burst", Value: fmt.Sprintf("%d", rl.Burst)},
				logx.LogField{Key: "redis", Value: redisConf.Host},
			)
		}
	}
	return limiters
}

// resolveRedisConf 获取限流器的 Redis 配置，未配置 Host 则使用全局默认
func resolveRedisConf(rl *TokenLimiterConf, defaultHost, defaultPass string) redis.RedisConf {
	if rl.Redis.Host != "" {
		return rl.Redis.ToRedisConf()
	}
	return redis.RedisConf{
		Host: defaultHost,
		Pass: defaultPass,
		Type: "node",
	}
}

// Middleware 返回限流中间件，仅对配置了 TokenLimiter 的路由生效
func Middleware(limiters map[string]*limit.TokenLimiter) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := r.Method + ":" + r.URL.Path
			key = strings.ToLower(key)

			if limiter, ok := limiters[key]; ok {
				if ok := limiter.AllowCtx(r.Context()); !ok {
					logx.Infow("请求被限流",
						logx.LogField{Key: "path", Value: r.URL.Path},
						logx.LogField{Key: "method", Value: r.Method},
					)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(`{"code":429,"message":"触发限流，请稍后重试"}`))
					return
				}
			}
			next(w, r)
		}
	}
}
