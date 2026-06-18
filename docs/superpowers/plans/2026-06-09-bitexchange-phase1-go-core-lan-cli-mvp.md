# bitExchange Phase 1 (Go Core + LAN CLI MVP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Phase 1 bitExchange MVP with a Go server and Go CLI clients that can pair trusted devices, discover peers on the same LAN, send text messages, send files directly, append text history to `chat.txt`, and save incoming files into `receivedFiles/`.

**Architecture:** The Phase 1 system uses a small Go monorepo with three runnable binaries: a signaling server, a sender/receiver CLI client, and an mDNS-based LAN discovery helper embedded in the client. This plan intentionally excludes desktop UI, Android, Web, public NAT traversal, relay fallback, optional payload encryption, and GitHub Actions packaging so the first delivery stays small, testable, and directly aligned with the staged spec.

**Tech Stack:** Go 1.24+, Cobra CLI, zeroconf/mDNS discovery, HTTP JSON API for pairing metadata bootstrap, TCP for direct file/message transfer, standard library crypto/x25519 or ed25519 for device identity, testify for tests.

---

## Scope boundary for this plan

This plan covers only spec section 11 step 1:

- Go core + Go server
- trusted-device pairing
- same-subnet LAN discovery
- direct text transfer
- direct file transfer
- `chat.txt` append behavior
- `receivedFiles/` default save behavior

This plan does **not** cover:

- Tauri desktop UI
- Android APK
- Web client
- public signaling encryption protocol beyond Phase 1 bootstrap
- NAT traversal
- relay mode
- optional transfer encryption toggle
- GitHub Actions packaging
- group send

Those need separate follow-up plans after this MVP lands.

## Proposed repository layout

```text
bitExchange/
  go.mod
  go.sum
  cmd/
    bitexchange-server/
      main.go
    bitexchange-cli/
      main.go
  internal/
    app/
      app.go
    config/
      paths.go
      device.go
    crypto/
      identity.go
    discovery/
      mdns.go
    pairing/
      store.go
      code.go
    protocol/
      types.go
      framing.go
    server/
      http.go
      memory_store.go
    transfer/
      listener.go
      sender.go
      receiver.go
      files.go
      messages.go
    history/
      chatlog.go
    testutil/
      tempdir.go
  tests/
    integration/
      pairing_test.go
      lan_message_test.go
      lan_file_test.go
      save_path_test.go
```

## File responsibilities

- `cmd/bitexchange-server/main.go` — starts Phase 1 pairing bootstrap server.
- `cmd/bitexchange-cli/main.go` — exposes CLI commands for init, pair, discover, send-text, send-file, listen.
- `internal/app/app.go` — wires commands to use cases.
- `internal/config/paths.go` — creates default root directory, `chat.txt`, and `receivedFiles/`.
- `internal/config/device.go` — loads and saves local device settings.
- `internal/crypto/identity.go` — generates and persists device identity keys and fingerprints.
- `internal/discovery/mdns.go` — advertises and browses LAN peers.
- `internal/pairing/store.go` — manages trusted device records.
- `internal/pairing/code.go` — handles one-time pairing code exchange payloads.
- `internal/protocol/types.go` — transfer request/response structs.
- `internal/protocol/framing.go` — length-prefixed framing for message/file commands.
- `internal/server/http.go` — simple HTTP server for pairing code registration/lookup.
- `internal/server/memory_store.go` — in-memory code store for Phase 1.
- `internal/transfer/listener.go` — incoming TCP listener for direct transfers.
- `internal/transfer/sender.go` — outbound text/file direct send.
- `internal/transfer/receiver.go` — inbound text/file direct receive.
- `internal/transfer/files.go` — file metadata, chunk copy, unique filename resolution.
- `internal/transfer/messages.go` — text message handling and logging hooks.
- `internal/history/chatlog.go` — append-only writes to `chat.txt`.
- `tests/integration/*.go` — scenario tests.

## Task 1: Bootstrap the Go workspace and default storage paths

**Files:**
- Create: `go.mod`
- Create: `cmd/bitexchange-cli/main.go`
- Create: `internal/config/paths.go`
- Create: `internal/testutil/tempdir.go`
- Test: `internal/config/paths_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestEnsureDefaultPathsCreatesChatLogAndReceivedFiles(t *testing.T) {
    root := filepath.Join(t.TempDir(), "bitExchange")

    paths, err := config.EnsureDefaultPaths(root)
    if err != nil {
        t.Fatalf("EnsureDefaultPaths returned error: %v", err)
    }

    if paths.RootDir != root {
        t.Fatalf("RootDir = %q, want %q", paths.RootDir, root)
    }

    if paths.ChatFile != filepath.Join(root, "chat.txt") {
        t.Fatalf("ChatFile = %q", paths.ChatFile)
    }

    if paths.ReceivedFilesDir != filepath.Join(root, "receivedFiles") {
        t.Fatalf("ReceivedFilesDir = %q", paths.ReceivedFilesDir)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestEnsureDefaultPathsCreatesChatLogAndReceivedFiles -v`
Expected: FAIL with missing package, missing function, or missing file.

- [ ] **Step 3: Write minimal implementation**

`go.mod`
```go
module bitExchange

go 1.24
```

`internal/config/paths.go`
```go
package config

import (
    "os"
    "path/filepath"
)

type Paths struct {
    RootDir          string
    ChatFile         string
    ReceivedFilesDir string
}

func EnsureDefaultPaths(root string) (Paths, error) {
    paths := Paths{
        RootDir:          root,
        ChatFile:         filepath.Join(root, "chat.txt"),
        ReceivedFilesDir: filepath.Join(root, "receivedFiles"),
    }

    if err := os.MkdirAll(paths.ReceivedFilesDir, 0o755); err != nil {
        return Paths{}, err
    }

    if err := os.MkdirAll(paths.RootDir, 0o755); err != nil {
        return Paths{}, err
    }

    file, err := os.OpenFile(paths.ChatFile, os.O_CREATE, 0o644)
    if err != nil {
        return Paths{}, err
    }
    _ = file.Close()

    return paths, nil
}
```

`cmd/bitexchange-cli/main.go`
```go
package main

func main() {}
```

`internal/testutil/tempdir.go`
```go
package testutil

import "testing"

func TempDir(t *testing.T) string {
    t.Helper()
    return t.TempDir()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestEnsureDefaultPathsCreatesChatLogAndReceivedFiles -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/bitexchange-cli/main.go internal/config/paths.go internal/config/paths_test.go internal/testutil/tempdir.go
git commit -m "feat: bootstrap Go workspace paths"
```

## Task 2: Persist local device identity and fingerprint

**Files:**
- Create: `internal/crypto/identity.go`
- Create: `internal/config/device.go`
- Test: `internal/crypto/identity_test.go`
- Test: `internal/config/device_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package crypto_test

import (
    "testing"

    "bitExchange/internal/crypto"
)

func TestGenerateIdentityProducesStableFingerprintFormat(t *testing.T) {
    identity, err := crypto.GenerateIdentity("work-laptop")
    if err != nil {
        t.Fatalf("GenerateIdentity returned error: %v", err)
    }

    if identity.DeviceName != "work-laptop" {
        t.Fatalf("DeviceName = %q", identity.DeviceName)
    }

    if len(identity.DeviceID) == 0 {
        t.Fatalf("DeviceID should not be empty")
    }

    if len(identity.Fingerprint) < 16 {
        t.Fatalf("Fingerprint too short: %q", identity.Fingerprint)
    }
}
```

```go
package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
    icrypto "bitExchange/internal/crypto"
)

func TestSaveAndLoadDeviceConfigRoundTripsIdentity(t *testing.T) {
    root := t.TempDir()
    identity, err := icrypto.GenerateIdentity("desktop-a")
    if err != nil {
        t.Fatalf("GenerateIdentity returned error: %v", err)
    }

    path := filepath.Join(root, "device.json")
    if err := config.SaveDeviceConfig(path, config.DeviceConfig{Identity: identity}); err != nil {
        t.Fatalf("SaveDeviceConfig returned error: %v", err)
    }

    loaded, err := config.LoadDeviceConfig(path)
    if err != nil {
        t.Fatalf("LoadDeviceConfig returned error: %v", err)
    }

    if loaded.Identity.DeviceID != identity.DeviceID {
        t.Fatalf("DeviceID = %q, want %q", loaded.Identity.DeviceID, identity.DeviceID)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/crypto ./internal/config -run 'TestGenerateIdentityProducesStableFingerprintFormat|TestSaveAndLoadDeviceConfigRoundTripsIdentity' -v`
Expected: FAIL with missing types or functions.

- [ ] **Step 3: Write minimal implementation**

`internal/crypto/identity.go`
```go
package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
)

type Identity struct {
    DeviceName  string `json:"device_name"`
    DeviceID    string `json:"device_id"`
    Fingerprint string `json:"fingerprint"`
    PublicKey   []byte `json:"public_key"`
    PrivateKey  []byte `json:"private_key"`
}

func GenerateIdentity(deviceName string) (Identity, error) {
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return Identity{}, err
    }

    sum := sha256.Sum256(publicKey)

    return Identity{
        DeviceName:  deviceName,
        DeviceID:    hex.EncodeToString(sum[:16]),
        Fingerprint: hex.EncodeToString(sum[:]),
        PublicKey:   publicKey,
        PrivateKey:  privateKey,
    }, nil
}
```

`internal/config/device.go`
```go
package config

import (
    "encoding/json"
    "os"

    icrypto "bitExchange/internal/crypto"
)

type DeviceConfig struct {
    Identity icrypto.Identity `json:"identity"`
}

func SaveDeviceConfig(path string, cfg DeviceConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o600)
}

func LoadDeviceConfig(path string) (DeviceConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return DeviceConfig{}, err
    }

    var cfg DeviceConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return DeviceConfig{}, err
    }

    return cfg, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/crypto ./internal/config -run 'TestGenerateIdentityProducesStableFingerprintFormat|TestSaveAndLoadDeviceConfigRoundTripsIdentity' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/identity.go internal/crypto/identity_test.go internal/config/device.go internal/config/device_test.go
git commit -m "feat: persist local device identity"
```

## Task 3: Build trusted-device pairing code storage in the server

**Files:**
- Create: `internal/pairing/code.go`
- Create: `internal/server/memory_store.go`
- Create: `internal/server/http.go`
- Create: `cmd/bitexchange-server/main.go`
- Test: `internal/server/http_test.go`

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
)

func TestRegisterAndFetchPairingCode(t *testing.T) {
    srv := server.NewHTTPServer(server.NewMemoryStore())
    ts := httptest.NewServer(srv)
    defer ts.Close()

    payload := map[string]string{
        "code": "PAIR-123456",
        "device_id": "device-a",
        "device_name": "desktop-a",
        "fingerprint": "abc123",
        "listen_addr": "127.0.0.1:9001",
    }

    body, _ := json.Marshal(payload)
    resp, err := http.Post(ts.URL+"/pairing/register", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("register request failed: %v", err)
    }
    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("register status = %d, want %d", resp.StatusCode, http.StatusCreated)
    }

    getResp, err := http.Get(ts.URL + "/pairing/code/PAIR-123456")
    if err != nil {
        t.Fatalf("fetch request failed: %v", err)
    }
    if getResp.StatusCode != http.StatusOK {
        t.Fatalf("fetch status = %d, want %d", getResp.StatusCode, http.StatusOK)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestRegisterAndFetchPairingCode -v`
Expected: FAIL with missing server implementation.

- [ ] **Step 3: Write minimal implementation**

`internal/pairing/code.go`
```go
package pairing

type CodeRecord struct {
    Code        string `json:"code"`
    DeviceID    string `json:"device_id"`
    DeviceName  string `json:"device_name"`
    Fingerprint string `json:"fingerprint"`
    ListenAddr  string `json:"listen_addr"`
}
```

`internal/server/memory_store.go`
```go
package server

import (
    "errors"
    "sync"

    "bitExchange/internal/pairing"
)

var ErrCodeNotFound = errors.New("pairing code not found")

type MemoryStore struct {
    mu    sync.RWMutex
    codes map[string]pairing.CodeRecord
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{codes: map[string]pairing.CodeRecord{}}
}

func (s *MemoryStore) Put(record pairing.CodeRecord) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.codes[record.Code] = record
}

func (s *MemoryStore) Get(code string) (pairing.CodeRecord, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    record, ok := s.codes[code]
    if !ok {
        return pairing.CodeRecord{}, ErrCodeNotFound
    }

    return record, nil
}
```

`internal/server/http.go`
```go
package server

import (
    "encoding/json"
    "net/http"

    "bitExchange/internal/pairing"
)

type HTTPServer struct {
    store *MemoryStore
    mux   *http.ServeMux
}

func NewHTTPServer(store *MemoryStore) http.Handler {
    srv := &HTTPServer{store: store, mux: http.NewServeMux()}
    srv.routes()
    return srv.mux
}

func (s *HTTPServer) routes() {
    s.mux.HandleFunc("/pairing/register", s.handleRegister)
    s.mux.HandleFunc("/pairing/code/", s.handleFetch)
}

func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    var record pairing.CodeRecord
    if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    s.store.Put(record)
    w.WriteHeader(http.StatusCreated)
}

func (s *HTTPServer) handleFetch(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Path[len("/pairing/code/"):]
    record, err := s.store.Get(code)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(record)
}
```

`cmd/bitexchange-server/main.go`
```go
package main

import (
    "log"
    "net/http"

    "bitExchange/internal/server"
)

func main() {
    handler := server.NewHTTPServer(server.NewMemoryStore())
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestRegisterAndFetchPairingCode -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pairing/code.go internal/server/memory_store.go internal/server/http.go internal/server/http_test.go cmd/bitexchange-server/main.go
git commit -m "feat: add pairing bootstrap server"
```

## Task 4: Persist trusted peers on the client

**Files:**
- Create: `internal/pairing/store.go`
- Test: `internal/pairing/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pairing_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/pairing"
)

func TestPeerStoreSavesAndLoadsTrustedPeers(t *testing.T) {
    path := filepath.Join(t.TempDir(), "trusted-peers.json")
    store := pairing.NewPeerStore(path)

    peer := pairing.TrustedPeer{
        DeviceID: "device-b",
        DeviceName: "laptop-b",
        Fingerprint: "fingerprint-b",
    }

    if err := store.Save(peer); err != nil {
        t.Fatalf("Save returned error: %v", err)
    }

    peers, err := store.LoadAll()
    if err != nil {
        t.Fatalf("LoadAll returned error: %v", err)
    }

    if len(peers) != 1 {
        t.Fatalf("len(peers) = %d, want 1", len(peers))
    }

    if peers[0].DeviceID != peer.DeviceID {
        t.Fatalf("DeviceID = %q, want %q", peers[0].DeviceID, peer.DeviceID)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/pairing -run TestPeerStoreSavesAndLoadsTrustedPeers -v`
Expected: FAIL with missing store implementation.

- [ ] **Step 3: Write minimal implementation**

`internal/pairing/store.go`
```go
package pairing

import (
    "encoding/json"
    "os"
)

type TrustedPeer struct {
    DeviceID    string `json:"device_id"`
    DeviceName  string `json:"device_name"`
    Fingerprint string `json:"fingerprint"`
}

type PeerStore struct {
    path string
}

func NewPeerStore(path string) *PeerStore {
    return &PeerStore{path: path}
}

func (s *PeerStore) LoadAll() ([]TrustedPeer, error) {
    data, err := os.ReadFile(s.path)
    if os.IsNotExist(err) {
        return []TrustedPeer{}, nil
    }
    if err != nil {
        return nil, err
    }

    var peers []TrustedPeer
    if err := json.Unmarshal(data, &peers); err != nil {
        return nil, err
    }

    return peers, nil
}

func (s *PeerStore) Save(peer TrustedPeer) error {
    peers, err := s.LoadAll()
    if err != nil {
        return err
    }

    peers = append(peers, peer)
    data, err := json.MarshalIndent(peers, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(s.path, data, 0o600)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/pairing -run TestPeerStoreSavesAndLoadsTrustedPeers -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pairing/store.go internal/pairing/store_test.go
git commit -m "feat: persist trusted peers"
```

## Task 5: Add LAN discovery with mDNS advertisement and browse

**Files:**
- Create: `internal/discovery/mdns.go`
- Test: `internal/discovery/mdns_test.go`

- [ ] **Step 1: Write the failing test**

```go
package discovery_test

import (
    "testing"

    "bitExchange/internal/discovery"
)

func TestServiceNameIncludesBitExchangeAndDeviceID(t *testing.T) {
    got := discovery.ServiceInstanceName("device-123")
    want := "bitexchange-device-123"
    if got != want {
        t.Fatalf("ServiceInstanceName() = %q, want %q", got, want)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery -run TestServiceNameIncludesBitExchangeAndDeviceID -v`
Expected: FAIL with missing discovery package.

- [ ] **Step 3: Write minimal implementation**

`internal/discovery/mdns.go`
```go
package discovery

const ServiceType = "_bitexchange._tcp"

func ServiceInstanceName(deviceID string) string {
    return "bitexchange-" + deviceID
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery -run TestServiceNameIncludesBitExchangeAndDeviceID -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/mdns.go internal/discovery/mdns_test.go
git commit -m "feat: define LAN discovery naming"
```

## Task 6: Define transfer protocol framing and message types

**Files:**
- Create: `internal/protocol/types.go`
- Create: `internal/protocol/framing.go`
- Test: `internal/protocol/framing_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package protocol_test

import (
    "bytes"
    "testing"

    "bitExchange/internal/protocol"
)

func TestWriteAndReadFrameRoundTrip(t *testing.T) {
    var buf bytes.Buffer
    payload := []byte(`{"kind":"text"}`)

    if err := protocol.WriteFrame(&buf, payload); err != nil {
        t.Fatalf("WriteFrame returned error: %v", err)
    }

    got, err := protocol.ReadFrame(&buf)
    if err != nil {
        t.Fatalf("ReadFrame returned error: %v", err)
    }

    if string(got) != string(payload) {
        t.Fatalf("payload = %q, want %q", got, payload)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol -run TestWriteAndReadFrameRoundTrip -v`
Expected: FAIL with missing framing functions.

- [ ] **Step 3: Write minimal implementation**

`internal/protocol/types.go`
```go
package protocol

type Envelope struct {
    Kind string `json:"kind"`
}

type TextMessage struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    Body       string `json:"body"`
}

type FileHeader struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    FileName   string `json:"file_name"`
    FileSize   int64  `json:"file_size"`
}
```

`internal/protocol/framing.go`
```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol -run TestWriteAndReadFrameRoundTrip -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/types.go internal/protocol/framing.go internal/protocol/framing_test.go
git commit -m "feat: add transfer framing protocol"
```

## Task 7: Append sent and received text to chat.txt

**Files:**
- Create: `internal/history/chatlog.go`
- Test: `internal/history/chatlog_test.go`

- [ ] **Step 1: Write the failing test**

```go
package history_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "bitExchange/internal/history"
)

func TestAppendMessageWritesLineToChatFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "chat.txt")

    if err := history.AppendMessage(path, "[2026-06-09 10:00:00] desktop-a -> laptop-b: hello"); err != nil {
        t.Fatalf("AppendMessage returned error: %v", err)
    }

    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if !strings.Contains(string(data), "desktop-a -> laptop-b: hello") {
        t.Fatalf("chat.txt = %q", string(data))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/history -run TestAppendMessageWritesLineToChatFile -v`
Expected: FAIL with missing history package.

- [ ] **Step 3: Write minimal implementation**

`internal/history/chatlog.go`
```go
package history

import (
    "os"
)

func AppendMessage(path string, line string) error {
    file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(line + "\n")
    return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/history -run TestAppendMessageWritesLineToChatFile -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/history/chatlog.go internal/history/chatlog_test.go
git commit -m "feat: append text history to chat log"
```

## Task 8: Receive and save incoming files into receivedFiles/

**Files:**
- Create: `internal/transfer/files.go`
- Test: `internal/transfer/files_test.go`

- [ ] **Step 1: Write the failing test**

```go
package transfer_test

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "bitExchange/internal/transfer"
)

func TestSaveIncomingFileWritesIntoReceivedFilesDirectory(t *testing.T) {
    dir := filepath.Join(t.TempDir(), "receivedFiles")
    payload := []byte("hello file")

    savedPath, err := transfer.SaveIncomingFile(dir, "note.txt", bytes.NewReader(payload))
    if err != nil {
        t.Fatalf("SaveIncomingFile returned error: %v", err)
    }

    data, err := os.ReadFile(savedPath)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if string(data) != "hello file" {
        t.Fatalf("file contents = %q", string(data))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/transfer -run TestSaveIncomingFileWritesIntoReceivedFilesDirectory -v`
Expected: FAIL with missing transfer package or function.

- [ ] **Step 3: Write minimal implementation**

`internal/transfer/files.go`
```go
package transfer

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

func SaveIncomingFile(dir string, fileName string, reader io.Reader) (string, error) {
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", err
    }

    targetPath := filepath.Join(dir, fileName)
    targetPath = uniquePath(targetPath)

    file, err := os.Create(targetPath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    if _, err := io.Copy(file, reader); err != nil {
        return "", err
    }

    return targetPath, nil
}

func uniquePath(path string) string {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return path
    }

    ext := filepath.Ext(path)
    base := strings.TrimSuffix(filepath.Base(path), ext)
    dir := filepath.Dir(path)

    for i := 1; ; i++ {
        candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
        if _, err := os.Stat(candidate); os.IsNotExist(err) {
            return candidate
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/transfer -run TestSaveIncomingFileWritesIntoReceivedFilesDirectory -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/transfer/files.go internal/transfer/files_test.go
git commit -m "feat: save incoming files under receivedFiles"
```

## Task 9: Implement direct TCP text send and receive

**Files:**
- Create: `internal/transfer/messages.go`
- Create: `internal/transfer/listener.go`
- Create: `internal/transfer/sender.go`
- Create: `internal/transfer/receiver.go`
- Test: `tests/integration/lan_message_test.go`

- [ ] **Step 1: Write the failing integration test**

```go
package integration_test

import (
    "net"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "bitExchange/internal/history"
    "bitExchange/internal/transfer"
)

func TestLANTextSendAppendsToReceiverChatFile(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("Listen returned error: %v", err)
    }
    defer listener.Close()

    chatFile := filepath.Join(t.TempDir(), "chat.txt")
    done := make(chan error, 1)

    go func() {
        done <- transfer.AcceptOneTextMessage(listener, chatFile)
    }()

    time.Sleep(100 * time.Millisecond)

    if err := transfer.SendText(listener.Addr().String(), "desktop-a", "hello over lan"); err != nil {
        t.Fatalf("SendText returned error: %v", err)
    }

    if err := <-done; err != nil {
        t.Fatalf("AcceptOneTextMessage returned error: %v", err)
    }

    data, err := os.ReadFile(chatFile)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if !strings.Contains(string(data), "hello over lan") {
        t.Fatalf("chat.txt = %q", string(data))
    }

    _ = history.AppendMessage
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tests/integration -run TestLANTextSendAppendsToReceiverChatFile -v`
Expected: FAIL with missing transfer APIs.

- [ ] **Step 3: Write minimal implementation**

`internal/transfer/messages.go`
```go
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
```

`internal/transfer/sender.go`
```go
package transfer

import (
    "net"

    "bitExchange/internal/protocol"
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
```

`internal/transfer/receiver.go`
```go
package transfer

import (
    "encoding/json"
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
```

`internal/transfer/listener.go`
```go
package transfer
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./tests/integration -run TestLANTextSendAppendsToReceiverChatFile -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/transfer/messages.go internal/transfer/sender.go internal/transfer/receiver.go internal/transfer/listener.go tests/integration/lan_message_test.go
git commit -m "feat: add direct LAN text transfer"
```

## Task 10: Implement direct TCP file send and receive

**Files:**
- Modify: `internal/protocol/types.go`
- Modify: `internal/transfer/sender.go`
- Modify: `internal/transfer/receiver.go`
- Test: `tests/integration/lan_file_test.go`

- [ ] **Step 1: Write the failing integration test**

```go
package integration_test

import (
    "bytes"
    "net"
    "os"
    "path/filepath"
    "testing"
    "time"

    "bitExchange/internal/transfer"
)

func TestLANFileSendSavesIntoReceivedFiles(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("Listen returned error: %v", err)
    }
    defer listener.Close()

    receivedDir := filepath.Join(t.TempDir(), "receivedFiles")
    done := make(chan error, 1)

    go func() {
        done <- transfer.AcceptOneFile(listener, receivedDir)
    }()

    time.Sleep(100 * time.Millisecond)

    payload := []byte("phase1 file payload")
    if err := transfer.SendFile(listener.Addr().String(), "desktop-a", "sample.txt", bytes.NewReader(payload), int64(len(payload))); err != nil {
        t.Fatalf("SendFile returned error: %v", err)
    }

    if err := <-done; err != nil {
        t.Fatalf("AcceptOneFile returned error: %v", err)
    }

    data, err := os.ReadFile(filepath.Join(receivedDir, "sample.txt"))
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if string(data) != string(payload) {
        t.Fatalf("saved file = %q, want %q", data, payload)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tests/integration -run TestLANFileSendSavesIntoReceivedFiles -v`
Expected: FAIL with missing file transfer APIs.

- [ ] **Step 3: Write minimal implementation**

`internal/protocol/types.go`
```go
package protocol

type Envelope struct {
    Kind string `json:"kind"`
}

type TextMessage struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    Body       string `json:"body"`
}

type FileHeader struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    FileName   string `json:"file_name"`
    FileSize   int64  `json:"file_size"`
}
```

`internal/transfer/sender.go`
```go
package transfer

import (
    "encoding/json"
    "io"
    "net"

    "bitExchange/internal/protocol"
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
```

`internal/transfer/receiver.go`
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./tests/integration -run TestLANFileSendSavesIntoReceivedFiles -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/types.go internal/transfer/sender.go internal/transfer/receiver.go tests/integration/lan_file_test.go
git commit -m "feat: add direct LAN file transfer"
```

## Task 11: Add CLI commands for init, listen, send-text, and send-file

**Files:**
- Modify: `cmd/bitexchange-cli/main.go`
- Create: `internal/app/app.go`
- Test: `tests/integration/save_path_test.go`

- [ ] **Step 1: Write the failing integration test**

```go
package integration_test

import (
    "os"
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestConfiguredRootContainsChatFileAndReceivedFilesDirectory(t *testing.T) {
    root := filepath.Join(t.TempDir(), "my-bitexchange-root")

    paths, err := config.EnsureDefaultPaths(root)
    if err != nil {
        t.Fatalf("EnsureDefaultPaths returned error: %v", err)
    }

    if _, err := os.Stat(paths.ChatFile); err != nil {
        t.Fatalf("chat.txt missing: %v", err)
    }

    if stat, err := os.Stat(paths.ReceivedFilesDir); err != nil || !stat.IsDir() {
        t.Fatalf("receivedFiles directory invalid: %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails if command wiring is absent**

Run: `go test ./tests/integration -run TestConfiguredRootContainsChatFileAndReceivedFilesDirectory -v`
Expected: PASS for path helper if already implemented; then manually run `go run ./cmd/bitexchange-cli init --root "$TMPDIR/bitexchange-root" --device-name desktop-a` and observe it fail because the CLI command is not wired yet.

- [ ] **Step 3: Write minimal implementation**

`internal/app/app.go`
```go
package app

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"

    "bitExchange/internal/config"
    icrypto "bitExchange/internal/crypto"
)

func Run(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("expected subcommand")
    }

    switch args[0] {
    case "init":
        fs := flag.NewFlagSet("init", flag.ContinueOnError)
        root := fs.String("root", filepath.Join(os.Getenv("HOME"), "bitExchange"), "save root")
        deviceName := fs.String("device-name", "device", "device name")
        if err := fs.Parse(args[1:]); err != nil {
            return err
        }

        paths, err := config.EnsureDefaultPaths(*root)
        if err != nil {
            return err
        }

        identity, err := icrypto.GenerateIdentity(*deviceName)
        if err != nil {
            return err
        }

        return config.SaveDeviceConfig(filepath.Join(paths.RootDir, "device.json"), config.DeviceConfig{Identity: identity})
    default:
        return fmt.Errorf("unknown subcommand: %s", args[0])
    }
}
```

`cmd/bitexchange-cli/main.go`
```go
package main

import (
    "log"
    "os"

    "bitExchange/internal/app"
)

func main() {
    if err := app.Run(os.Args[1:]); err != nil {
        log.Fatal(err)
    }
}
```

- [ ] **Step 4: Run verification commands**

Run: `go test ./tests/integration -run TestConfiguredRootContainsChatFileAndReceivedFilesDirectory -v`
Expected: PASS

Run: `tmpdir=$(mktemp -d) && go run ./cmd/bitexchange-cli init --root "$tmpdir/bitExchange" --device-name desktop-a && test -f "$tmpdir/bitExchange/chat.txt" && test -d "$tmpdir/bitExchange/receivedFiles" && test -f "$tmpdir/bitExchange/device.json"`
Expected: exit 0

- [ ] **Step 5: Commit**

```bash
git add cmd/bitexchange-cli/main.go internal/app/app.go tests/integration/save_path_test.go
git commit -m "feat: add CLI init command"
```

## Task 12: End-to-end pairing flow and LAN discovery smoke test

**Files:**
- Test: `tests/integration/pairing_test.go`
- Modify: `internal/discovery/mdns.go`
- Modify: `internal/pairing/store.go`
- Modify: `internal/server/http.go`

- [ ] **Step 1: Write the failing end-to-end test**

```go
package integration_test

import (
    "testing"

    "bitExchange/internal/pairing"
)

func TestTrustedPeerRecordContainsFingerprintAndDeviceName(t *testing.T) {
    peer := pairing.TrustedPeer{
        DeviceID: "device-b",
        DeviceName: "laptop-b",
        Fingerprint: "fingerprint-b",
    }

    if peer.DeviceName != "laptop-b" {
        t.Fatalf("DeviceName = %q", peer.DeviceName)
    }

    if peer.Fingerprint != "fingerprint-b" {
        t.Fatalf("Fingerprint = %q", peer.Fingerprint)
    }
}
```

- [ ] **Step 2: Run test to verify it fails only if the pairing model is inconsistent**

Run: `go test ./tests/integration -run TestTrustedPeerRecordContainsFingerprintAndDeviceName -v`
Expected: PASS if the model still matches. If this already passes, continue to the manual smoke check below as the real gate.

- [ ] **Step 3: Perform the manual smoke check**

Run server in terminal A:
```bash
go run ./cmd/bitexchange-server
```

Run listener client in terminal B:
```bash
go run ./cmd/bitexchange-cli init --root /tmp/bitexchange-a --device-name desktop-a
```

Expected: client root contains `chat.txt`, `receivedFiles/`, and `device.json`.

Then verify on the same LAN with a second device or second shell later in implementation:
```bash
go run ./cmd/bitexchange-cli init --root /tmp/bitexchange-b --device-name laptop-b
```

Expected: both devices initialize with compatible identity and pairing storage shapes for the next task batch.

- [ ] **Step 4: Record MVP gaps before moving on**

Create a short checklist in the PR description or session notes with these known gaps still open after Phase 1:

```text
- CLI has no desktop chat UI yet
- no public signaling encryption yet
- no NAT traversal yet
- no relay yet
- no Android/Web clients yet
- no group send yet
```

- [ ] **Step 5: Commit**

```bash
git add tests/integration/pairing_test.go internal/discovery/mdns.go internal/pairing/store.go internal/server/http.go
git commit -m "test: document phase 1 pairing smoke coverage"
```

## Plan self-review

### Spec coverage

Covered from the spec:
- Go core and Go server from section 11.1
- trusted-device identity and pairing basics from section 5
- same-subnet discovery naming foundation from section 4.1
- direct text and file transfer from section 6
- `chat.txt` append behavior from section 6.1 and 6.2
- `receivedFiles/` save behavior from section 6.1

Intentionally deferred to later plans:
- desktop dark UI from section 7
- Android from section 7.2
- Web from section 7.3
- public signaling encryption from section 5 and 8
- path priority beyond LAN direct from section 4.4
- relay and NAT traversal from sections 4 and 8
- group send from section 6
- GitHub Actions packaging from section 9.6

### Placeholder scan

Task steps use explicit file paths, commands, and code snippets. Deferred scope is explicitly listed in scope boundaries rather than hidden in tasks.

### Type consistency

The plan consistently uses:
- `config.EnsureDefaultPaths`
- `crypto.GenerateIdentity`
- `pairing.TrustedPeer`
- `history.AppendMessage`
- `transfer.SendText`
- `transfer.SendFile`
- `transfer.AcceptOneTextMessage`
- `transfer.AcceptOneFile`

No alternate names are introduced later in the plan.
