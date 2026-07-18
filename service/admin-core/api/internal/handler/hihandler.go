package handler

import (
	"net/http"

	"api/internal/logic"
	"api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func HiHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		l := logic.NewHiLogic(r.Context(), svcCtx)
		resp, err := l.Hi(name)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
