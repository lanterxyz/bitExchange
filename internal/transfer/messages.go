package transfer

import (
    "encoding/json"
    "fmt"
    "time"

    "bitExchange/internal/history"
    "bitExchange/internal/protocol"
)

func appendReceivedText(chatFile string, from string, body string) error {
    line := fmt.Sprintf("[%s] %s: %s", time.Now().Format("2006-01-02 15:04:05"), from, body)
    return history.AppendMessage(chatFile, line)
}

func encodeTextMessage(from string, body string) ([]byte, error) {
    return json.Marshal(protocol.TextMessage{
        Kind:       "text",
        FromDevice: from,
        Body:       body,
    })
}
