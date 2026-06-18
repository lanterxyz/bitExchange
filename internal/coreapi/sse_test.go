package coreapi_test

import (
	"testing"
	"time"

	"bitExchange/internal/coreapi"
)

func TestSSEBrokerDeliversToSubscriber(t *testing.T) {
	broker := coreapi.NewSSEBroker()
	defer broker.Close()

	sub, cancel := broker.Subscribe()
	defer cancel()

	time.Sleep(50 * time.Millisecond)
	broker.Publish(coreapi.SSEEvent{Event: "progress", Data: "50%"})

	select {
	case got := <-sub:
		if got.Data != "50%" {
			t.Fatalf("got = %q", got.Data)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for event")
	}
}

func TestSSEBrokerUnsubscribeStopsDelivery(t *testing.T) {
	broker := coreapi.NewSSEBroker()
	defer broker.Close()

	_, cancel := broker.Subscribe()
	cancel()
	broker.Publish(coreapi.SSEEvent{Event: "x", Data: "y"})
}
