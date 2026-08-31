package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/response"
)

type Bucket struct {
	tokens     float64
	refilledAt time.Time
}

type Limiter struct {
	Limit      float64
	RefillRate float64

	mu      sync.Mutex
	Clients map[string]*Bucket
} // todo: create init func for this struct

func NewLimiter(limit int, refillRate int) *Limiter {
	l := &Limiter{
		Limit:      float64(limit),
		RefillRate: float64(refillRate),

		Clients: make(map[string]*Bucket),
	}



	go l.cleanup(time.Minute * 5) // one cleanup per client
	return l
}

func (l *Limiter) cleanup(t time.Duration) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		for ip, c := range l.Clients {
			if time.Since(c.refilledAt) > t {
				delete(l.Clients, ip)
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
			c, ok := l.Clients[clientIp]
			if !ok {
				c = &Bucket{
					tokens:     l.Limit,
					refilledAt: time.Now(),
				}

				l.Clients[clientIp] = c
			}

			elapsed := time.Since(c.refilledAt).Seconds()
			newTokens := elapsed * l.RefillRate

			c.tokens += newTokens
			c.tokens = min(c.tokens, l.Limit)
			c.refilledAt = time.Now()

			if c.tokens < 1 { // 100 requests cap for now
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Duration(float64(time.Second)/l.RefillRate))))

				l.mu.Unlock()
				response.JSON(w, http.StatusTooEarly, nil, http.StatusText(http.StatusTooEarly))
				return
			}

			c.tokens--

			l.mu.Unlock()

			// headers
			w.Header().Set("RateLimit-Limit", strconv.Itoa(int(l.Limit)))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(int(c.tokens)))
			hf(w, r)
		}
	}
}
