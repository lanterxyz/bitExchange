package pathselector

import (
    "fmt"
    "net"
    "time"
)

type Path int

const (
    PathFailed           Path = 0
    PathRelay            Path = 1
    PathP2P              Path = 2
    PathPrivateCandidate Path = 3
    PathLAN              Path = 4
)

func (p Path) String() string {
    switch p {
    case PathLAN:
        return "lan-direct"
    case PathPrivateCandidate:
        return "private-candidate"
    case PathP2P:
        return "p2p"
    case PathRelay:
        return "relay"
    case PathFailed:
        return "failed"
    default:
        return "unknown"
    }
}

func ProbeTCP(host string, port int, timeout time.Duration) bool {
    addr := fmt.Sprintf("%s:%d", host, port)
    conn, err := net.DialTimeout("tcp", addr, timeout)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}

func ClassifyReachable(host string, port int) Path {
    if ProbeTCP(host, port, 2*time.Second) {
        return PathLAN
    }
    return PathFailed
}
