package transfer

import (
    "bytes"
    "encoding/json"
    "io"
    "net"

    "bitExchange/internal/protocol"
)

func AcceptOneTextMessage(listener net.Listener, chatFile string) error {
    conn, err := listener.Accept()
    if err != nil {
        return err
    }
    defer conn.Close()

    payload, err := protocol.ReadFrame(conn)
    if err != nil {
        return err
    }

    var msg protocol.TextMessage
    if err := json.Unmarshal(payload, &msg); err != nil {
        return err
    }

    return appendReceivedText(chatFile, msg.FromDevice, msg.Body)
}

func AcceptOneFile(listener net.Listener, receivedDir string) error {
    conn, err := listener.Accept()
    if err != nil {
        return err
    }
    defer conn.Close()

    payload, err := protocol.ReadFrame(conn)
    if err != nil {
        return err
    }

    var header protocol.FileHeader
    if err := json.Unmarshal(payload, &header); err != nil {
        return err
    }

    body, err := io.ReadAll(io.LimitReader(conn, header.FileSize))
    if err != nil {
        return err
    }

    _, err = SaveIncomingFile(receivedDir, header.FileName, bytes.NewReader(body))
    return err
}
