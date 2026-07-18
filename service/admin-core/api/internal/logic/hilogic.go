package logic

import (
	"context"
	"fmt"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HiLogic {
	return &HiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HiLogic) Hi(name string) (resp *types.HiResponse, err error) {
	return &types.HiResponse{
		Message: fmt.Sprintf("hi, %s from admin-core", name),
	}, nil
}
