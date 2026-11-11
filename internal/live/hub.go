package live

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	channel       = "battle_events"
	minRetryDelay = time.Second
	maxRetryDelay = 30 * time.Second
	closeTimeout  = 5 * time.Second
)

type Hub struct {
	pool   *pgxpool.Pool
	logger *slog.Logger

	mu     sync.Mutex
	subs   map[string]map[chan struct{}]struct{}
	closed bool
}

func NewHub(pool *pgxpool.Pool, logger *slog.Logger) *Hub {
	return &Hub{pool: pool, logger: logger, subs: make(map[string]map[chan struct{}]struct{})}
}

func (h *Hub) Subscribe(battleID string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		close(ch)
		return ch, func() {}
	}
	set := h.subs[battleID]
	if set == nil {
		set = make(map[chan struct{}]struct{})
		h.subs[battleID] = set
	}
	set[ch] = struct{}{}
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if set, ok := h.subs[battleID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, battleID)
			}
		}
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer h.close()
	delay := minRetryDelay
	for {
		connected, err := h.listen(ctx)
		if ctx.Err() != nil {
			return
		}
		if connected {
			delay = minRetryDelay
		}
		h.logger.Error("battle event listener disconnected", "error", err, "retry_in", delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, maxRetryDelay)
	}
}

func (h *Hub) listen(ctx context.Context) (bool, error) {
	pooled, err := h.pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	conn := pooled.Hijack()
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()
	if _, err := conn.Exec(ctx, "LISTEN "+channel); err != nil {
		return false, err
	}
	h.logger.Info("battle event listener connected")
	h.signalAll()
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return true, err
		}
		h.signal(n.Payload)
	}
}

func (h *Hub) signal(battleID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[battleID] {
		notify(ch)
	}
}

func (h *Hub) signalAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, set := range h.subs {
		for ch := range set {
			notify(ch)
		}
	}
}

func (h *Hub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for id, set := range h.subs {
		for ch := range set {
			close(ch)
		}
		delete(h.subs, id)
	}
}

func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
