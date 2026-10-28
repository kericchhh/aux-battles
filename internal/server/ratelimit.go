package server

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	rate      rate.Limit
	burst     int
	nextSweep time.Time
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
	}
}

func (l *ipLimiter) allow(remoteAddr string) bool {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}
	now := time.Now()
	l.mu.Lock()
	if now.After(l.nextSweep) {
		for key, v := range l.visitors {
			if now.Sub(v.lastSeen) > 10*time.Minute {
				delete(l.visitors, key)
			}
		}
		l.nextSweep = now.Add(time.Minute)
	}
	v := l.visitors[ip]
	if v == nil {
		v = &visitor{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = now
	l.mu.Unlock()

	return v.limiter.Allow()
}

func (l *ipLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(r.RemoteAddr) {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next(w, r)
	}
}
