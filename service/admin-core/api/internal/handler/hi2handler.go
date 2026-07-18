package handler

import (
	"net/http"
	"strings"

	"api/internal/logic"
	"api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func Hi2Handler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 URL 中提取 /hi/ 后面的部分作为 name
		name := strings.TrimPrefix(r.URL.Path, "/api/admin-core/hi/")
		l := logic.NewHiLogic(r.Context(), svcCtx)
		resp, err := l.Hi(name)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
