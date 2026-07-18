package logic

import (
	"context"

	"rpc/internal/svc"
	"rpc/rpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type HelloLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHelloLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HelloLogic {
	return &HelloLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HelloLogic) Hello(in *rpc.HelloRequest) (*rpc.HelloResponse, error) {
	return &rpc.HelloResponse{
		Message: "hello, " + in.Name + " from admin-core rpc",
	}, nil
}