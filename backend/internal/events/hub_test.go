package events

import (
	"io"
	"log/slog"
	"testing"
)

func newTestHub() *Hub {
	return NewHub("test", slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func received(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestPublishReachesOnlyThatGame(t *testing.T) {
	h := newTestHub()
	a, unsubA := h.Subscribe("a")
	defer unsubA()
	b, unsubB := h.Subscribe("b")
	defer unsubB()

	h.Publish("a")
	if !received(a) {
		t.Error("subscriber of a got no signal")
	}
	if received(b) {
		t.Error("subscriber of b got a signal meant for a")
	}
}

func TestSignalsCoalesce(t *testing.T) {
	h := newTestHub()
	ch, unsub := h.Subscribe("a")
	defer unsub()

	h.Publish("a")
	h.Publish("a") // must not block even though nobody has read yet
	if !received(ch) {
		t.Fatal("expected a signal")
	}
	if received(ch) {
		t.Fatal("expected the two signals to be coalesced into one")
	}
}

func TestUnsubscribe(t *testing.T) {
	h := newTestHub()
	ch, unsub := h.Subscribe("a")
	unsub()
	h.Publish("a")
	if received(ch) {
		t.Fatal("unsubscribed channel received a signal")
	}
	if len(h.subs) != 0 {
		t.Fatalf("subs map not cleaned up: %v", h.subs)
	}
}

func TestPublishAllAndClose(t *testing.T) {
	h := newTestHub()
	a, unsubA := h.Subscribe("a")
	defer unsubA()
	b, unsubB := h.Subscribe("b")
	defer unsubB()

	h.publishAll()
	if !received(a) || !received(b) {
		t.Fatal("publishAll should wake every subscriber")
	}

	h.Close()
	h.Close() // second call must not panic
	select {
	case <-h.Done():
	default:
		t.Fatal("Done not closed after Close")
	}
}
