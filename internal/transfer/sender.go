package transfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"

	"bitExchange/internal/pathselector"
	"bitExchange/internal/protocol"
	isig "bitExchange/internal/signaling"
)

func SendText(addr string, from string, body string) error {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return err
    }
    defer conn.Close()

    payload, err := encodeTextMessage(from, body)
    if err != nil {
        return err
    }

    return protocol.WriteFrame(conn, payload)
}

func SendFile(addr string, from string, fileName string, reader io.Reader, size int64) error {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return err
    }
    defer conn.Close()

    header, err := json.Marshal(protocol.FileHeader{
        Kind:       "file",
        FromDevice: from,
        FileName:   fileName,
        FileSize:   size,
    })
    if err != nil {
        return err
    }

    if err := protocol.WriteFrame(conn, header); err != nil {
        return err
    }

    _, err = io.Copy(conn, reader)
    return err
}

func SendTextWithSelector(sel *pathselector.Selector, targetHost string, targetPort int, candidates []isig.OnlineEntry, from string, body string) (pathselector.Path, error) {
    path, err := sel.Select(targetHost, targetPort, candidates)
    if err != nil {
        return path, err
    }

    switch path {
    case pathselector.PathLAN, pathselector.PathPrivateCandidate:
        return path, SendText(fmt.Sprintf("%s:%d", targetHost, targetPort), from, body)
    case pathselector.PathRelay:
        return path, fmt.Errorf("relay send not yet wired through path selector — use relay.Sender directly")
    default:
        return path, fmt.Errorf("path %s not implemented", path)
    }
}
