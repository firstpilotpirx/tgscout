package telegram

import (
	"sync"

	"github.com/gotd/td/telegram"
)

// ConnectionEvent is a change of the connection to Telegram during a run.
type ConnectionEvent int

const (
	// ConnectionLost is a drop of the connection; gotd reconnects by itself.
	ConnectionLost ConnectionEvent = iota + 1
	// ConnectionBack is the first ready connection after a drop.
	ConnectionBack
)

// connectionWatch turns gotd's connection states into events. gotd calls
// state on its own goroutine and must never wait, so the events go through a
// buffered channel to the watcher's goroutine.
type connectionWatch struct {
	mu     sync.Mutex
	ready  bool // a ready connection was seen in this run
	lost   bool // the connection dropped after it was ready
	events chan ConnectionEvent
	once   sync.Once
}

func newConnectionWatch() *connectionWatch {
	return &connectionWatch{events: make(chan ConnectionEvent, 64)}
}

// state takes one gotd state; it never blocks.
func (w *connectionWatch) state(state telegram.ConnectionState) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var event ConnectionEvent
	switch state {
	case telegram.ConnectionStateDisconnected:
		if w.ready && !w.lost {
			w.lost, event = true, ConnectionLost
		}
	case telegram.ConnectionStateReady:
		if w.lost {
			w.lost, event = false, ConnectionBack
		}
		w.ready = true
	case telegram.ConnectionStateConnecting:
	}
	if event == 0 {
		return
	}
	select {
	case w.events <- event:
	default: // the watcher is far behind; it reads the drop from the next events
	}
}

// watch calls f with every event on its own goroutine; only the first call counts.
func (w *connectionWatch) watch(f func(ConnectionEvent)) {
	w.once.Do(func() {
		go func() {
			for event := range w.events {
				f(event)
			}
		}()
	})
}

// close ends the watcher's goroutine once the events are read.
func (w *connectionWatch) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	close(w.events)
}
