package worker

import (
	"context"
	"sync"
	"time"
)

type pauseWorkerController struct {
	mu         sync.Mutex
	pauseUntil time.Time
}

func (p *pauseWorkerController) Pause(dur time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	until := time.Now().Add(dur)
	if until.After(p.pauseUntil) {
		p.pauseUntil = until
	}
}

func (p *pauseWorkerController) Wait(ctx context.Context) error {
	for {
		p.mu.Lock()
		wait := time.Until(p.pauseUntil)
		p.mu.Unlock()

		if wait <= 0 {
			return nil
		}

		timer := time.NewTimer(wait)

		select {
		case <-timer.C:
			continue

		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}
