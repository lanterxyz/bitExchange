package integration_test

import (
	"net"
	"testing"

	"bitExchange/internal/pathselector"
)

func TestPathSelectorSelectsLANForLocalListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	sel := pathselector.NewSelector(pathselector.SelectorConfig{
		LocalDeviceID:   "device-a",
		LocalListenPort: port,
		RelayEnabled:    false,
	})

	path, err := sel.Select(host, port, nil)
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}
	if path != pathselector.PathLAN {
		t.Fatalf("Select() = %s, want %s", path, pathselector.PathLAN)
	}
}
