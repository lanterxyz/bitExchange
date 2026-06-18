package signaling

import (
	"net"
	"time"
)

type OnlineEntry struct {
	DeviceID      string    `json:"device_id"`
	DeviceName    string    `json:"device_name"`
	Fingerprint   string    `json:"fingerprint"`
	ListenPort    int       `json:"listen_port"`
	PrivateAddrs  []string  `json:"private_addrs"`
	RelayEnabled  bool      `json:"relay_enabled"`
	RelayMaxBytes int64     `json:"relay_max_bytes"`
	RegisteredAt  time.Time `json:"-"`
}

// SignedEnvelope is an encrypted, signed signaling message between two devices.
type SignedEnvelope struct {
	FromDeviceID string `json:"from_device_id"`
	ToDeviceID   string `json:"to_device_id"`
	MessageID    string `json:"message_id"`
	Timestamp    int64  `json:"timestamp"`
	Nonce        []byte `json:"nonce"`
	Ciphertext   []byte `json:"ciphertext"`
	Signature    []byte `json:"signature"`
}

type ClientConfig struct {
	ServerURL     string
	DeviceID      string
	DeviceName    string
	Fingerprint   string
	ListenPort    int
	PrivateAddrs  []string
	RelayEnabled  bool
	RelayMaxBytes int64
}

type Client struct {
	cfg ClientConfig
}

func NewClient(cfg ClientConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Config() ClientConfig {
	return c.cfg
}

func (c *Client) Connect(timeout time.Duration) error {
	// Will be replaced with real WSS implementation later
	return net.ErrClosed
}
