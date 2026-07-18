package retry

// MiddlewareName 对应 gateway.yaml 中 Middlewares 下的 key 名称
const MiddlewareName = "Retry"

// RetryConf 重试配置
type RetryConf struct {
	Attempts      int                    `json:"attempts"`
	PerTryTimeout string                 `json:"perTryTimeout"`
	Conditions    []map[string]interface{} `json:"conditions"`
}
