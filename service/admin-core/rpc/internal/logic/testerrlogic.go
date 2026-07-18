package logic

import (
	"context"
	"fmt"

	"rpc/internal/svc"
	"rpc/rpc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type TesterrLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTesterrLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TesterrLogic {
	return &TesterrLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TesterrLogic) Testerr(in *rpc.Request) (*rpc.Response, error) {
	err := status.Error(codes.InvalidArgument, "invalid argument: "+in.Ping)

	// 设置 gRPC 响应 trailers，将错误 code 和 message 写入 trailers，
	// 便于 Postman 等客户端从 Trailers 中查看错误原因
	md := metadata.Pairs(
		"code-bin", fmt.Sprintf("%d", codes.InvalidArgument),
		"message-bin", "invalid argument: "+in.Ping,
		// "err-bin", "报错文件位置",
	)
	if err := grpc.SetTrailer(l.ctx, md); err != nil {
		l.Logger.Error(err)
	}

	return nil, err
}
