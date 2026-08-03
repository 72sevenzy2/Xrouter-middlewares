package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/response"
)

// client request details
type client struct {
	requests int
	resetTime time.Time
}

var clients = map[string]*client{}
var mu sync.Mutex // cleanup gorountine will also modify client.

func RateLimiter() core.Middleware {
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			clientIp := r.RemoteAddr

			mu.Lock()
			c, ok := clients[clientIp]
			if !ok {
				c = &client{
					resetTime: time.Now().Add(time.Minute),
				}

				clients[clientIp] = c
			}

			if time.Now().After(c.resetTime) {
				c.requests = 0
				c.resetTime = time.Now().Add(time.Minute)
			}

			c.requests++


			if c.requests > 100 { // 100 requests cap for now
				mu.Unlock()
				response.JSON(w, response.WithError(http.StatusText(http.StatusTooManyRequests)), response.WithStatus(http.StatusTooManyRequests))
				return
			}

		}
	}
}
