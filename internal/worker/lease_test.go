package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLeaseRenewalHasIndependentQueryDeadline(t *testing.T) {
	ticks := make(chan time.Time, 1)
	ticks <- time.Now()
	queryErr := errors.New("renewal failed")
	err := maintainLease(context.Background(), ticks, func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= 0 || remaining > 5*time.Second {
			t.Fatalf("renewal deadline=%v, remaining=%v", ok, remaining)
		}
		return queryErr
	})
	if !errors.Is(err, queryErr) {
		t.Fatalf("renewal result=%v", err)
	}
}

func TestLeaseMonitorExitsOnProcessingCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := maintainLease(ctx, make(chan time.Time), func(context.Context) error { t.Fatal("unexpected renewal"); return nil }); err != nil {
		t.Fatal(err)
	}
}
