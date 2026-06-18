package discovery_test

import (
    "testing"

    "bitExchange/internal/discovery"
)

func TestServiceNameIncludesBitExchangeAndDeviceID(t *testing.T) {
    got := discovery.ServiceInstanceName("device-123")
    want := "bitexchange-device-123"
    if got != want {
        t.Fatalf("ServiceInstanceName() = %q, want %q", got, want)
    }
}
