package pathselector

import (
	"fmt"
	"net"
	"strconv"
	"time"

	isig "bitExchange/internal/signaling"
)

type SelectorConfig struct {
	LocalDeviceID   string
	LocalListenPort int
	RelayEnabled    bool
	RelayMaxBytes   int64
}

type Selector struct {
	cfg SelectorConfig
}

func NewSelector(cfg SelectorConfig) *Selector {
	return &Selector{cfg: cfg}
}

func (s *Selector) Select(targetHost string, targetPort int, candidates []isig.OnlineEntry) (Path, error) {
	// Step 1: try LAN direct
	if ProbeTCP(targetHost, targetPort, 2*time.Second) {
		return PathLAN, nil
	}

	// Step 2: try private candidate addresses from signaling
	for _, entry := range candidates {
		for _, candAddr := range entry.PrivateAddrs {
			host, port, err := net.SplitHostPort(candAddr)
			if err != nil {
				continue
			}
			portNum, err := strconv.Atoi(port)
			if err != nil {
				continue
			}
			if ProbeTCP(host, portNum, 2*time.Second) {
				return PathPrivateCandidate, nil
			}
		}
	}

	// Step 3: P2P hole-punch — not yet implemented in this stage
	// Step 4: relay fallback
	if s.cfg.RelayEnabled {
		return PathRelay, nil
	}

	return PathFailed, fmt.Errorf("no path available to %s:%d", targetHost, targetPort)
}
