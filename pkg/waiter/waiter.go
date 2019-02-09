package waiter

import (
	"sync"

	"github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/pkg/ilist"
)

type EventType uint64

type Waiter struct {
	mu sync.RWMutex

	count   int
	waiters ilist.List
}

type Event struct {
	ilist.Entry

	Mask     EventType
	Context  interface{}
	Callback func(e *Event)
}

func (w *Waiter) Register(e *Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.count++

	w.waiters.PushBack(e)
}

func triggerChan(e *Event) {
	c := e.Context.(chan struct{})

	select {
	case c <- struct{}{}:
	default:
	}
}

func (w *Waiter) RegisterChannel(mask EventType, c chan struct{}) *Event {
	e := &Event{
		Callback: triggerChan,
		Context:  c,
		Mask:     mask,
	}

	w.Register(e)

	return e
}

func (w *Waiter) Unregister(e *Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.count--

	w.waiters.Remove(e)
}

func (w *Waiter) Notify(mask EventType) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	log.L.Trace("waiters-notify", "count", w.count)

	for it := w.waiters.Front(); it != nil; it = it.Next() {
		e := it.(*Event)
		log.L.Trace("waiters-walk", "event-mask", e.Mask, "notify-mask", mask, "match", mask&e.Mask)
		if mask&e.Mask != 0 {
			e.Callback(e)
		}
	}
}
