package telegram

import (
	"testing"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/stretchr/testify/assert"
)

func TestTheFirstReadyIsNoEvent(t *testing.T) {
	watch := newConnectionWatch()
	var got []ConnectionEvent
	watch.watch(func(event ConnectionEvent) { got = append(got, event) })

	watch.state(telegram.ConnectionStateConnecting)
	watch.state(telegram.ConnectionStateReady)
	watch.close()

	assert.Empty(t, got)
}

func TestADropIsLostAndTheNextReadyIsBack(t *testing.T) {
	watch := newConnectionWatch()
	events := make(chan ConnectionEvent, 10)
	watch.watch(func(event ConnectionEvent) { events <- event })

	for _, state := range []telegram.ConnectionState{telegram.ConnectionStateReady,
		telegram.ConnectionStateDisconnected, telegram.ConnectionStateConnecting,
		telegram.ConnectionStateDisconnected, telegram.ConnectionStateReady, telegram.ConnectionStateReady} {
		watch.state(state)
	}

	assert.Equal(t, ConnectionLost, receive(t, events))
	assert.Equal(t, ConnectionBack, receive(t, events))
	assert.Empty(t, events, "a second Disconnected or Ready in a row is no event")
}

func TestTheStateCallbackNeverBlocks(t *testing.T) {
	watch := newConnectionWatch()
	held := make(chan struct{})
	watch.watch(func(ConnectionEvent) { <-held })
	defer close(held)
	done := make(chan struct{})

	go func() {
		watch.state(telegram.ConnectionStateReady)
		for range 100 {
			watch.state(telegram.ConnectionStateDisconnected)
			watch.state(telegram.ConnectionStateReady)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("gotd's state callback blocked on a slow watcher")
	}
}

func receive(t *testing.T, events <-chan ConnectionEvent) ConnectionEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("no connection event")
		return 0
	}
}
