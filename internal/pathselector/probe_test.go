package pathselector_test

import (
    "net"
    "testing"
    "time"

    "bitExchange/internal/pathselector"
)

func TestClassifyReachableReturnsFailedForUnreachable(t *testing.T) {
    result := pathselector.ClassifyReachable("127.0.0.1", 19999)
    if result != pathselector.PathFailed {
        t.Fatalf("ClassifyReachable() = %s, want %s", result, pathselector.PathFailed)
    }
}

func TestPathPriorityOrderCorrect(t *testing.T) {
    if pathselector.PathLAN <= pathselector.PathPrivateCandidate {
        t.Fatalf("PathLAN should have higher value than PathPrivateCandidate")
    }
    if pathselector.PathPrivateCandidate <= pathselector.PathP2P {
        t.Fatalf("PathPrivateCandidate should have higher value than PathP2P")
    }
    if pathselector.PathP2P <= pathselector.PathRelay {
        t.Fatalf("PathP2P should have higher value than PathRelay")
    }
}

func TestProbeTCPDetectsOpenPort(t *testing.T) {
    listener, _ := net.Listen("tcp", "127.0.0.1:0")
    defer listener.Close()
    addr := listener.Addr().String()
    host, portStr, _ := net.SplitHostPort(addr)
    var port int
    for _, c := range portStr {
        port = port*10 + int(c-'0')
    }

    if !pathselector.ProbeTCP(host, port, 2*time.Second) {
        t.Fatalf("ProbeTCP should detect open port")
    }
}
