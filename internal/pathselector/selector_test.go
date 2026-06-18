package pathselector_test

import (
	"net"
	"testing"

	"bitExchange/internal/pathselector"
)

func TestSelectorPrefersLANWhenReachable(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
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
		t.Fatalf("Select returned error: %v", err)
	}
	if path != pathselector.PathLAN {
		t.Fatalf("Select() = %s, want %s", path, pathselector.PathLAN)
	}
}

func TestSelectorReturnsFailedWhenAllPathsUnavailable(t *testing.T) {
	sel := pathselector.NewSelector(pathselector.SelectorConfig{
		LocalDeviceID:   "device-a",
		LocalListenPort: 19001,
		RelayEnabled:    false,
	})

	_, err := sel.Select("10.255.255.1", 19999, nil)
	if err == nil {
		t.Fatalf("Select should have failed on unreachable address")
	}
}

func TestSelectorReturnsRelayWhenEnabledAndDirectFails(t *testing.T) {
	sel := pathselector.NewSelector(pathselector.SelectorConfig{
		LocalDeviceID:   "device-a",
		LocalListenPort: 19001,
		RelayEnabled:    true,
	})

	path, err := sel.Select("10.255.255.1", 19999, nil)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if path != pathselector.PathRelay {
		t.Fatalf("Select() = %s, want %s", path, pathselector.PathRelay)
	}
}
