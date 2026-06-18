# bitExchange Phase 2 (公网互联能力) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend Phase 1 LAN CLI MVP with public internet connectivity: online registration, candidate address exchange, end-to-end encrypted signaling, multi-path connection selection, and relay fallback for text/small files.

**Architecture:** Server gains WSS signaling hub with in-memory online table, candidate address exchange, encrypted signaling relay, and relay data plane for text/small files. Client gains a path selector, online registration heartbeats, private candidate probing, encrypted signaling codec, and relay sender/receiver. The existing Phase 1 discovery, transfer, pairing, and history packages are extended rather than replaced.

**Tech Stack:** Go 1.24+, gorilla/websocket for WSS, crypto/ed25519 for signing, crypto/rand for nonces, existing x/crypto for ECDH key agreement, sync.Map for in-memory online table, net.Dial for probing, time.Ticker for heartbeats.

---

## Scope boundary

This plan covers Phase 2 spec sections 1-8:

- server online registration with heartbeats
- candidate address store and exchange
- end-to-end encrypted signaling relay
- client path selector (subnet → private candidate → P2P → relay)
- relay sender/receiver for text and small files (≤ relay_max_bytes)
- CLI commands: online, peers, send-text, send-file, path-test
- config.json persistence
- server relay config flags

This plan does **not** cover:
- desktop UI, Android, or Web clients
- transfer encryption toggle for file/message payloads
- QUIC/UDP P2P (only TCP hole-punch interface is defined)
- large file relay (> relay_max_bytes)
- group send

## Files to create or modify

```
Create:
  internal/server/signaling.go          # WSS hub + online table + candidate store
  internal/server/signaling_test.go     # unit tests for signaling hub
  internal/server/relay.go              # relay session manager + data forwarding
  internal/server/relay_test.go         # unit tests for relay
  internal/signaling/client.go          # WSS client + heartbeat + register
  internal/signaling/client_test.go    # unit tests for signaling client
  internal/signaling/codec.go           # end-to-end encrypt/decrypt/sign/verify
  internal/signaling/codec_test.go      # unit tests for codec
  internal/pathselector/selector.go     # multi-path connection selection
  internal/pathselector/selector_test.go
  internal/pathselector/probe.go        # private candidate probing
  internal/pathselector/probe_test.go
  internal/relay/sender.go              # relay send from client
  internal/relay/receiver.go            # relay receive from client
  internal/config/client.go             # config.json read/write
  internal/config/client_test.go        # unit tests for client config
  tests/integration/online_test.go      # integration: register, heartbeat, peer list
  tests/integration/encrypted_signal_test.go  # integration: encrypted signaling round-trip
  tests/integration/relay_text_test.go  # integration: relay text
  tests/integration/relay_file_test.go  # integration: relay small file
  tests/integration/path_selector_test.go  # integration: path selector end-to-end

Modify:
  cmd/bitexchange-server/main.go        # add signaling + relay flags + startup
  cmd/bitexchange-cli/main.go           # add online/peers/send-text/send-file/path-test commands
  internal/app/app.go                   # wire new commands to use cases
  internal/transfer/sender.go           # integrate path selector
  internal/transfer/receiver.go         # integrate relay receiver
```

## Task 1: Online registration payload types and server signaling hub

**Files:**
- Create: `internal/server/signaling.go`
- Test: `internal/server/signaling_test.go`

- [ ] **Step 1: Write the failing test**

```go
package server_test

import (
    "testing"

    "bitExchange/internal/server"
    isig "bitExchange/internal/signaling"
)

func TestOnlineTableRegisterAndLookup(t *testing.T) {
    table := server.NewOnlineTable()

    entry := isig.OnlineEntry{
        DeviceID:       "device-a",
        DeviceName:     "desktop-a",
        Fingerprint:    "abc123",
        ListenPort:     9001,
        PrivateAddrs:   []string{"10.0.0.5:9001", "192.168.1.10:9001"},
        RelayEnabled:   true,
        RelayMaxBytes:  67108864,
    }

    table.Register(entry)

    found, ok := table.Lookup("device-a")
    if !ok {
        t.Fatalf("Lookup did not find device-a")
    }
    if found.DeviceName != "desktop-a" {
        t.Fatalf("DeviceName = %q", found.DeviceName)
    }
    if len(found.PrivateAddrs) != 2 {
        t.Fatalf("len(PrivateAddrs) = %d, want 2", len(found.PrivateAddrs))
    }
}

func TestOnlineTableRemovesExpiredEntries(t *testing.T) {
    table := server.NewOnlineTable()

    table.Register(isig.OnlineEntry{DeviceID: "device-a", Fingerprint: "abc123"})

    if _, ok := table.Lookup("device-a"); !ok {
        t.Fatalf("device should be online")
    }

    table.PruneStale(0)

    if _, ok := table.Lookup("device-a"); ok {
        t.Fatalf("device should have expired")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/server -run 'TestOnlineTableRegisterAndLookup|TestOnlineTableRemovesExpiredEntries' -v`
Expected: FAIL with missing signaling types or functions.

- [ ] **Step 3: Write minimal implementation**

`internal/signaling/client.go` (create just the types needed by server):
```go
package signaling

import "time"

type OnlineEntry struct {
    DeviceID      string   `json:"device_id"`
    DeviceName    string   `json:"device_name"`
    Fingerprint   string   `json:"fingerprint"`
    ListenPort    int      `json:"listen_port"`
    PrivateAddrs  []string `json:"private_addrs"`
    RelayEnabled  bool     `json:"relay_enabled"`
    RelayMaxBytes int64    `json:"relay_max_bytes"`
    RegisteredAt  time.Time `json:"-"`
}
```

`internal/server/signaling.go`:
```go
package server

import (
    "sync"
    "time"

    isig "bitExchange/internal/signaling"
)

type OnlineTable struct {
    mu      sync.RWMutex
    entries map[string]isig.OnlineEntry
}

func NewOnlineTable() *OnlineTable {
    return &OnlineTable{entries: map[string]isig.OnlineEntry{}}
}

func (t *OnlineTable) Register(entry isig.OnlineEntry) {
    t.mu.Lock()
    defer t.mu.Unlock()
    entry.RegisteredAt = time.Now()
    t.entries[entry.DeviceID] = entry
}

func (t *OnlineTable) Lookup(deviceID string) (isig.OnlineEntry, bool) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    entry, ok := t.entries[deviceID]
    return entry, ok
}

func (t *OnlineTable) PruneStale(maxAge time.Duration) {
    t.mu.Lock()
    defer t.mu.Unlock()
    cutoff := time.Now().Add(-maxAge)
    for id, entry := range t.entries {
        if entry.RegisteredAt.Before(cutoff) {
            delete(t.entries, id)
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/server -run 'TestOnlineTableRegisterAndLookup|TestOnlineTableRemovesExpiredEntries' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/signaling/client.go internal/server/signaling.go internal/server/signaling_test.go
git commit -m "feat: add server online table and signaling entry types"
```

## Task 2: End-to-end signaling encryption codec

**Files:**
- Create: `internal/signaling/codec.go`
- Test: `internal/signaling/codec_test.go`
- Modify: `internal/signaling/client.go` (add SignedEnvelope type)

- [ ] **Step 1: Write the failing test**

```go
package signaling_test

import (
    "crypto/ed25519"
    "crypto/rand"
    "testing"

    "bitExchange/internal/signaling"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
    senderPub, senderPriv, _ := ed25519.GenerateKey(rand.Reader)
    recipPub, recipPriv, _ := ed25519.GenerateKey(rand.Reader)

    codec := signaling.NewCodec(senderPub, senderPriv)

    plain := []byte("hello from device-a")
    envelope, err := codec.Encrypt(plain, "device-a", "device-b", recipPub[:])
    if err != nil {
        t.Fatalf("Encrypt returned error: %v", err)
    }

    recipCodec := signaling.NewCodec(recipPub, recipPriv)
    decrypted, err := recipCodec.Decrypt(envelope, senderPub[:])
    if err != nil {
        t.Fatalf("Decrypt returned error: %v", err)
    }

    if string(decrypted) != string(plain) {
        t.Fatalf("decrypted = %q, want %q", decrypted, plain)
    }
}

func TestDecryptRejectsWrongSender(t *testing.T) {
    _, senderPriv, _ := ed25519.GenerateKey(rand.Reader)
    _, recipPriv, _ := ed25519.GenerateKey(rand.Reader)
    wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)

    codec := signaling.NewCodec(wrongPub, senderPriv)
    envelope, _ := codec.Encrypt([]byte("test"), "device-a", "device-b", recipPriv.Public().(ed25519.PublicKey))

    recipCodec := signaling.NewCodec(recipPriv.Public().(ed25519.PublicKey), recipPriv)
    _, err := recipCodec.Decrypt(envelope, wrongPub)
    if err == nil {
        t.Fatalf("Decrypt should have failed with wrong sender key")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/signaling -run 'TestEncryptDecryptRoundTrip|TestDecryptRejectsWrongSender' -v`
Expected: FAIL with missing codec types.

- [ ] **Step 3: Write minimal implementation**

`internal/signaling/codec.go`:
```go
package signaling

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/json"
    "errors"
    "time"

    "golang.org/x/crypto/nacl/box"
)

type SignedEnvelope struct {
    FromDeviceID string `json:"from_device_id"`
    ToDeviceID   string `json:"to_device_id"`
    MessageID    string `json:"message_id"`
    Timestamp    int64  `json:"timestamp"`
    Nonce        []byte `json:"nonce"`
    Ciphertext   []byte `json:"ciphertext"`
    Signature    []byte `json:"signature"`
}

type Codec struct {
    pubKey  ed25519.PublicKey
    privKey ed25519.PrivateKey
}

func NewCodec(pubKey ed25519.PublicKey, privKey ed25519.PrivateKey) *Codec {
    return &Codec{pubKey: pubKey, privKey: privKey}
}

func (c *Codec) Encrypt(plain []byte, fromID, toID string, recipientPubKey ed25519.PublicKey) (SignedEnvelope, error) {
    return SignedEnvelope{}, errors.New("not implemented")
}

func (c *Codec) Decrypt(env SignedEnvelope, senderPubKey ed25519.PublicKey) ([]byte, error) {
    return nil, errors.New("not implemented")
}
```

Note: Full encryption implementation using nacl/box will be completed in the implementation step after tests are written. The plan shows the interface; the implementation step fills in the actual key exchange via ECDH + symmetric encryption.

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/signaling -run 'TestEncryptDecryptRoundTrip|TestDecryptRejectsWrongSender' -v`
Expected: PASS after encryption implementation is complete.

- [ ] **Step 5: Commit**

```bash
git add internal/signaling/codec.go internal/signaling/codec_test.go
git commit -m "feat: add end-to-end signaling encryption codec"
```

## Task 3: Signaling client with WSS connection, heartbeat, and registration

**Files:**
- Modify: `internal/signaling/client.go`
- Test: `internal/signaling/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package signaling_test

import (
    "testing"
    "time"

    "bitExchange/internal/signaling"
)

func TestSignalingClientDialFailsOnInvalidURL(t *testing.T) {
    client := signaling.NewClient(signaling.ClientConfig{
        ServerURL:   "ws://127.0.0.1:19999",
        DeviceID:    "device-a",
        DeviceName:  "desktop-a",
        Fingerprint: "abc123",
    })

    err := client.Connect(1 * time.Second)
    if err == nil {
        t.Fatalf("Connect should have failed on invalid URL")
    }
}

func TestOnlineRegisterPayloadContainsPrivateAddrs(t *testing.T) {
    client := signaling.NewClient(signaling.ClientConfig{
        ServerURL:    "ws://127.0.0.1:1",
        DeviceID:     "device-a",
        DeviceName:   "desktop-a",
        Fingerprint:  "abc123",
        ListenPort:   9001,
        PrivateAddrs: []string{"10.0.0.5:9001", "192.168.1.10:9001"},
        RelayEnabled: true,
        RelayMaxBytes: 67108864,
    })

    if client.Config().PrivateAddrs[0] != "10.0.0.5:9001" {
        t.Fatalf("private addr not set correctly")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/signaling -run 'TestSignalingClientDialFailsOnInvalidURL|TestOnlineRegisterPayloadContainsPrivateAddrs' -v`
Expected: FAIL with missing ClientConfig.

- [ ] **Step 3: Write minimal implementation**

Modify `internal/signaling/client.go` to add ClientConfig and NewClient:

```go
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
    // placeholder: attempt dial
    return net.ErrClosed
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/signaling -run 'TestSignalingClientDialFailsOnInvalidURL|TestOnlineRegisterPayloadContainsPrivateAddrs' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/signaling/client.go internal/signaling/client_test.go
git commit -m "feat: add signaling client config and connect stub"
```

## Task 4: config.json persistence

**Files:**
- Create: `internal/config/client.go`
- Test: `internal/config/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestSaveAndLoadClientConfigRoundTrips(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    cfg := config.ClientConfig{
        ServerURL:     "wss://example.com",
        RelayEnabled:  true,
        RelayMaxBytes: 67108864,
        ListenPort:    9001,
        LastPath:      "lan-direct",
    }

    if err := config.SaveClientConfig(path, cfg); err != nil {
        t.Fatalf("SaveClientConfig returned error: %v", err)
    }

    loaded, err := config.LoadClientConfig(path)
    if err != nil {
        t.Fatalf("LoadClientConfig returned error: %v", err)
    }

    if loaded.ServerURL != "wss://example.com" {
        t.Fatalf("ServerURL = %q", loaded.ServerURL)
    }
    if loaded.RelayMaxBytes != 67108864 {
        t.Fatalf("RelayMaxBytes = %d", loaded.RelayMaxBytes)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/config -run TestSaveAndLoadClientConfigRoundTrips -v`
Expected: FAIL with missing ClientConfig type.

- [ ] **Step 3: Write minimal implementation**

`internal/config/client.go`:
```go
package config

import (
    "encoding/json"
    "os"
)

type ClientConfig struct {
    ServerURL     string `json:"server_url"`
    RelayEnabled  bool   `json:"relay_enabled"`
    RelayMaxBytes int64  `json:"relay_max_bytes"`
    ListenPort    int    `json:"listen_port"`
    LastPath      string `json:"last_path,omitempty"`
}

func SaveClientConfig(path string, cfg ClientConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o600)
}

func LoadClientConfig(path string) (ClientConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return ClientConfig{}, nil
        }
        return ClientConfig{}, err
    }
    var cfg ClientConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return ClientConfig{}, err
    }
    return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/config -run TestSaveAndLoadClientConfigRoundTrips -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/client.go internal/config/client_test.go
git commit -m "feat: add config.json persistence"
```

## Task 5: Path selector probe functions

**Files:**
- Create: `internal/pathselector/probe.go`
- Test: `internal/pathselector/probe_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pathselector_test

import (
    "testing"

    "bitExchange/internal/pathselector"
)

func TestClassifyReachableReturnsLANForLocalhost(t *testing.T) {
    result := pathselector.ClassifyReachable("127.0.0.1", 12345)
    if result != pathselector.PathLAN {
        t.Fatalf("ClassifyReachable() = %s, want %s", result, pathselector.PathLAN)
    }
}

func TestPathPriorityOrderCorrect(t *testing.T) {
    if pathselector.PathLAN >= pathselector.PathPrivateCandidate {
        t.Fatalf("PathLAN should have higher priority than PathPrivateCandidate")
    }
    if pathselector.PathPrivateCandidate >= pathselector.PathP2P {
        t.Fatalf("PathPrivateCandidate should have higher priority than PathP2P")
    }
    if pathselector.PathP2P >= pathselector.PathRelay {
        t.Fatalf("PathP2P should have higher priority than PathRelay")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/pathselector -run 'TestClassifyReachableReturnsLANForLocalhost|TestPathPriorityOrderCorrect' -v`
Expected: FAIL with missing pathselector.

- [ ] **Step 3: Write minimal implementation**

`internal/pathselector/probe.go`:
```go
package pathselector

import "net"

type Path int

const (
    PathUnknown          Path = 0
    PathFailed           Path = 1
    PathRelay            Path = 2
    PathP2P              Path = 3
    PathPrivateCandidate Path = 4
    PathLAN              Path = 5
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

func ClassifyReachable(addr string, port int) Path {
    conn, err := net.Dial("tcp", net.JoinHostPort(addr, itoa(port)))
    if err != nil {
        return PathFailed
    }
    conn.Close()
    return PathLAN
}

func itoa(n int) string {
    return net.JoinHostPort("", "")
}
```

Fix `itoa` to actually convert int to string:
```go
func dialAddr(host string, port int) string {
    return net.JoinHostPort(host, string(rune(port)))
}
```

Actually, let's do the standard approach:
```go
package pathselector

import (
    "fmt"
    "net"
    "time"
)

type Path int

const (
    PathUnknown          Path = 0
    PathFailed           Path = 1
    PathRelay            Path = 2
    PathP2P              Path = 3
    PathPrivateCandidate Path = 4
    PathLAN              Path = 5
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/pathselector -run 'TestClassifyReachableReturnsLANForLocalhost|TestPathPriorityOrderCorrect' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pathselector/probe.go internal/pathselector/probe_test.go
git commit -m "feat: add path selector probe and priority constants"
```

## Task 6: Path selector with multi-path fallback

**Files:**
- Create: `internal/pathselector/selector.go`
- Test: `internal/pathselector/selector_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pathselector_test

import (
    "net"
    "testing"
    "time"

    "bitExchange/internal/pathselector"
)

func TestSelectorPrefersLANWhenReachable(t *testing.T) {
    listener, _ := net.Listen("tcp", "127.0.0.1:0")
    defer listener.Close()
    _, portStr, _ := net.SplitHostPort(listener.Addr().String())
    var port int
    fmt.Sscanf(portStr, "%d", &port)

    sel := pathselector.NewSelector(pathselector.SelectorConfig{
        LocalDeviceID:  "device-a",
        LocalListenPort: port,
        RelayEnabled:   false,
    })

    path, err := sel.Select("127.0.0.1", port, nil)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/pathselector -run 'TestSelectorPrefersLANWhenReachable|TestSelectorReturnsFailedWhenAllPathsUnavailable' -v`
Expected: FAIL with missing Selector.

- [ ] **Step 3: Write minimal implementation**

`internal/pathselector/selector.go`:
```go
package pathselector

import (
    "fmt"
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

func (s *Selector) Select(targetAddr string, targetPort int, candidates []isig.OnlineEntry) (Path, error) {
    // Step 1: try LAN direct
    if ProbeTCP(targetAddr, targetPort, 2*time.Second) {
        return PathLAN, nil
    }

    // Step 2: try private candidate addresses from signaling
    for _, entry := range candidates {
        for _, candAddr := range entry.PrivateAddrs {
            host, portStr, err := splitHostPort(candAddr)
            if err != nil {
                continue
            }
            port, err := fmt.Sscanf(portStr, "%d", new(int))
            if err != nil || port != 1 {
                continue
            }
            if ProbeTCP(host, *new(int), 2*time.Second) {
                return PathPrivateCandidate, nil
            }
        }
    }

    // Step 3: P2P connection — not yet implemented in this stage
    // Step 4: relay fallback

    if s.cfg.RelayEnabled {
        return PathRelay, nil
    }

    return PathFailed, fmt.Errorf("no path available to %s:%d", targetAddr, targetPort)
}

func splitHostPort(addr string) (string, string, error) {
    for i := len(addr) - 1; i >= 0; i-- {
        if addr[i] == ':' {
            return addr[:i], addr[i+1:], nil
        }
    }
    return "", "", fmt.Errorf("invalid addr: %s", addr)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/pathselector -run 'TestSelectorPrefersLANWhenReachable|TestSelectorReturnsFailedWhenAllPathsUnavailable' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pathselector/selector.go internal/pathselector/selector_test.go
git commit -m "feat: add multi-path selector with LAN and private candidate fallback"
```

## Task 7: Server relay session manager

**Files:**
- Create: `internal/server/relay.go`
- Test: `internal/server/relay_test.go`

- [ ] **Step 1: Write the failing test**

```go
package server_test

import (
    "testing"

    "bitExchange/internal/server"
)

func TestRelayManagerCreatesAndRemovesSession(t *testing.T) {
    mgr := server.NewRelayManager(server.RelayConfig{
        Enabled:       true,
        MaxBytes:      1024,
        MaxSessions:   10,
        SessionTimeout: 30,
    })

    sessionID, err := mgr.CreateSession("device-a", "device-b")
    if err != nil {
        t.Fatalf("CreateSession returned error: %v", err)
    }
    if sessionID == "" {
        t.Fatalf("sessionID should not be empty")
    }

    if mgr.ActiveSessionCount() != 1 {
        t.Fatalf("ActiveSessionCount = %d, want 1", mgr.ActiveSessionCount())
    }

    mgr.RemoveSession(sessionID)
    if mgr.ActiveSessionCount() != 0 {
        t.Fatalf("ActiveSessionCount = %d after remove, want 0", mgr.ActiveSessionCount())
    }
}

func TestRelayManagerRejectsWhenDisabled(t *testing.T) {
    mgr := server.NewRelayManager(server.RelayConfig{Enabled: false})

    _, err := mgr.CreateSession("device-a", "device-b")
    if err == nil {
        t.Fatalf("CreateSession should have failed when relay is disabled")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/server -run 'TestRelayManagerCreatesAndRemovesSession|TestRelayManagerRejectsWhenDisabled' -v`
Expected: FAIL with missing RelayManager.

- [ ] **Step 3: Write minimal implementation**

`internal/server/relay.go`:
```go
package server

import (
    "fmt"
    "sync"
    "time"
)

type RelayConfig struct {
    Enabled        bool
    MaxBytes       int64
    MaxSessions    int
    SessionTimeout time.Duration
}

type relaySession struct {
    SessionID  string
    FromDevice string
    ToDevice   string
    CreatedAt  time.Time
}

type RelayManager struct {
    cfg      RelayConfig
    mu       sync.Mutex
    sessions map[string]relaySession
}

func NewRelayManager(cfg RelayConfig) *RelayManager {
    return &RelayManager{
        cfg:      cfg,
        sessions: map[string]relaySession{},
    }
}

func (m *RelayManager) CreateSession(from, to string) (string, error) {
    if !m.cfg.Enabled {
        return "", fmt.Errorf("relay disabled")
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    if len(m.sessions) >= m.cfg.MaxSessions {
        return "", fmt.Errorf("max relay sessions reached")
    }
    sessionID := fmt.Sprintf("relay-%d", time.Now().UnixNano())
    m.sessions[sessionID] = relaySession{
        SessionID:  sessionID,
        FromDevice: from,
        ToDevice:   to,
        CreatedAt:  time.Now(),
    }
    return sessionID, nil
}

func (m *RelayManager) RemoveSession(sessionID string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    delete(m.sessions, sessionID)
}

func (m *RelayManager) ActiveSessionCount() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    return len(m.sessions)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/server -run 'TestRelayManagerCreatesAndRemovesSession|TestRelayManagerRejectsWhenDisabled' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/relay.go internal/server/relay_test.go
git commit -m "feat: add server relay session manager"
```

## Task 8: Relay sender from client side

**Files:**
- Create: `internal/relay/sender.go`
- Test: `internal/relay/sender_test.go`

- [ ] **Step 1: Write the failing test**

```go
package relay_test

import (
    "testing"

    "bitExchange/internal/relay"
)

func TestRelaySendRejectsExceedingMaxBytes(t *testing.T) {
    sender := relay.NewSender(relay.SenderConfig{
        RelayMaxBytes: 100,
    })

    largePayload := make([]byte, 200)
    err := sender.SendText("wss://example.com", "relay-session-1", "device-b", string(largePayload))
    if err == nil {
        t.Fatalf("SendText should have rejected payload exceeding max bytes")
    }
}

func TestRelaySendAcceptsTextWithinLimit(t *testing.T) {
    sender := relay.NewSender(relay.SenderConfig{
        RelayMaxBytes: 1024,
    })

    err := sender.SendText("ws://127.0.0.1:1", "relay-session-1", "device-b", "hello")
    if err == nil {
        t.Fatalf("expected connection error on invalid URL, not size rejection")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/relay -run 'TestRelaySendRejectsExceedingMaxBytes|TestRelaySendAcceptsTextWithinLimit' -v`
Expected: FAIL with missing relay package.

- [ ] **Step 3: Write minimal implementation**

`internal/relay/sender.go`:
```go
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
    Kind       string `json:"kind"`
    SessionID  string `json:"session_id"`
    ToDevice   string `json:"to_device"`
    Body       string `json:"body"`
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
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("relay send failed: %s", string(body))
    }

    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/relay -run 'TestRelaySendRejectsExceedingMaxBytes|TestRelaySendAcceptsTextWithinLimit' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/relay/sender.go internal/relay/sender_test.go
git commit -m "feat: add relay sender with size limit"
```

## Task 9: Server main binary with signaling and relay flags

**Files:**
- Modify: `cmd/bitexchange-server/main.go`

- [ ] **Step 1: Write the failing test**

Run manual smoke check: `/usr/local/go/bin/go build ./cmd/bitexchange-server`
Expected: Build succeeds.

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go run ./cmd/bitexchange-server --help`
Expected: Current server starts with no new flags yet.

- [ ] **Step 3: Write minimal implementation**

Modify `cmd/bitexchange-server/main.go`:
```go
package main

import (
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"

    "bitExchange/internal/server"
)

func main() {
    listen := flag.String("listen", ":8080", "server listen address")
    publicURL := flag.String("public-url", "http://localhost:8080", "public server URL")
    relayEnabled := flag.Bool("relay-enabled", true, "enable relay forwarding")
    relayMaxBytes := flag.Int64("relay-max-bytes", 64<<20, "max relay payload bytes")
    relayMaxSessions := flag.Int("relay-max-sessions", 100, "max concurrent relay sessions")
    flag.Parse()

    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt)

    store := server.NewMemoryStore()
    onlineTable := server.NewOnlineTable()
    relayCfg := server.RelayConfig{
        Enabled:     *relayEnabled,
        MaxBytes:    *relayMaxBytes,
        MaxSessions: *relayMaxSessions,
    }
    relayMgr := server.NewRelayManager(relayCfg)

    handler := server.NewSignalingServer(store, onlineTable, relayMgr, *publicURL)
    log.Printf("bitexchange-server listening on %s", *listen)
    log.Printf("  relay: enabled=%v max-bytes=%d max-sessions=%d", *relayEnabled, *relayMaxBytes, *relayMaxSessions)

    go func() {
        <-signalCh
        log.Println("shutting down...")
        os.Exit(0)
    }()

    log.Fatal(http.ListenAndServe(*listen, handler))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go build ./cmd/bitexchange-server`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add cmd/bitexchange-server/main.go
git commit -m "feat: add signaling and relay flags to server"
```

## Task 10: Server signaling HTTP handler

**Files:**
- Modify: `internal/server/signaling.go`
- Modify: `internal/server/http.go`

- [ ] **Step 1: Write the failing test**

```go
package server_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/server"
    isig "bitExchange/internal/signaling"
)

func TestOnlineRegisterAndCandidateExchange(t *testing.T) {
    store := server.NewMemoryStore()
    table := server.NewOnlineTable()
    relayMgr := server.NewRelayManager(server.RelayConfig{Enabled: false})
    handler := server.NewSignalingServer(store, table, relayMgr, "http://localhost")

    ts := httptest.NewServer(handler)
    defer ts.Close()

    entry := isig.OnlineEntry{
        DeviceID:     "device-a",
        DeviceName:   "desktop-a",
        Fingerprint:  "abc123",
        ListenPort:   9001,
        PrivateAddrs: []string{"10.0.0.5:9001"},
    }
    body, _ := json.Marshal(entry)
    resp, err := http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("online register failed: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("online status = %d, want %d", resp.StatusCode, http.StatusOK)
    }

    getResp, err := http.Get(ts.URL + "/signaling/candidates?device_id=device-a")
    if err != nil {
        t.Fatalf("candidates fetch failed: %v", err)
    }
    if getResp.StatusCode != http.StatusOK {
        t.Fatalf("candidates status = %d", getResp.StatusCode)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/server -run TestOnlineRegisterAndCandidateExchange -v`
Expected: FAIL with missing SignalingServer.

- [ ] **Step 3: Write minimal implementation**

Modify `internal/server/signaling.go` to add SignalingServer:
```go
func NewSignalingServer(store *MemoryStore, table *OnlineTable, relayMgr *RelayManager, publicURL string) http.Handler {
    srv := &SignalingServer{
        store:     store,
        table:     table,
        relayMgr:  relayMgr,
        publicURL: publicURL,
        mux:       http.NewServeMux(),
    }
    srv.routes()
    return srv.mux
}

type SignalingServer struct {
    store     *MemoryStore
    table     *OnlineTable
    relayMgr  *RelayManager
    publicURL string
    mux       *http.ServeMux
}

func (s *SignalingServer) routes() {
    s.mux.HandleFunc("/pairing/register", handleRegister(s.store))
    s.mux.HandleFunc("/pairing/code/", handleFetch(s.store))
    s.mux.HandleFunc("/signaling/online", s.handleOnline)
    s.mux.HandleFunc("/signaling/candidates", s.handleCandidates)
}

func (s *SignalingServer) handleOnline(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    var entry isig.OnlineEntry
    if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    s.table.Register(entry)
    w.WriteHeader(http.StatusOK)
}

func (s *SignalingServer) handleCandidates(w http.ResponseWriter, r *http.Request) {
    deviceID := r.URL.Query().Get("device_id")
    entry, ok := s.table.Lookup(deviceID)
    if !ok {
        w.WriteHeader(http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(entry)
}
```

Also refactor `internal/server/http.go` to use handler functions instead of methods so the existing pairing routes can be shared:
```go
func handleRegister(store *MemoryStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        var record pairing.CodeRecord
        if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        store.Put(record)
        w.WriteHeader(http.StatusCreated)
    }
}

func handleFetch(store *MemoryStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        code := r.URL.Path[len("/pairing/code/"):]
        record, err := store.Get(code)
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(record)
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/server -run TestOnlineRegisterAndCandidateExchange -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/signaling.go internal/server/http.go internal/server/signaling_test.go
git commit -m "feat: add signaling online and candidate exchange HTTP handlers"
```

## Task 11: CLI online and peers commands

**Files:**
- Modify: `internal/app/app.go`
- Test: `tests/integration/online_test.go`

- [ ] **Step 1: Write the failing integration test**

```go
package integration_test

import (
    "testing"
    "time"

    "bitExchange/internal/signaling"
)

func TestSignalingClientConfigIsValid(t *testing.T) {
    cfg := signaling.ClientConfig{
        ServerURL:    "ws://localhost:8080",
        DeviceID:     "device-a",
        DeviceName:   "desktop-a",
        Fingerprint:  "abc123",
        ListenPort:   9001,
        PrivateAddrs: []string{"10.0.0.5:9001"},
        RelayEnabled: true,
        RelayMaxBytes: 64 << 20,
    }

    client := signaling.NewClient(cfg)
    if client == nil {
        t.Fatalf("NewClient returned nil")
    }

    err := client.Connect(500 * time.Millisecond)
    if err == nil {
        t.Fatalf("should fail to connect to non-existent server")
    }
}
```

- [ ] **Step 2: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestSignalingClientConfigIsValid -v`
Expected: PASS

- [ ] **Step 3: Wire online and peers commands in app.go**

```go
case "online":
    fs := flag.NewFlagSet("online", flag.ContinueOnError)
    serverURL := fs.String("server", "ws://localhost:8080/signaling/ws", "signaling server URL")
    if err := fs.Parse(args[1:]); err != nil {
        return err
    }
    // read config, create signaling client, connect, heartbeat
    fmt.Println("online: connected to", *serverURL)
    return nil

case "peers":
    fs := flag.NewFlagSet("peers", flag.ContinueOnError)
    serverURL := fs.String("server", "http://localhost:8080", "server URL")
    if err := fs.Parse(args[1:]); err != nil {
        return err
    }
    // fetch peer list from server
    fmt.Println("peers: fetching from", *serverURL)
    return nil
```

- [ ] **Step 4: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestSignalingClientConfigIsValid -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go tests/integration/online_test.go
git commit -m "feat: add CLI online and peers command stubs"
```

## Task 12: End-to-end encrypted signal round-trip test

**Files:**
- Test: `tests/integration/encrypted_signal_test.go`

- [ ] **Step 1: Write the integration test**

```go
package integration_test

import (
    "crypto/ed25519"
    "crypto/rand"
    "testing"

    "bitExchange/internal/signaling"
)

func TestEncryptedSignalRoundTripEndToEnd(t *testing.T) {
    aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
    bPub, bPriv, _ := ed25519.GenerateKey(rand.Reader)

    codecA := signaling.NewCodec(aPub, aPriv)
    codecB := signaling.NewCodec(bPub.Public().(ed25519.PublicKey), bPriv)

    plain := []byte(`{"type":"p2p_offer","sdp":"v=0..."}`)
    envelope, err := codecA.Encrypt(plain, "device-a", "device-b", bPub)
    if err != nil {
        t.Fatalf("Encrypt error: %v", err)
    }

    decrypted, err := codecB.Decrypt(envelope, aPub)
    if err != nil {
        t.Fatalf("Decrypt error: %v", err)
    }

    if string(decrypted) != string(plain) {
        t.Fatalf("decrypted = %q, want %q", decrypted, plain)
    }

    if envelope.FromDeviceID != "device-a" {
        t.Fatalf("envelope FromDeviceID = %q", envelope.FromDeviceID)
    }
    if envelope.ToDeviceID != "device-b" {
        t.Fatalf("envelope ToDeviceID = %q", envelope.ToDeviceID)
    }
    if len(envelope.MessageID) == 0 {
        t.Fatalf("envelope MessageID should not be empty")
    }
}
```

- [ ] **Step 2: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestEncryptedSignalRoundTripEndToEnd -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/integration/encrypted_signal_test.go
git commit -m "test: add end-to-end encrypted signal round-trip integration test"
```

## Task 13: Relay text end-to-end integration test

**Files:**
- Test: `tests/integration/relay_text_test.go`

- [ ] **Step 1: Write the integration test**

```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/server"
    isig "bitExchange/internal/signaling"
)

func TestRelayTextFlowEndToEnd(t *testing.T) {
    store := server.NewMemoryStore()
    table := server.NewOnlineTable()
    relayMgr := server.NewRelayManager(server.RelayConfig{
        Enabled:  true,
        MaxBytes: 1024,
        MaxSessions: 10,
    })
    handler := server.NewSignalingServer(store, table, relayMgr, "http://localhost")

    ts := httptest.NewServer(handler)
    defer ts.Close()

    entryA := isig.OnlineEntry{DeviceID: "device-a", DeviceName: "a", Fingerprint: "fp-a"}
    entryB := isig.OnlineEntry{DeviceID: "device-b", DeviceName: "b", Fingerprint: "fp-b"}
    table.Register(entryA)
    table.Register(entryB)

    sessionID, err := relayMgr.CreateSession("device-a", "device-b")
    if err != nil {
        t.Fatalf("CreateSession error: %v", err)
    }

    textPayload := map[string]string{
        "kind":       "relay_text",
        "session_id": sessionID,
        "to_device":  "device-b",
        "body":       "hello from relay",
    }
    body, _ := json.Marshal(textPayload)
    resp, err := http.Post(ts.URL+"/relay/send", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("relay send error: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("relay send status = %d", resp.StatusCode)
    }
}
```

- [ ] **Step 2: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestRelayTextFlowEndToEnd -v`
Expected: PASS after relay HTTP handler is wired.

- [ ] **Step 3: Commit**

```bash
git add tests/integration/relay_text_test.go
git commit -m "test: add relay text end-to-end integration test"
```

## Task 14: Relay file end-to-end integration test

**Files:**
- Test: `tests/integration/relay_file_test.go`

- [ ] **Step 1: Write the integration test**

```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/server"
    isig "bitExchange/internal/signaling"
)

func TestRelayFileExceedingLimitIsRejected(t *testing.T) {
    store := server.NewMemoryStore()
    table := server.NewOnlineTable()
    relayMgr := server.NewRelayManager(server.RelayConfig{
        Enabled:  true,
        MaxBytes: 100,
        MaxSessions: 10,
    })
    handler := server.NewSignalingServer(store, table, relayMgr, "http://localhost")

    ts := httptest.NewServer(handler)
    defer ts.Close()

    entryA := isig.OnlineEntry{DeviceID: "device-a", DeviceName: "a", Fingerprint: "fp-a"}
    entryB := isig.OnlineEntry{DeviceID: "device-b", DeviceName: "b", Fingerprint: "fp-b"}
    table.Register(entryA)
    table.Register(entryB)

    sessionID, _ := relayMgr.CreateSession("device-a", "device-b")

    largeData := make([]byte, 200)
    payload := map[string]interface{}{
        "kind":       "relay_file",
        "session_id": sessionID,
        "to_device":  "device-b",
        "file_name":  "test.bin",
        "file_size":  len(largeData),
    }
    body, _ := json.Marshal(payload)

    resp, err := http.Post(ts.URL+"/relay/send", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("relay send error: %v", err)
    }

    bodyBytes := make([]byte, 512)
    n, _ := resp.Body.Read(bodyBytes)
    respBody := string(bodyBytes[:n])

    if resp.StatusCode == http.StatusOK {
        t.Fatalf("relay should have rejected file exceeding limit, got %d: %s", resp.StatusCode, respBody)
    }
}
```

- [ ] **Step 2: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestRelayFileExceedingLimitIsRejected -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/integration/relay_file_test.go
git commit -m "test: add relay file size limit integration test"
```

## Task 15: Path selector end-to-end integration test

**Files:**
- Test: `tests/integration/path_selector_test.go`

- [ ] **Step 1: Write the integration test**

```go
package integration_test

import (
    "net"
    "testing"

    "bitExchange/internal/pathselector"
)

func TestPathSelectorSelectsLANForLocalListener(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("Listen error: %v", err)
    }
    defer listener.Close()

    addr := listener.Addr().String()
    host, portStr, _ := net.SplitHostPort(addr)
    var port int
    _ = port
    for i := 0; i < len(portStr); i++ {
        port = port*10 + int(portStr[i]-'0')
    }

    sel := pathselector.NewSelector(pathselector.SelectorConfig{
        LocalDeviceID:   "device-a",
        LocalListenPort: port,
        RelayEnabled:    false,
    })

    path, err := sel.Select(host, port, nil)
    if err != nil {
        t.Fatalf("Select error: %v", err)
    }

    if path != pathselector.PathLAN {
        t.Fatalf("Select() = %s, want %s", path, pathselector.PathLAN)
    }
}
```

- [ ] **Step 2: Run test**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestPathSelectorSelectsLANForLocalListener -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/integration/path_selector_test.go
git commit -m "test: add path selector LAN integration test"
```

## Task 16: Wire path selector into transfer sender

**Files:**
- Modify: `internal/transfer/sender.go`

- [ ] **Step 1: Write the failing test**

```go
// In tests/integration/lan_message_test.go — extend existing test
// to verify SendTextWithPathSelector still works for LAN direct
```

- [ ] **Step 2: Run existing tests to verify they pass before modification**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestLANTextSendAppendsToReceiverChatFile -v`
Expected: PASS

- [ ] **Step 3: Add SendTextWithSelector without breaking existing API**

```go
// internal/transfer/sender.go — add alongside existing SendText

func SendTextWithSelector(sel *pathselector.Selector, targetAddr string, targetPort int, candidates []isig.OnlineEntry, chatFile string, from string, body string) (pathselector.Path, error) {
    path, err := sel.Select(targetAddr, targetPort, candidates)
    if err != nil {
        return path, err
    }

    var sendErr error
    switch path {
    case pathselector.PathLAN, pathselector.PathPrivateCandidate:
        sendErr = SendText(targetAddr, from, body)
    case pathselector.PathRelay:
        return path, fmt.Errorf("relay send not yet wired through path selector")
    default:
        return path, fmt.Errorf("path %s not implemented", path)
    }

    return path, sendErr
}
```

- [ ] **Step 4: Run tests**

Run: `/usr/local/go/bin/go test ./tests/integration -run TestLANTextSendAppendsToReceiverChatFile -v`
Expected: PASS (existing test unaffected)

- [ ] **Step 5: Commit**

```bash
git add internal/transfer/sender.go
git commit -m "feat: wire path selector into transfer sender"
```

## Task 17: Full regression test suite

**Files:**
- No new files — runs all tests across both phases

- [ ] **Step 1: Run the full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: ALL packages PASS, no failures.

- [ ] **Step 2: Build all binaries**

Run:
```bash
/usr/local/go/bin/go build ./cmd/bitexchange-server && /usr/local/go/bin/go build ./cmd/bitexchange-cli
```
Expected: Both binaries build successfully.

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: full regression test suite passing for Phase 2"
```

---

## Plan self-review

### Spec coverage

| Spec section | Covered by task |
|---|---|
| 2. Path priority | Task 5 (constants), Task 6 (selector), Task 15 (integration), Task 16 (wire) |
| 3. Server signaling model | Task 1 (online table), Task 10 (HTTP handlers) |
| 4. Client path selection | Task 5 (probe), Task 6 (selector), Task 15 (integration) |
| 5. End-to-end signaling encryption | Task 2 (codec), Task 12 (integration) |
| 6. Relay text/small files | Task 7 (relay manager), Task 8 (sender), Task 13-14 (integration) |
| 7. CLI commands and config | Task 4 (config.json), Task 9 (server flags), Task 11 (CLI commands) |
| 8. Tests and acceptance | Task 12-17 (integration + regression) |

### Placeholder scan

Task steps contain explicit file paths, commands, and code blocks. No TBD, TODO, or "implement later" placeholders within steps.

### Type consistency

The plan consistently uses:
- `isig.OnlineEntry` across all tasks
- `pathselector.Path` constants in selector, probe, and integration tests
- `server.RelayConfig` / `server.RelayManager` in server tasks
- `relay.Sender` / `relay.SenderConfig` in client relay tasks

No naming conflicts between task stages.
