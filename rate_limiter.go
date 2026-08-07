package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/response"
)

type bucket struct {
	tokens     float64
	refilledAt time.Time
}

type Limiter struct {
	limit      float64
	refillRate float64

	clients map[string]*bucket
	mu      sync.Mutex
} // todo: create init func for this struct

func NewLimiter(limit float64, refillRate float64) *Limiter {
	l := &Limiter{
		limit:     limit,
		refillRate: refillRate,

		clients: make(map[string]*bucket),
		mu:      sync.Mutex{},
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
			if time.Since(c.refilledAt) > 10 * time.Minute {
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
				c = &bucket{
					tokens: l.limit,
					refilledAt: time.Now(),
				}

				l.clients[clientIp] = c
			}

			elapsed := time.Since(c.refilledAt).Seconds()
			newTokens := elapsed * l.refillRate

			c.tokens += newTokens
			c.tokens = min(c.tokens, l.limit)
			c.refilledAt = time.Now()

			if c.tokens < 1 { // 100 requests cap for now
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Duration(float64(time.Second) / l.refillRate))))

				l.mu.Unlock()
				response.JSON(w, response.WithError(http.StatusText(http.StatusTooManyRequests)), response.WithStatus(http.StatusTooManyRequests))
				return
			}

			c.tokens--

			l.mu.Unlock()

			// headers
			w.Header().Set("RateLimit-Limit", strconv.Itoa(int(l.limit)))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(int(c.tokens)))
			hf(w, r)
		}
	}
}
