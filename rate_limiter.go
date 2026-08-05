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

type Limiter struct {
	limit     int
	resetTime time.Duration

	clients map[string]*client
	mu      sync.Mutex
} // todo: create init func for this struct 

func NewLimiter(limit int, delay time.Duration) *Limiter {
	l := &Limiter{
		limit: limit,
		resetTime: delay,

		clients: make(map[string]*client),
		mu: sync.Mutex{},
	}

	go l.cleanup() // one cleanup per client
	return l
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		for ip, c := range l.clients {
			if time.Now().After(c.resetAt) {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *Limiter) RateLimiter() core.Middleware {
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			clientIp := r.RemoteAddr

			l.mu.Lock()
			c, ok := l.clients[clientIp]
			if !ok {
				c = &client{
					resetAt: time.Now().Add(l.resetTime),
				}

				l.clients[clientIp] = c
			}

			if time.Now().After(c.resetAt) {
				c.requests = 0
				c.resetAt = time.Now().Add(l.resetTime)
			}

			c.requests++

			if c.requests > l.limit { // 100 requests cap for now
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(c.resetAt).Seconds())))

				l.mu.Unlock()
				response.JSON(w, response.WithError(http.StatusText(http.StatusTooManyRequests)), response.WithStatus(http.StatusTooManyRequests))
				return
			}

			remaining := l.limit - c.requests
			reset := max(0, int(time.Until(c.resetAt).Seconds()))
			l.mu.Unlock()

			// headers
			w.Header().Set("RateLimit-Limit", strconv.Itoa(l.limit))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("RateLimit-Reset", strconv.Itoa(reset))
			hf(w, r)
		}
	}
}
