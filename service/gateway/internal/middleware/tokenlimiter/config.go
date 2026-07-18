package tokenlimiter

import (
	"encoding/json"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// MiddlewareName 对应 gateway.yaml 中 Middlewares 下的 key 名称
const MiddlewareName = "TokenLimiter"

type (
	// RedisConf 限流器专用 Redis 配置（不校验 Type，避免与 TokenLimiter.Type 冲突）
	RedisConf struct {
		Host string `json:",optional"`
		Pass string `json:",optional"`
	}

	// TokenLimiterConf 限流器配置
	TokenLimiterConf struct {
		Redis RedisConf
		Type  string `json:",optional"` // 限流类型（path/user/ip等）
		Rate  int    `json:",default=100"`
		Burst int    `json:",default=200"`
	}

	// MiddlewaresConf 中间件配置列表，每个元素是 {middlewareName: config}
	MiddlewaresConf []map[string]interface{}

	// UpstreamConfig 给 BuildLimiters 使用的精简上游配置
	UpstreamConfig struct {
		Middlewares MiddlewaresConf
		Mappings    []MappingRateLimit
	}

	// MappingRateLimit 给 BuildLimiters 使用的精简路由限流配置
	MappingRateLimit struct {
		Method      string
		Path        string
		Middlewares MiddlewaresConf
	}
)

// ToRedisConf 转换为 redis.RedisConf
func (r *RedisConf) ToRedisConf() redis.RedisConf {
	return redis.RedisConf{
		Host: r.Host,
		Pass: r.Pass,
		Type: "node",
	}
}

// ExtractTokenLimiter 从中间件列表中提取 TokenLimiter 配置
func ExtractTokenLimiter(entries MiddlewaresConf) *TokenLimiterConf {
	for _, m := range entries {
		raw, ok := m[MiddlewareName]
		if !ok || raw == nil {
			continue
		}
		b, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var conf TokenLimiterConf
		if err := json.Unmarshal(b, &conf); err != nil {
			continue
		}
		return &conf
	}
	return nil
}

// HasMiddleware 检查中间件列表中是否包含指定名称的中间件
func HasMiddleware(entries MiddlewaresConf, name string) bool {
	for _, m := range entries {
		if _, ok := m[name]; ok {
			return true
		}
	}
	return false
}