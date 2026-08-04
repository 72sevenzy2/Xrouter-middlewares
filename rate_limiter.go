package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/response"
)

// client request details
type client struct {
	requests int
	resetAt  time.Time
}


var (
	clients = map[string]*client{}
	mu      sync.Mutex // cleanup gorountine will also modify client.
)

func RateLimiter(resetTime time.Duration, limit int) core.Middleware {
	// cleanup gorountine
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, c := range clients {
				if time.Now().After(c.resetAt) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			clientIp := r.RemoteAddr

			mu.Lock()
			c, ok := clients[clientIp]
			if !ok {
				c = &client{
					resetAt: time.Now().Add(resetTime),
				}

				clients[clientIp] = c
			}

			if time.Now().After(c.resetAt) {
				c.requests = 0
				c.resetAt = time.Now().Add(resetTime)
			}

			c.requests++

			if c.requests > limit { // 100 requests cap for now
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(c.resetAt).Seconds())))

				mu.Unlock()
				response.JSON(w, response.WithError(http.StatusText(http.StatusTooManyRequests)), response.WithStatus(http.StatusTooManyRequests))
				return
			}

			remaining := limit - c.requests
			reset := max(0, int(time.Until(c.resetAt).Seconds()))
			mu.Unlock()

			// headers
			w.Header().Set("RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("RateLimit-Reset", strconv.Itoa(reset))
			hf(w, r)
		}
	}
}
