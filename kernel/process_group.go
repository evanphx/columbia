package kernel

import (
	"context"
	"sync"

	"github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/pkg/ilist"
	"github.com/evanphx/columbia/pkg/waiter"
)

type ProcessGroup struct {
	mu sync.RWMutex

	processCount int
	processes    ilist.List

	events waiter.Waiter
}

func (pg *ProcessGroup) RLock() {
	pg.mu.RLock()
}

func (pg *ProcessGroup) RUnlock() {
	pg.mu.RUnlock()
}

func NewProcessGroup() *ProcessGroup {
	pg := &ProcessGroup{}

	return pg
}

func (pg *ProcessGroup) Remove(p *Process) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	pg.processCount--
	pg.processes.Remove(p)
}

func (pg *ProcessGroup) Add(p *Process) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	pg.processCount++
	pg.processes.PushBack(p)
}

const (
	_ waiter.EventType = iota
	ProcessExitted
)

func (pg *ProcessGroup) ReapAny(ctx context.Context, block bool) (*Process, error) {
	if !block {
		return pg.reapOnce()
	}

	c := make(chan struct{}, 1)
	ev := pg.events.RegisterChannel(ProcessExitted, c)
	defer pg.events.Unregister(ev)

	for {
		process, err := pg.reapOnce()
		if err != nil {
			return nil, err
		}

		if process != nil {
			return process, nil
		}

		log.L.Trace("process-waiting-reap")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c:
			// ok, try the loop again
		}
	}
}

func (pg *ProcessGroup) reapOnce() (*Process, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	log.L.Trace("process-reap-once", "count", pg.processCount)

	for it := pg.processes.Front(); it != nil; it = it.Next() {
		p := it.(*Process)

		if p.status == Dead {
			pg.processCount--
			pg.processes.Remove(p)
			return p, nil
		}
	}

	return nil, nil
}

func (pg *ProcessGroup) ProcessExitted(p *Process) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	log.L.Trace("process-exitted", "pid", p.Pid)
	pg.events.Notify(ProcessExitted)

	return nil
}
