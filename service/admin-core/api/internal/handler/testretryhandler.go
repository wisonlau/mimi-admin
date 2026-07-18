package handler

import (
	"net/http"

	"api/internal/logic"
	"api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestRetryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewTestRetryLogic(r.Context(), svcCtx)
		resp, err := l.TestRetry()
		if err != nil {
			if re, ok := err.(*logic.RetryError); ok {
				httpx.WriteJson(w, re.Code, map[string]interface{}{
					"code":    re.Code,
					"message": re.Message,
				})
				return
			}
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}