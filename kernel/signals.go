package kernel

import (
	"sync"

	"github.com/evanphx/columbia/log"
)

type Signals struct {
	mu       sync.Mutex
	Handlers map[int]int64
	waiting  map[int]struct{}
}

func (s *Signals) AddHandler(signo int, handler int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Handlers == nil {
		s.Handlers = make(map[int]int64)
	}

	if handler == 0 {
		delete(s.Handlers, signo)
		return
	}

	s.Handlers[signo] = handler
}

func (s *Signals) Handler(signo int) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	handler, ok := s.Handlers[signo]
	return handler, ok
}

func (s *Signals) Queue(signo int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.waiting == nil {
		s.waiting = make(map[int]struct{})
	}

	s.waiting[signo] = struct{}{}
}

func (s *Signals) Dequeue() (int, int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for signo, _ := range s.waiting {
		delete(s.waiting, signo)
		return signo, s.Handlers[signo], true
	}

	return 0, 0, false
}

func (p *Process) AddSignalHandler(signo int, handler int64) {
	handler = int64(p.Vm.ResolveFromTable(handler))

	log.L.Trace("add-signal-handler", "signal", signo, "handler", handler)
	p.signals.AddHandler(signo, handler)
}

// This doesn't execute the handler, it just sets up the process
// context
func (p *Process) DeliverSignal(signo int) error {
	p.signals.Queue(signo)
	p.Interrupt()
	return nil
}

func (p *Task) CheckInterrupt(ret int64) bool {
	signo, handler, ok := p.signals.Dequeue()
	if !ok {
		return false
	}

	log.L.Trace("process-setup-signal", "signal", signo, "handler", handler)

	p.Vm.SetupIntoFunction(ret, handler, uint64(signo))
	return true
}
