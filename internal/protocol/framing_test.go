package protocol_test

import (
    "bytes"
    "testing"

    "bitExchange/internal/protocol"
)

func TestWriteAndReadFrameRoundTrip(t *testing.T) {
    var buf bytes.Buffer
    payload := []byte(`{"kind":"text"}`)

    if err := protocol.WriteFrame(&buf, payload); err != nil {
        t.Fatalf("WriteFrame returned error: %v", err)
    }

    got, err := protocol.ReadFrame(&buf)
    if err != nil {
        t.Fatalf("ReadFrame returned error: %v", err)
    }

    if string(got) != string(payload) {
        t.Fatalf("payload = %q, want %q", got, payload)
    }
}
