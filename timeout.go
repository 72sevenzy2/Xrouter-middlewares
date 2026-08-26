package middleware

import (
	"context"
	"time"
	"net/http"
	"github.com/72sevenzy2/http-router/core"
)

// timeout middleware

func Timeout(t time.Duration) core.Middleware {
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), t) // initialising timeout (in seconds)

			defer cancel() // cancelling at the end of the func (current handler)

			// shallow copy of original request, (preserving other Request{} fields)
			req := *r
			req.Request = r.WithContext(ctx)

			hf(w, &req)
		}
	}
}
