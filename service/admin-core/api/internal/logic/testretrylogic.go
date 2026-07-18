package logic

import (
	"context"
	"math/rand"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type TestRetryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTestRetryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestRetryLogic {
	return &TestRetryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestRetryLogic) TestRetry() (resp *types.RetryResponse, err error) {
	// 模拟随机失败：50% 概率返回 503
	if rand.Intn(100) < 50 {
		return nil, &RetryError{Code: http.StatusServiceUnavailable, Message: "service temporarily unavailable"}
	}
	return &types.RetryResponse{
		Message: "success",
	}, nil
}

type RetryError struct {
	Code    int
	Message string
}

func (e *RetryError) Error() string {
	return e.Message
}

func (e *RetryError) WriteTo(w http.ResponseWriter) {
	httpx.WriteJson(w, e.Code, map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	})
}
