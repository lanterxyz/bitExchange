package coreapi

import "sync"

type SSEEvent struct {
	Event string
	Data  string
}

type SSEBroker struct {
	mu          sync.Mutex
	subscribers map[chan SSEEvent]struct{}
	closed      bool
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{subscribers: map[chan SSEEvent]struct{}{}}
}

func (b *SSEBroker) Subscribe() (chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 16)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
	return ch, cancel
}

func (b *SSEBroker) Publish(ev SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	for ch := range b.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *SSEBroker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}
