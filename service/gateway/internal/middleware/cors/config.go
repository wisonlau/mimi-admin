package cors

import (
	"encoding/json"
	"strings"
)

// MiddlewareName 对应 gateway.yaml 中 Middlewares 下的 key 名称
const MiddlewareName = "Cors"

// CorsConf 跨域配置
type CorsConf struct {
	AllowCredentials bool        `json:"allowCredentials"`
	AllowHeaders     StringSlice `json:"allowHeaders"`
	AllowOrigins     []string    `json:"allowOrigins"`
	AllowMethods     []string    `json:"allowMethods"`
}

// StringSlice 支持从 JSON 字符串或数组解析
type StringSlice []string

func (s *StringSlice) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串（逗号分隔）
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = splitAndTrim(str)
		return nil
	}
	// 尝试解析为字符串数组
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*s = list
		return nil
	}
	*s = nil
	return nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
