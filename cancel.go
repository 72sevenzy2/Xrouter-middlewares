package middleware

import (
	"context"
	"net/http"

	"github.com/72sevenzy2/http-router/core"
)

func Canceller() (core.Middleware, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	mw := func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			reqCtx, c := context.WithCancel(ctx)
			defer c()

			r.Request = r.Request.WithContext(reqCtx)
			hf(w, r)
		}
	}

	return mw, cancel
}
