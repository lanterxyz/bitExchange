package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SenderConfig struct {
	RelayMaxBytes int64
}

type Sender struct {
	cfg SenderConfig
}

func NewSender(cfg SenderConfig) *Sender {
	return &Sender{cfg: cfg}
}

type relayTextPayload struct {
	Kind      string `json:"kind"`
	SessionID string `json:"session_id"`
	ToDevice  string `json:"to_device"`
	Body      string `json:"body"`
}

func (s *Sender) SendText(serverURL, sessionID, toDevice, body string) error {
	if int64(len(body)) > s.cfg.RelayMaxBytes {
		return fmt.Errorf("relay refused: text exceeds %d bytes limit", s.cfg.RelayMaxBytes)
	}

	payload := relayTextPayload{
		Kind:      "relay_text",
		SessionID: sessionID,
		ToDevice:  toDevice,
		Body:      body,
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(serverURL+"/relay/send", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("relay send failed: %s", string(respBody))
	}
	return nil
}
