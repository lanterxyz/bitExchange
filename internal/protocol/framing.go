package protocol

import (
    "encoding/binary"
    "io"
)

func WriteFrame(w io.Writer, payload []byte) error {
    if err := binary.Write(w, binary.BigEndian, uint32(len(payload))); err != nil {
        return err
    }
    _, err := w.Write(payload)
    return err
}

func ReadFrame(r io.Reader) ([]byte, error) {
    var size uint32
    if err := binary.Read(r, binary.BigEndian, &size); err != nil {
        return nil, err
    }

    payload := make([]byte, size)
    if _, err := io.ReadFull(r, payload); err != nil {
        return nil, err
    }

    return payload, nil
}
