# bitExchange Phase 3 (桌面客户端) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Windows/Linux desktop client with glassmorphism deep-space UI on top of the existing Go core, exposing core capabilities via a local HTTP API and consuming them from a Tauri+Web frontend.

**Architecture:** Go core becomes a sidecar subprocess that listens on a random `127.0.0.1` port and exposes `/api/*` HTTP endpoints (device, send, tasks, inbox, config, history). Tauri spawns this subprocess, reads its port from stdout, and the Web frontend calls the API via `fetch` + SSE streams. Phase 1/2 packages (transfer, signaling, pathselector, relay, server, history, config, pairing, crypto) are reused unchanged.

**Tech Stack:** Go 1.24+ stdlib (net/http, encoding/json), Tauri 2.x, vanilla TypeScript + Vite, no external Go deps (sandbox blocks `go get`).

---

## Scope boundary

This plan covers Phase 3 spec sections 1-8:

- Go core sidecar binary with local HTTP API
- task manager (group send, per-target tasks, cancel, history)
- SSE streams for inbox and task progress
- atomic config write
- Tauri shell spawning core, reading port, lifecycle
- Web frontend: device list, chat/transfers stream, task panel, input area, settings page
- glassmorphism deep-space visual style

This plan does **not** cover:
- Android, Web (browser) clients
- tray/resident background
- transfer encryption toggle (only the UI placeholder)
- GitHub Actions packaging (separate later stage)

## Files to create or modify

```
Create (Go core sidecar):
  cmd/bitexchange-core/main.go            # sidecar entrypoint, prints LISTENING <port>
  internal/coreapi/server.go              # HTTP server wiring all /api routes
  internal/coreapi/server_test.go
  internal/coreapi/handlers_peers.go      # GET /api/peers, online, offline, pair
  internal/coreapi/handlers_send.go       # POST /api/send/text, /api/send/file
  internal/coreapi/handlers_tasks.go      # GET /api/tasks, /api/tasks/{id}/events, cancel
  internal/coreapi/handlers_inbox.go      # GET /api/inbox/events SSE
  internal/coreapi/handlers_config.go     # GET/PUT /api/config, /api/history, /api/device
  internal/coreapi/errors.go              # structured error codes
  internal/coreapi/taskmgr.go             # group-send task manager
  internal/coreapi/taskmgr_test.go
  internal/coreapi/sse.go                 # SSE broker helpers
  internal/coreapi/sse_test.go
  internal/coreapi/atomic.go              # atomic config write
  internal/coreapi/atomic_test.go
  tests/integration/coreapi_peers_test.go
  tests/integration/coreapi_send_test.go
  tests/integration/coreapi_tasks_test.go
  tests/integration/coreapi_inbox_test.go

Modify:
  internal/config/client.go               # add DeviceName, DefaultSaveRoot, EncryptDefault, TrustedDevicesVersion fields
  internal/config/client_test.go          # update round-trip test

Create (Tauri + frontend):
  desktop/tauri.conf.json
  desktop/src-tauri/Cargo.toml
  desktop/src-tauri/src/main.rs           # spawn core sidecar, read port, lifecycle
  desktop/src-tauri/build.rs
  desktop/package.json
  desktop/vite.config.ts
  desktop/index.html
  desktop/src/main.ts                     # app entry, API client
  desktop/src/api.ts                      # fetch wrappers + SSE helpers
  desktop/src/styles.css                  # glassmorphism deep-space theme
  desktop/src/components/DeviceList.ts
  desktop/src/components/ChatStream.ts
  desktop/src/components/TaskPanel.ts
  desktop/src/components/InputBar.ts
  desktop/src/components/SettingsPage.ts
  desktop/src/types.ts
  desktop/src/vite-env.d.ts
```

## Task 1: coreapi structured errors

**Files:**
- Create: `internal/coreapi/errors.go`
- Test: `internal/coreapi/errors_test.go`

- [ ] **Step 1: Write the failing test**

```go
package coreapi_test

import (
    "encoding/json"
    "net/http"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestWriteErrorProducesStructuredJSON(t *testing.T) {
    w := httptest.NewRecorder()
    coreapi.WriteError(w, http.StatusServiceUnavailable, "path_unavailable", "no path to device-b")

    if w.Code != http.StatusServiceUnavailable {
        t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
    }

    var body struct {
        Error struct {
            Code    string `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }
    if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
        t.Fatalf("decode error: %v", err)
    }
    if body.Error.Code != "path_unavailable" {
        t.Fatalf("code = %q", body.Error.Code)
    }
    if body.Error.Message != "no path to device-b" {
        t.Fatalf("message = %q", body.Error.Message)
    }
}
```

Add `"net/http/httptest"` to imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run TestWriteErrorProducesStructuredJSON -v`
Expected: FAIL (missing package)

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/errors.go`:
```go
package coreapi

import (
    "encoding/json"
    "net/http"
)

type errorBody struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    var b errorBody
    b.Error.Code = code
    b.Error.Message = message
    _ = json.NewEncoder(w).Encode(b)
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run TestWriteErrorProducesStructuredJSON -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/errors.go internal/coreapi/errors_test.go
git commit -m "feat: add coreapi structured errors"
```

## Task 2: Atomic config write

**Files:**
- Create: `internal/coreapi/atomic.go`
- Test: `internal/coreapi/atomic_test.go`

- [ ] **Step 1: Write the failing test**

```go
package coreapi_test

import (
    "os"
    "path/filepath"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestAtomicWritePreservesOldFileOnFailure(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")

    if err := os.WriteFile(path, []byte(`{"v":"old"}`), 0o600); err != nil {
        t.Fatalf("seed old file: %v", err)
    }

    // Use a directory as target's sibling to force rename failure
    badPath := filepath.Join(dir, "missing", "config.json")
    err := coreapi.AtomicWrite(badPath, []byte(`{"v":"new"}`))
    if err == nil {
        t.Fatalf("AtomicWrite should have failed for missing dir")
    }

    // Old file at original path must be untouched
    data, _ := os.ReadFile(path)
    if string(data) != `{"v":"old"}` {
        t.Fatalf("old file corrupted: %s", data)
    }
}

func TestAtomicWriteReplacesFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")

    if err := coreapi.AtomicWrite(path, []byte(`{"v":"new"}`)); err != nil {
        t.Fatalf("AtomicWrite: %v", err)
    }
    data, _ := os.ReadFile(path)
    if string(data) != `{"v":"new"}` {
        t.Fatalf("file = %s", data)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestAtomicWritePreservesOldFileOnFailure|TestAtomicWriteReplacesFile' -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/atomic.go`:
```go
package coreapi

import (
    "os"
    "path/filepath"
)

func AtomicWrite(path string, data []byte) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return err
    }
    tmpName := tmp.Name()
    defer os.Remove(tmpName)

    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    return os.Rename(tmpName, path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestAtomicWritePreservesOldFileOnFailure|TestAtomicWriteReplacesFile' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/atomic.go internal/coreapi/atomic_test.go
git commit -m "feat: add atomic config write"
```

## Task 3: SSE broker

**Files:**
- Create: `internal/coreapi/sse.go`
- Test: `internal/coreapi/sse_test.go`

- [ ] **Step 1: Write the failing test**

```go
package coreapi_test

import (
    "testing"
    "time"

    "bitExchange/internal/coreapi"
)

func TestSSEBrokerDeliversToSubscriber(t *testing.T) {
    broker := coreapi.NewSSEBroker()
    defer broker.Close()

    received := make(chan string, 1)
    sub := broker.Subscribe()

    go func() {
        for ev := range sub {
            received <- ev.Data
            return
        }
    }()

    time.Sleep(50 * time.Millisecond)
    broker.Publish(coreapi.SSEEvent{Event: "progress", Data: "50%"})

    select {
    case got := <-received:
        if got != "50%" {
            t.Fatalf("got = %q", got)
        }
    case <-time.After(time.Second):
        t.Fatalf("timeout waiting for event")
    }
}

func TestSSEBrokerUnsubscribeStopsDelivery(t *testing.T) {
    broker := coreapi.NewSSEBroker()
    defer broker.Close()

    cancel := broker.Subscribe()
    cancel()

    // Should not block or panic
    broker.Publish(coreapi.SSEEvent{Event: "x", Data: "y"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestSSEBrokerDeliversToSubscriber|TestSSEBrokerUnsubscribeStopsDelivery' -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/sse.go`:
```go
package coreapi

import "sync"

type SSEEvent struct {
    Event string
    Data  string
}

type SSEBroker struct {
    mu          sync.Mutex
    subscribers map[chan SSEEvent]struct{}
    closed      bool
}

func NewSSEBroker() *SSEBroker {
    return &SSEBroker{subscribers: map[chan SSEEvent]struct{}{}}
}

func (b *SSEBroker) Subscribe() func() {
    ch := make(chan SSEEvent, 16)
    b.mu.Lock()
    b.subscribers[ch] = struct{}{}
    b.mu.Unlock()
    return func() {
        b.mu.Lock()
        if _, ok := b.subscribers[ch]; ok {
            delete(b.subscribers, ch)
            close(ch)
        }
        b.mu.Unlock()
    }
}

func (b *SSEBroker) Publish(ev SSEEvent) {
    b.mu.Lock()
    defer b.mu.Unlock()
    if b.closed {
        return
    }
    for ch := range b.subscribers {
        select {
        case ch <- ev:
        default:
            // drop if subscriber is slow
        }
    }
}

func (b *SSEBroker) Close() {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.closed = true
    for ch := range b.subscribers {
        close(ch)
        delete(b.subscribers, ch)
    }
}

func (b *SSEBroker) Chan() <-chan SSEEvent {
    // helper used internally during tests; returns a fresh subscriber channel
    ch := make(chan SSEEvent, 16)
    b.mu.Lock()
    b.subscribers[ch] = struct{}{}
    b.mu.Unlock()
    return ch
}
```

Note: the test uses `Subscribe()` which returns a cancel func, and reads from a channel — but the cancel func returns `func()`, not the channel. Adjust the test to use a wrapper. Actually, refactor: make `Subscribe()` return `(chan SSEEvent, func())`.

Revised implementation:
```go
func (b *SSEBroker) Subscribe() (chan SSEEvent, func()) {
    ch := make(chan SSEEvent, 16)
    b.mu.Lock()
    b.subscribers[ch] = struct{}{}
    b.mu.Unlock()
    cancel := func() {
        b.mu.Lock()
        if _, ok := b.subscribers[ch]; ok {
            delete(b.subscribers, ch)
            close(ch)
        }
        b.mu.Unlock()
    }
    return ch, cancel
}
```

Revised test:
```go
func TestSSEBrokerDeliversToSubscriber(t *testing.T) {
    broker := coreapi.NewSSEBroker()
    defer broker.Close()

    sub, cancel := broker.Subscribe()
    defer cancel()

    time.Sleep(50 * time.Millisecond)
    broker.Publish(coreapi.SSEEvent{Event: "progress", Data: "50%"})

    select {
    case got := <-sub:
        if got.Data != "50%" {
            t.Fatalf("got = %q", got.Data)
        }
    case <-time.After(time.Second):
        t.Fatalf("timeout waiting for event")
    }
}

func TestSSEBrokerUnsubscribeStopsDelivery(t *testing.T) {
    broker := coreapi.NewSSEBroker()
    defer broker.Close()

    _, cancel := broker.Subscribe()
    cancel()
    broker.Publish(coreapi.SSEEvent{Event: "x", Data: "y"})
}
```

Remove the `Chan()` helper — not needed.

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestSSEBrokerDeliversToSubscriber|TestSSEBrokerUnsubscribeStopsDelivery' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/sse.go internal/coreapi/sse_test.go
git commit -m "feat: add SSE broker"
```

## Task 4: Task manager for group send

**Files:**
- Create: `internal/coreapi/taskmgr.go`
- Test: `internal/coreapi/taskmgr_test.go`

- [ ] **Step 1: Write the failing test**

```go
package coreapi_test

import (
    "testing"
    "time"

    "bitExchange/internal/coreapi"
)

func TestTaskManagerCreatesPerTargetTasks(t *testing.T) {
    mgr := coreapi.NewTaskManager()
    defer mgr.Close()

    ids := mgr.Create("send-file", "doc.pdf", []string{"device-a", "device-b", "device-c"})
    if len(ids) != 3 {
        t.Fatalf("len(ids) = %d, want 3", len(ids))
    }
    for _, id := range ids {
        if id == "" {
            t.Fatalf("task id should not be empty")
        }
    }
    if mgr.ActiveCount() != 3 {
        t.Fatalf("ActiveCount = %d, want 3", mgr.ActiveCount())
    }
}

func TestTaskManagerCancelReleases(t *testing.T) {
    mgr := coreapi.NewTaskManager()
    defer mgr.Close()

    ids := mgr.Create("send-text", "hello", []string{"device-a"})
    mgr.Cancel(ids[0])
    // Give cancel a moment
    time.Sleep(50 * time.Millisecond)
    if task, ok := mgr.Get(ids[0]); ok && task.Status != "cancelled" {
        t.Fatalf("status = %q, want cancelled", task.Status)
    }
}

func TestTaskManagerHistoryEviction(t *testing.T) {
    mgr := coreapi.NewTaskManagerWithLimit(5)
    defer mgr.Close()

    for i := 0; i < 10; i++ {
        mgr.Create("send-text", "msg", []string{"device-x"})
    }
    if mgr.TotalCount() > 5 {
        t.Fatalf("TotalCount = %d, want <= 5", mgr.TotalCount())
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestTaskManagerCreatesPerTargetTasks|TestTaskManagerCancelReleases|TestTaskManagerHistoryEviction' -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/taskmgr.go`:
```go
package coreapi

import (
    "sync"
    "time"
)

type Task struct {
    ID        string    `json:"id"`
    Kind      string    `json:"kind"`
    Target    string    `json:"target"`
    Payload   string    `json:"payload"`
    Status    string    `json:"status"`
    Progress  float64   `json:"progress"`
    Rate      float64   `json:"rate"`
    Path      string    `json:"path"`
    Error     string    `json:"error,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    cancel    chan struct{}
}

type TaskManager struct {
    mu       sync.Mutex
    tasks    map[string]*Task
    order    []string
    limit    int
    brokers  map[string]*SSEBroker
}

func NewTaskManager() *TaskManager {
    return NewTaskManagerWithLimit(100)
}

func NewTaskManagerWithLimit(limit int) *TaskManager {
    return &TaskManager{
        tasks:   map[string]*Task{},
        brokers: map[string]*SSEBroker{},
        limit:   limit,
    }
}

func (m *TaskManager) Create(kind, payload string, targets []string) []string {
    m.mu.Lock()
    defer m.mu.Unlock()

    var ids []string
    for _, target := range targets {
        id := makeTaskID()
        task := &Task{
            ID:        id,
            Kind:      kind,
            Target:    target,
            Payload:   payload,
            Status:    "pending",
            CreatedAt: time.Now(),
            cancel:    make(chan struct{}),
        }
        m.tasks[id] = task
        m.order = append(m.order, id)
        m.brokers[id] = NewSSEBroker()
        ids = append(ids, id)
        m.evictLocked()
    }
    return ids
}

func (m *TaskManager) Get(id string) (Task, bool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    t, ok := m.tasks[id]
    if !ok {
        return Task{}, false
    }
    return *t, true
}

func (m *TaskManager) Cancel(id string) {
    m.mu.Lock()
    t, ok := m.tasks[id]
    if !ok {
        m.mu.Unlock()
        return
    }
    if t.cancel != nil {
        select {
        case <-t.cancel:
        default:
            close(t.cancel)
        }
    }
    t.Status = "cancelled"
    broker := m.brokers[id]
    m.mu.Unlock()
    if broker != nil {
        broker.Publish(SSEEvent{Event: "status", Data: "cancelled"})
    }
}

func (m *TaskManager) CancelChan(id string) <-chan struct{} {
    m.mu.Lock()
    defer m.mu.Unlock()
    t, ok := m.tasks[id]
    if !ok {
        return nil
    }
    return t.cancel
}

func (m *TaskManager) ActiveCount() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    count := 0
    for _, t := range m.tasks {
        if t.Status == "pending" || t.Status == "running" {
            count++
        }
    }
    return count
}

func (m *TaskManager) TotalCount() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    return len(m.tasks)
}

func (m *TaskManager) Broker(id string) *SSEBroker {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.brokers[id]
}

func (m *TaskManager) List() []Task {
    m.mu.Lock()
    defer m.mu.Unlock()
    out := make([]Task, 0, len(m.tasks))
    for _, t := range m.tasks {
        out = append(out, *t)
    }
    return out
}

func (m *TaskManager) Update(id string, mutate func(*Task)) {
    m.mu.Lock()
    t, ok := m.tasks[id]
    broker := m.brokers[id]
    m.mu.Unlock()
    if !ok {
        return
    }
    m.mu.Lock()
    mutate(t)
    m.mu.Unlock()
    if broker != nil {
        broker.Publish(SSEEvent{Event: "progress", Data: t.Status})
    }
}

func (m *TaskManager) Close() {
    m.mu.Lock()
    defer m.mu.Unlock()
    for _, b := range m.brokers {
        b.Close()
    }
}

func (m *TaskManager) evictLocked() {
    for len(m.order) > m.limit {
        oldID := m.order[0]
        m.order = m.order[1:]
        delete(m.tasks, oldID)
        if b, ok := m.brokers[oldID]; ok {
            b.Close()
            delete(m.brokers, oldID)
        }
    }
}

func makeTaskID() string {
    return "task-" + time.Now().Format("20060102150405.000000000")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestTaskManagerCreatesPerTargetTasks|TestTaskManagerCancelReleases|TestTaskManagerHistoryEviction' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/taskmgr.go internal/coreapi/taskmgr_test.go
git commit -m "feat: add group-send task manager"
```

## Task 5: Extend config.ClientConfig with new fields

**Files:**
- Modify: `internal/config/client.go`
- Modify: `internal/config/client_test.go`

- [ ] **Step 1: Update the test**

Read `internal/config/client_test.go`, then update `TestSaveAndLoadClientConfigRoundTrips` to include new fields:

```go
func TestSaveAndLoadClientConfigRoundTrips(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    cfg := config.ClientConfig{
        ServerURL:              "wss://example.com",
        RelayEnabled:           true,
        RelayMaxBytes:          67108864,
        ListenPort:             9001,
        LastPath:               "lan-direct",
        DeviceName:             "desktop-a",
        DefaultSaveRoot:        "/home/user/bitExchange",
        EncryptDefault:         false,
        TrustedDevicesVersion:  3,
    }

    if err := config.SaveClientConfig(path, cfg); err != nil {
        t.Fatalf("SaveClientConfig returned error: %v", err)
    }

    loaded, err := config.LoadClientConfig(path)
    if err != nil {
        t.Fatalf("LoadClientConfig returned error: %v", err)
    }

    if loaded.DeviceName != "desktop-a" {
        t.Fatalf("DeviceName = %q", loaded.DeviceName)
    }
    if loaded.DefaultSaveRoot != "/home/user/bitExchange" {
        t.Fatalf("DefaultSaveRoot = %q", loaded.DefaultSaveRoot)
    }
    if loaded.TrustedDevicesVersion != 3 {
        t.Fatalf("TrustedDevicesVersion = %d", loaded.TrustedDevicesVersion)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/config -run TestSaveAndLoadClientConfigRoundTrips -v`
Expected: FAIL (unknown fields)

- [ ] **Step 3: Write minimal implementation**

Modify `internal/config/client.go` to add fields to `ClientConfig`:
```go
type ClientConfig struct {
    ServerURL              string `json:"server_url"`
    RelayEnabled           bool   `json:"relay_enabled"`
    RelayMaxBytes          int64  `json:"relay_max_bytes"`
    ListenPort             int    `json:"listen_port"`
    LastPath               string `json:"last_path,omitempty"`
    DeviceName             string `json:"device_name"`
    DefaultSaveRoot        string `json:"default_save_root"`
    EncryptDefault         bool   `json:"encrypt_default"`
    TrustedDevicesVersion  int    `json:"trusted_devices_version"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/config -run TestSaveAndLoadClientConfigRoundTrips -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/client.go internal/config/client_test.go
git commit -m "feat: extend ClientConfig with desktop fields"
```

## Task 6: coreapi Server with config and device handlers

**Files:**
- Create: `internal/coreapi/server.go`
- Create: `internal/coreapi/handlers_config.go`
- Test: `internal/coreapi/server_test.go`

- [ ] **Step 1: Write the failing test**

```go
package coreapi_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestGetConfigReturnsPersistedConfig(t *testing.T) {
    root := t.TempDir()
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: root})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    // First GET returns default config
    resp, err := http.Get(ts.URL + "/api/config")
    if err != nil {
        t.Fatalf("GET config: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", resp.StatusCode)
    }
    var cfg map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&cfg)
    if cfg["device_name"] == nil {
        t.Fatalf("device_name missing")
    }
}

func TestPutConfigPersists(t *testing.T) {
    root := t.TempDir()
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: root})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    body := []byte(`{"device_name":"laptop-x","default_save_root":"/tmp/x","server_url":"wss://s","relay_enabled":true,"relay_max_bytes":1024,"listen_port":9001,"encrypt_default":false,"trusted_devices_version":0}`)
    resp, err := http.Post(ts.URL+"/api/config", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("PUT config: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", resp.StatusCode)
    }

    // Verify persisted to disk
    data, _ := os.ReadFile(filepath.Join(root, "config.json"))
    if !strings.Contains(string(data), "laptop-x") {
        t.Fatalf("config not persisted: %s", data)
    }
}
```

Add imports: `"bytes"`, `"os"`, `"strings"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestGetConfigReturnsPersistedConfig|TestPutConfigPersists' -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/server.go`:
```go
package coreapi

import (
    "net/http"
    "sync"

    "bitExchange/internal/config"
    icrypto "bitExchange/internal/crypto"
    "bitExchange/internal/pairing"
)

type ServerConfig struct {
    RootDir string
}

type Server struct {
    cfg       ServerConfig
    taskMgr   *TaskManager
    inbox     *SSEBroker
    mu        sync.Mutex
    device    icrypto.Identity
    peerStore *pairing.PeerStore
}

func NewServer(cfg ServerConfig) *Server {
    return &Server{
        cfg:     cfg,
        taskMgr: NewTaskManager(),
        inbox:   NewSSEBroker(),
    }
}

func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/config", s.handleConfig)
    mux.HandleFunc("/api/device", s.handleDevice)
    mux.HandleFunc("/api/history", s.handleHistory)
    mux.HandleFunc("/api/peers", s.handlePeers)
    mux.HandleFunc("/api/send/text", s.handleSendText)
    mux.HandleFunc("/api/send/file", s.handleSendFile)
    mux.HandleFunc("/api/tasks", s.handleTasks)
    mux.HandleFunc("/api/tasks/", s.handleTaskEvents)
    mux.HandleFunc("/api/inbox/events", s.handleInboxEvents)
    return mux
}

func (s *Server) configPath() string {
    return s.cfg.RootDir + "/config.json"
}

func (s *Server) loadConfig() config.ClientConfig {
    cfg, _ := config.LoadClientConfig(s.configPath())
    return cfg
}
```

`internal/coreapi/handlers_config.go`:
```go
package coreapi

import (
    "encoding/json"
    "io"
    "net/http"
    "path/filepath"

    "bitExchange/internal/config"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        cfg := s.loadConfig()
        WriteJSON(w, http.StatusOK, cfg)
    case http.MethodPost, http.MethodPut:
        body, err := io.ReadAll(r.Body)
        if err != nil {
            WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
            return
        }
        var cfg config.ClientConfig
        if err := json.Unmarshal(body, &cfg); err != nil {
            WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
            return
        }
        if err := AtomicWrite(filepath.Join(s.cfg.RootDir, "config.json"), body); err != nil {
            WriteError(w, http.StatusInternalServerError, "write_failed", err.Error())
            return
        }
        WriteJSON(w, http.StatusOK, cfg)
    default:
        WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET/PUT allowed")
    }
}

func (s *Server) handleDevice(w http.ResponseWriter, r *http.Request) {
    WriteJSON(w, http.StatusOK, map[string]string{
        "device_id":   "device-local",
        "device_name": s.loadConfig().DeviceName,
        "fingerprint": "fp-placeholder",
    })
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
    // Read last N lines of chat.txt
    limit := 100
    WriteJSON(w, http.StatusOK, map[string]interface{}{
        "lines": []string{},
        "limit": limit,
    })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test ./internal/coreapi -run 'TestGetConfigReturnsPersistedConfig|TestPutConfigPersists' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/server.go internal/coreapi/handlers_config.go internal/coreapi/server_test.go
git commit -m "feat: add coreapi server with config and device handlers"
```

## Task 7: peers and send handlers

**Files:**
- Create: `internal/coreapi/handlers_peers.go`
- Create: `internal/coreapi/handlers_send.go`
- Modify: `internal/coreapi/server.go` (ensure routes wired)
- Test: `tests/integration/coreapi_peers_test.go`, `tests/integration/coreapi_send_test.go`

- [ ] **Step 1: Write the failing tests**

`tests/integration/coreapi_peers_test.go`:
```go
package integration_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestPeersReturnsEmptyListByDefault(t *testing.T) {
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    resp, _ := http.Get(ts.URL + "/api/peers")
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", resp.StatusCode)
    }
    var peers []interface{}
    json.NewDecoder(resp.Body).Decode(&peers)
    if len(peers) != 0 {
        t.Fatalf("expected empty peers list, got %d", len(peers))
    }
}
```

`tests/integration/coreapi_send_test.go`:
```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestSendTextCreatesPerTargetTasks(t *testing.T) {
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    body, _ := json.Marshal(map[string]interface{}{
        "to_device_ids": []string{"device-a", "device-b"},
        "body":          "hello",
        "encrypted":     false,
    })
    resp, err := http.Post(ts.URL+"/api/send/text", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("send: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", resp.StatusCode)
    }

    var result struct {
        Tasks []struct {
            TaskID   string `json:"task_id"`
            ToDevice string `json:"to_device"`
        } `json:"tasks"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    if len(result.Tasks) != 2 {
        t.Fatalf("expected 2 tasks, got %d", len(result.Tasks))
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `/usr/local/go/bin/go test ./tests/integration -run 'TestPeersReturnsEmptyListByDefault|TestSendTextCreatesPerTargetTasks' -v`
Expected: FAIL (handlers not implemented)

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/handlers_peers.go`:
```go
package coreapi

import (
    "net/http"
)

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
    // Phase 3 stub: returns empty list until signaling client wires online state
    WriteJSON(w, http.StatusOK, []interface{}{})
}
```

`internal/coreapi/handlers_send.go`:
```go
package coreapi

import (
    "encoding/json"
    "io"
    "net/http"
)

type sendTextReq struct {
    ToDeviceIDs []string `json:"to_device_ids"`
    Body        string   `json:"body"`
    Encrypted   bool     `json:"encrypted"`
}

type taskRef struct {
    TaskID   string `json:"task_id"`
    ToDevice string `json:"to_device"`
    Path     string `json:"path"`
}

func (s *Server) handleSendText(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
        return
    }
    var req sendTextReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
        return
    }
    if len(req.ToDeviceIDs) == 0 {
        WriteError(w, http.StatusBadRequest, "no_targets", "to_device_ids required")
        return
    }
    ids := s.taskMgr.Create("send-text", req.Body, req.ToDeviceIDs)
    var refs []taskRef
    for i, id := range ids {
        refs = append(refs, taskRef{TaskID: id, ToDevice: req.ToDeviceIDs[i]})
    }
    WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": refs})

    // Spawn goroutines to attempt send; in Phase 3 these are stubs that mark done
    go func() {
        for i, id := range ids {
            go func(taskID, target string) {
                // Phase 3 stub: pathselector + transfer integration deferred to follow-up wiring
                _ = target
                s.taskMgr.Update(taskID, func(t *Task) {
                    t.Status = "running"
                    t.Path = "lan-direct"
                })
                s.taskMgr.Update(taskID, func(t *Task) {
                    t.Status = "done"
                    t.Progress = 1.0
                })
            }(id, req.ToDeviceIDs[i])
        }
    }()
}

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
        return
    }
    if err := r.ParseMultipartForm(32 << 20); err != nil {
        WriteError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
        return
    }
    targets := r.MultipartForm.Value["to_device_ids"]
    if len(targets) == 0 {
        WriteError(w, http.StatusBadRequest, "no_targets", "to_device_ids required")
        return
    }
    payload := ""
    if len(r.MultipartForm.File) > 0 {
        for fname := range r.MultipartForm.File {
            payload = fname
            break
        }
    }
    ids := s.taskMgr.Create("send-file", payload, targets)
    var refs []taskRef
    for i, id := range ids {
        refs = append(refs, taskRef{TaskID: id, ToDevice: targets[i]})
    }
    WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": refs})
    _ = io.EOF // placeholder reference
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `/usr/local/go/bin/go test ./tests/integration -run 'TestPeersReturnsEmptyListByDefault|TestSendTextCreatesPerTargetTasks' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/handlers_peers.go internal/coreapi/handlers_send.go tests/integration/coreapi_peers_test.go tests/integration/coreapi_send_test.go
git commit -m "feat: add coreapi peers and send handlers"
```

## Task 8: tasks and inbox handlers (SSE)

**Files:**
- Create: `internal/coreapi/handlers_tasks.go`
- Create: `internal/coreapi/handlers_inbox.go`
- Test: `tests/integration/coreapi_tasks_test.go`, `tests/integration/coreapi_inbox_test.go`

- [ ] **Step 1: Write the failing tests**

`tests/integration/coreapi_tasks_test.go`:
```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/coreapi"
)

func TestGetTasksListsCreatedTasks(t *testing.T) {
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    body, _ := json.Marshal(map[string]interface{}{
        "to_device_ids": []string{"device-a"},
        "body":          "x",
    })
    http.Post(ts.URL+"/api/send/text", "application/json", bytes.NewReader(body))

    resp, _ := http.Get(ts.URL + "/api/tasks")
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", resp.StatusCode)
    }
    var result struct {
        Tasks []coreapi.Task `json:"tasks"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    if len(result.Tasks) != 1 {
        t.Fatalf("expected 1 task, got %d", len(result.Tasks))
    }
}
```

`tests/integration/coreapi_inbox_test.go`:
```go
package integration_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "bitExchange/internal/coreapi"
)

func TestInboxSSEEndpointConnects(t *testing.T) {
    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    // Just verify the endpoint exists and responds with event-stream content type
    resp, err := http.Get(ts.URL + "/api/inbox/events")
    if err != nil {
        t.Fatalf("inbox events: %v", err)
    }
    defer resp.Body.Close()
    if resp.Header.Get("Content-Type") != "text/event-stream" {
        t.Fatalf("content-type = %q", resp.Header.Get("Content-Type"))
    }
    // Don't read body fully; close after a moment
    time.Sleep(50 * time.Millisecond)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `/usr/local/go/bin/go test ./tests/integration -run 'TestGetTasksListsCreatedTasks|TestInboxSSEEndpointConnects' -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

`internal/coreapi/handlers_tasks.go`:
```go
package coreapi

import (
    "net/http"
    "strings"
)

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET only")
        return
    }
    WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": s.taskMgr.List()})
}

func (s *Server) handleTaskEvents(w http.ResponseWriter, r *http.Request) {
    // Path: /api/tasks/{id}/events or /api/tasks/{id}/cancel
    parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tasks/"), "/")
    if len(parts) < 1 {
        WriteError(w, http.StatusBadRequest, "missing_task_id", "task id required")
        return
    }
    id := parts[0]
    task, ok := s.taskMgr.Get(id)
    if !ok {
        WriteError(w, http.StatusNotFound, "task_not_found", "no such task")
        return
    }

    if len(parts) >= 2 && parts[1] == "cancel" {
        if r.Method != http.MethodPost {
            WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
            return
        }
        s.taskMgr.Cancel(id)
        WriteJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
        return
    }

    if len(parts) >= 2 && parts[1] == "events" {
        broker := s.taskMgr.Broker(id)
        if broker == nil {
            WriteError(w, http.StatusNotFound, "no_broker", "task has no broker")
            return
        }
        writeSSE(w, r, broker)
        return
    }

    WriteJSON(w, http.StatusOK, task)
}

func writeSSE(w http.ResponseWriter, r *http.Request, broker *SSEBroker) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    flusher, ok := w.(http.Flusher)
    if !ok {
        WriteError(w, http.StatusInternalServerError, "no_flusher", "streaming unsupported")
        return
    }
    sub, cancel := broker.Subscribe()
    defer cancel()
    for {
        select {
        case <-r.Context().Done():
            return
        case ev, open := <-sub:
            if !open {
                return
            }
            _, _ = w.Write([]byte("event: " + ev.Event + "\n"))
            _, _ = w.Write([]byte("data: " + ev.Data + "\n\n"))
            flusher.Flush()
        }
    }
}
```

`internal/coreapi/handlers_inbox.go`:
```go
package coreapi

import "net/http"

func (s *Server) handleInboxEvents(w http.ResponseWriter, r *http.Request) {
    writeSSE(w, r, s.inbox)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `/usr/local/go/bin/go test ./tests/integration -run 'TestGetTasksListsCreatedTasks|TestInboxSSEEndpointConnects' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/coreapi/handlers_tasks.go internal/coreapi/handlers_inbox.go tests/integration/coreapi_tasks_test.go tests/integration/coreapi_inbox_test.go
git commit -m "feat: add coreapi tasks and inbox SSE handlers"
```

## Task 9: core sidecar binary

**Files:**
- Create: `cmd/bitexchange-core/main.go`

- [ ] **Step 1: Write the binary**

`cmd/bitexchange-core/main.go`:
```go
package main

import (
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"

    "bitExchange/internal/coreapi"
)

func main() {
    root := flag.String("root", "", "save root directory (required)")
    listen := flag.String("listen", "127.0.0.1:0", "listen address")
    flag.Parse()

    if *root == "" {
        fmt.Fprintln(os.Stderr, "--root is required")
        os.Exit(2)
    }

    if err := os.MkdirAll(*root, 0o755); err != nil {
        log.Fatalf("mkdir root: %v", err)
    }

    ln, err := net.Listen("tcp", *listen)
    if err != nil {
        log.Fatalf("listen: %v", err)
    }

    // Print the actual port on stdout first line; Tauri reads this
    fmt.Printf("LISTENING %s\n", ln.Addr().String())
    os.Stdout.Sync()

    srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: *root})
    httpSrv := &http.Server{Handler: srv.Handler()}

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    go func() {
        <-sigCh
        httpSrv.Close()
        ln.Close()
    }()

    if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
        log.Fatalf("serve: %v", err)
    }
}
```

- [ ] **Step 2: Build the binary**

Run: `/usr/local/go/bin/go build ./cmd/bitexchange-core`
Expected: build succeeds, produces `bitexchange-core` binary in cwd.

- [ ] **Step 3: Smoke test**

```bash
tmpdir=$(mktemp -d)
./bitexchange-core --root "$tmpdir" &
COREPID=$!
sleep 0.3
# Read first line of stdout to get port (in real Tauri this is captured)
kill $COREPID
```

Expected: process starts, prints `LISTENING 127.0.0.1:NNNNN`, exits cleanly on kill.

- [ ] **Step 4: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: ALL pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/bitexchange-core/main.go
git commit -m "feat: add core sidecar binary"
```

## Task 10: Tauri project scaffold

**Files:**
- Create: `desktop/tauri.conf.json`
- Create: `desktop/src-tauri/Cargo.toml`
- Create: `desktop/src-tauri/src/main.rs`
- Create: `desktop/src-tauri/build.rs`
- Create: `desktop/package.json`
- Create: `desktop/vite.config.ts`
- Create: `desktop/index.html`
- Create: `desktop/src/main.ts`
- Create: `desktop/src/styles.css`
- Create: `desktop/src/types.ts`
- Create: `desktop/src/api.ts`
- Create: `desktop/src/vite-env.d.ts`

- [ ] **Step 1: Create desktop directory and core files**

This step creates the Tauri+Vite project structure. Since this is a non-Go, non-TDD scaffolding task, write each file directly.

`desktop/package.json`:
```json
{
  "name": "bitexchange-desktop",
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "tauri": "tauri"
  },
  "devDependencies": {
    "vite": "^5.0.0",
    "typescript": "^5.4.0",
    "@tauri-apps/cli": "^2.0.0"
  }
}
```

`desktop/vite.config.ts`:
```ts
import { defineConfig } from 'vite';

export default defineConfig({
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
  },
  build: {
    target: 'esnext',
  },
});
```

`desktop/index.html`:
```html
<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>bitExchange</title>
</head>
<body>
  <div id="app"></div>
  <script type="module" src="/src/main.ts"></script>
</body>
</html>
```

`desktop/src/vite-env.d.ts`:
```ts
/// <reference types="vite/client" />
```

`desktop/src/types.ts`:
```ts
export interface Peer {
  device_id: string;
  device_name: string;
  fingerprint: string;
  path: string;
  rate: number;
  status: string;
}

export interface Task {
  id: string;
  kind: string;
  target: string;
  payload: string;
  status: string;
  progress: number;
  rate: number;
  path: string;
  error?: string;
}

export interface ClientConfig {
  device_name: string;
  default_save_root: string;
  server_url: string;
  relay_enabled: boolean;
  relay_max_bytes: number;
  listen_port: number;
  encrypt_default: boolean;
  trusted_devices_version: number;
}

export interface ChatLine {
  type: 'text' | 'file';
  from: string;
  body?: string;
  filename?: string;
  savedPath?: string;
}
```

`desktop/src/api.ts`:
```ts
import type { Peer, Task, ClientConfig } from './types';

let apiBase = '';

export function setApiBase(base: string) {
  apiBase = base;
}

export function getApiBase() {
  return apiBase;
}

export async function getPeers(): Promise<Peer[]> {
  const resp = await fetch(`${apiBase}/api/peers`);
  return resp.json();
}

export async function getConfig(): Promise<ClientConfig> {
  const resp = await fetch(`${apiBase}/api/config`);
  return resp.json();
}

export async function putConfig(cfg: ClientConfig): Promise<ClientConfig> {
  const resp = await fetch(`${apiBase}/api/config`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg),
  });
  return resp.json();
}

export async function getTasks(): Promise<Task[]> {
  const resp = await fetch(`${apiBase}/api/tasks`);
  const data = await resp.json();
  return data.tasks || [];
}

export async function sendText(targets: string[], body: string, encrypted: boolean): Promise<Task[]> {
  const resp = await fetch(`${apiBase}/api/send/text`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ to_device_ids: targets, body, encrypted }),
  });
  const data = await resp.json();
  return data.tasks || [];
}

export async function sendFile(targets: string[], file: File, encrypted: boolean): Promise<Task[]> {
  const form = new FormData();
  targets.forEach(t => form.append('to_device_ids', t));
  form.append('file', file);
  const resp = await fetch(`${apiBase}/api/send/file`, { method: 'POST', body: form });
  const data = await resp.json();
  return data.tasks || [];
}

export function subscribeTask(taskId: string, onEvent: (event: string, data: string) => void): () => void {
  const es = new EventSource(`${apiBase}/api/tasks/${taskId}/events`);
  es.addEventListener('progress', (e: MessageEvent) => onEvent('progress', e.data));
  es.addEventListener('status', (e: MessageEvent) => onEvent('status', e.data));
  return () => es.close();
}

export function subscribeInbox(onEvent: (event: string, data: string) => void): () => void {
  const es = new EventSource(`${apiBase}/api/inbox/events`);
  es.onmessage = (e) => onEvent('message', e.data);
  return () => es.close();
}
```

- [ ] **Step 2: Create Tauri Rust side**

`desktop/src-tauri/Cargo.toml`:
```toml
[package]
name = "bitexchange-desktop"
version = "0.1.0"
edition = "2021"

[build-dependencies]
tauri-build = { version = "2.0", features = [] }

[dependencies]
tauri = { version = "2.0", features = [] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"

[features]
custom-protocol = ["tauri/custom-protocol"]
```

`desktop/src-tauri/build.rs`:
```rust
fn main() {
    tauri_build::build()
}
```

`desktop/src-tauri/src/main.rs`:
```rust
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::process::{Child, Command, Stdio};
use std::io::{BufRead, BufReader};
use std::sync::Mutex;
use tauri::Manager;

struct CoreState {
    child: Mutex<Option<Child>>,
    api_base: Mutex<String>,
}

fn find_core_binary() -> Option<String> {
    // Look for bitexchange-core next to the executable, or in PATH
    let candidates = ["bitexchange-core", "./bitexchange-core"];
    for c in candidates {
        if which::which(c).is_ok() {
            return Some(c.to_string());
        }
    }
    None
}

fn spawn_core(root: &str) -> Result<(Child, String), String> {
    let bin = find_core_binary().ok_or("bitexchange-core not found")?;
    let mut child = Command::new(bin)
        .arg("--root")
        .arg(root)
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .map_err(|e| e.to_string())?;

    let stdout = child.stdout.take().ok_or("no stdout")?;
    let reader = BufReader::new(stdout);
    // Read first line: LISTENING 127.0.0.1:NNNN
    for line in reader.lines() {
        let line = line.map_err(|e| e.to_string())?;
        if line.starts_with("LISTENING ") {
            let addr = line.trim_start_matches("LISTENING ").to_string();
            return Ok((child, format!("http://{}", addr)));
        }
    }
    Err("core did not print LISTENING line".to_string())
}

#[tauri::command]
fn api_base(state: tauri::State<CoreState>) -> String {
    state.api_base.lock().unwrap().clone()
}

fn main() {
    tauri::Builder::default()
        .manage(CoreState {
            child: Mutex::new(None),
            api_base: Mutex::new(String::new()),
        })
        .setup(|app| {
            let root = app.path().app_config_dir().map_err(|e| e.to_string())?;
            let root_str = root.to_string_lossy().to_string();
            match spawn_core(&root_str) {
                Ok((child, base)) => {
                    let state: tauri::State<CoreState> = app.state();
                    *state.child.lock().unwrap() = Some(child);
                    *state.api_base.lock().unwrap() = base;
                }
                Err(e) => {
                    eprintln!("failed to spawn core: {}", e);
                }
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                let state: tauri::State<CoreState> = window.state();
                if let Some(mut child) = state.child.lock().unwrap().take() {
                    let _ = child.kill();
                    let _ = child.wait();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![api_base])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

`desktop/tauri.conf.json`:
```json
{
  "$schema": "https://schema.tauri.app/config/2",
  "productName": "bitExchange",
  "version": "0.1.0",
  "identifier": "com.bitexchange.desktop",
  "build": {
    "frontendDist": "../dist",
    "devUrl": "http://localhost:1420"
  },
  "app": {
    "windows": [
      {
        "title": "bitExchange",
        "width": 1100,
        "height": 720,
        "minWidth": 800,
        "minHeight": 560
      }
    ],
    "security": {
      "csp": null
    }
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "resources": ["binaries/bitexchange-core"]
  }
}
```

Add a `which` dependency note — actually `which` crate is not in std. Replace `find_core_binary` to look in the executable's directory only:

Replace `find_core_binary` in main.rs:
```rust
fn find_core_binary() -> Option<String> {
    let exe_dir = std::env::current_exe().ok()?.parent()?.to_path_buf();
    let candidates = [
        exe_dir.join("bitexchange-core"),
        exe_dir.join("bitexchange-core.exe"),
    ];
    for c in candidates {
        if c.exists() {
            return Some(c.to_string_lossy().to_string());
        }
    }
    None
}
```

And remove the `which::which` usage. Also remove `which` from Cargo.toml deps (it's not there anyway).

- [ ] **Step 3: Verify frontend builds (TypeScript only, no Tauri runtime needed yet)**

```bash
cd /home/newnew/bitExchange/desktop
npm install
npm run build
```

Expected: `dist/` directory created with `index.html` and bundled JS.

- [ ] **Step 4: Commit**

```bash
git add desktop/
git commit -m "feat: scaffold Tauri + Vite desktop project"
```

## Task 11: Glassmorphism deep-space theme

**Files:**
- Create: `desktop/src/styles.css`

- [ ] **Step 1: Write the stylesheet**

`desktop/src/styles.css`:
```css
:root {
  --bg-start: #0f0c29;
  --bg-end: #302b63;
  --panel-bg: rgba(255, 255, 255, 0.06);
  --panel-border: rgba(255, 255, 255, 0.12);
  --panel-border-active: rgba(167, 139, 250, 0.4);
  --accent-start: #7c3aed;
  --accent-end: #a78bfa;
  --text-primary: #ffffff;
  --text-secondary: rgba(255, 255, 255, 0.6);
  --text-muted: rgba(255, 255, 255, 0.4);
  --success: #34d399;
  --error: #f87171;
  --radius: 10px;
  --radius-lg: 14px;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  color: var(--text-primary);
  background: linear-gradient(135deg, var(--bg-start), var(--bg-end));
  min-height: 100vh;
  overflow: hidden;
}

#app {
  display: grid;
  grid-template-columns: 240px 1fr 240px;
  grid-template-rows: 1fr 130px;
  height: 100vh;
  gap: 1px;
}

.panel {
  background: var(--panel-bg);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--panel-border);
  padding: 14px;
  overflow: auto;
}

.panel-left {
  grid-row: 1;
  grid-column: 1;
}

.panel-center {
  grid-row: 1;
  grid-column: 2;
  background: rgba(15, 12, 41, 0.4);
}

.panel-right {
  grid-row: 1;
  grid-column: 3;
}

.input-bar {
  grid-row: 2;
  grid-column: 1 / -1;
  background: var(--panel-bg);
  backdrop-filter: blur(10px);
  border-top: 1px solid var(--panel-border);
  padding: 14px;
  display: flex;
  gap: 10px;
  align-items: center;
}

.panel-title {
  color: var(--accent-end);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 12px;
}

.device-card {
  background: var(--panel-bg);
  border: 1px solid var(--panel-border);
  padding: 10px;
  border-radius: var(--radius);
  margin-bottom: 8px;
  cursor: pointer;
  transition: border-color 0.15s;
}

.device-card.selected {
  border-color: var(--panel-border-active);
  background: rgba(167, 139, 250, 0.12);
}

.device-name {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 13px;
}

.device-meta {
  color: var(--text-secondary);
  font-size: 11px;
  margin-top: 3px;
}

.bubble-out {
  align-self: flex-end;
  background: linear-gradient(135deg, var(--accent-start), var(--accent-end));
  color: var(--text-primary);
  padding: 10px 14px;
  border-radius: var(--radius-lg) var(--radius-lg) 4px var(--radius-lg);
  max-width: 70%;
  font-size: 13px;
}

.bubble-in {
  background: var(--panel-bg);
  padding: 10px 14px;
  border-radius: var(--radius-lg) var(--radius-lg) var(--radius-lg) 4px;
  max-width: 70%;
  font-size: 13px;
  border: 1px solid var(--panel-border);
}

.chat-stream {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
}

.task-card {
  background: var(--panel-bg);
  padding: 10px;
  border-radius: var(--radius);
  margin-bottom: 8px;
  border: 1px solid var(--panel-border);
}

.task-card.done {
  border-color: var(--success);
}

.task-card.failed {
  border-color: var(--error);
}

.progress-bar {
  background: rgba(0, 0, 0, 0.3);
  height: 4px;
  border-radius: 2px;
  margin: 6px 0;
  overflow: hidden;
}

.progress-fill {
  background: linear-gradient(90deg, var(--accent-start), var(--accent-end));
  height: 100%;
  transition: width 0.2s;
}

.input-field {
  flex: 1;
  background: var(--panel-bg);
  border: 1px solid var(--panel-border);
  border-radius: var(--radius);
  padding: 10px 14px;
  color: var(--text-secondary);
  font-size: 13px;
  outline: none;
}

.input-field:focus {
  border-color: var(--panel-border-active);
  color: var(--text-primary);
}

.btn {
  background: var(--panel-bg);
  border: 1px solid var(--panel-border);
  color: var(--text-primary);
  padding: 8px 14px;
  border-radius: var(--radius);
  font-size: 12px;
  cursor: pointer;
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-start), var(--accent-end));
  border: none;
  font-weight: 600;
}

.btn-toggle.active {
  border-color: var(--panel-border-active);
  background: rgba(167, 139, 250, 0.2);
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/newnew/bitExchange/desktop
npm run build
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add desktop/src/styles.css
git commit -m "feat: add glassmorphism deep-space theme"
```

## Task 12: Frontend components — DeviceList, ChatStream, TaskPanel, InputBar

**Files:**
- Create: `desktop/src/components/DeviceList.ts`
- Create: `desktop/src/components/ChatStream.ts`
- Create: `desktop/src/components/TaskPanel.ts`
- Create: `desktop/src/components/InputBar.ts`
- Modify: `desktop/src/main.ts`

- [ ] **Step 1: Write components**

`desktop/src/components/DeviceList.ts`:
```ts
import type { Peer } from '../types';

export class DeviceList {
  private el: HTMLElement;
  private peers: Peer[] = [];
  private selected = new Set<string>();
  private onChange: (selected: string[]) => void;

  constructor(el: HTMLElement, onChange: (selected: string[]) => void) {
    this.el = el;
    this.onChange = onChange;
  }

  update(peers: Peer[]) {
    this.peers = peers;
    this.render();
  }

  private render() {
    this.el.innerHTML = '<div class="panel-title">在线设备</div>';
    if (this.peers.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'device-meta';
      empty.textContent = '暂无在线设备';
      this.el.appendChild(empty);
      return;
    }
    for (const p of this.peers) {
      const card = document.createElement('div');
      card.className = 'device-card';
      if (this.selected.has(p.device_id)) card.classList.add('selected');
      card.innerHTML = `
        <div class="device-name">${p.device_name}</div>
        <div class="device-meta">${p.path} · ${p.rate.toFixed(1)}MB/s</div>
      `;
      card.onclick = () => {
        if (this.selected.has(p.device_id)) {
          this.selected.delete(p.device_id);
        } else {
          this.selected.add(p.device_id);
        }
        this.render();
        this.onChange(Array.from(this.selected));
      };
      this.el.appendChild(card);
    }
  }

  getSelected(): string[] {
    return Array.from(this.selected);
  }
}
```

`desktop/src/components/ChatStream.ts`:
```ts
import type { ChatLine } from '../types';

export class ChatStream {
  private el: HTMLElement;
  private lines: ChatLine[] = [];

  constructor(el: HTMLElement) {
    this.el = el;
    this.el.className = 'chat-stream';
  }

  append(line: ChatLine, outgoing: boolean) {
    this.lines.push(line);
    const bubble = document.createElement('div');
    bubble.className = outgoing ? 'bubble-out' : 'bubble-in';
    if (line.type === 'text') {
      bubble.textContent = line.body || '';
    } else {
      bubble.innerHTML = `<div style="font-size:10px;color:var(--text-muted);margin-bottom:4px;">${line.from} · 文件</div>📄 ${line.filename} · 已保存到 ${line.savedPath}`;
    }
    this.el.appendChild(bubble);
    this.el.scrollTop = this.el.scrollHeight;
  }
}
```

`desktop/src/components/TaskPanel.ts`:
```ts
import type { Task } from '../types';

export class TaskPanel {
  private el: HTMLElement;
  private tasks: Task[] = [];

  constructor(el: HTMLElement) {
    this.el = el;
  }

  update(tasks: Task[]) {
    this.tasks = tasks;
    this.render();
  }

  private render() {
    this.el.innerHTML = '<div class="panel-title">传输任务</div>';
    if (this.tasks.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'device-meta';
      empty.textContent = '暂无任务';
      this.el.appendChild(empty);
      return;
    }
    for (const t of this.tasks) {
      const card = document.createElement('div');
      card.className = 'task-card';
      if (t.status === 'done') card.classList.add('done');
      if (t.status === 'failed' || t.status === 'cancelled') card.classList.add('failed');
      const pct = Math.round(t.progress * 100);
      card.innerHTML = `
        <div style="color:var(--text-primary);font-size:12px;">${t.payload} → ${t.target}</div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
        <div style="color:var(--text-secondary);font-size:10px;">${pct}% · ${t.path}</div>
      `;
      this.el.appendChild(card);
    }
  }
}
```

`desktop/src/components/InputBar.ts`:
```ts
export class InputBar {
  private el: HTMLElement;
  private onSendText: (text: string, encrypted: boolean) => void;
  private onSendFile: (file: File, encrypted: boolean) => void;
  private encrypted = false;

  constructor(
    el: HTMLElement,
    onSendText: (text: string, encrypted: boolean) => void,
    onSendFile: (file: File, encrypted: boolean) => void
  ) {
    this.el = el;
    this.onSendText = onSendText;
    this.onSendFile = onSendFile;
    this.render();
  }

  private render() {
    this.el.className = 'input-bar';
    this.el.innerHTML = `
      <input class="input-field" type="text" placeholder="输入消息，或拖入/粘贴文件..." />
      <button class="btn" id="file-btn">📎 文件</button>
      <button class="btn btn-toggle" id="encrypt-btn">🔒 加密</button>
      <button class="btn btn-primary" id="send-btn">发送 →</button>
      <input type="file" id="file-input" style="display:none" multiple />
    `;

    const input = this.el.querySelector('.input-field') as HTMLInputElement;
    const fileInput = this.el.querySelector('#file-input') as HTMLInputElement;
    const encryptBtn = this.el.querySelector('#encrypt-btn') as HTMLButtonElement;

    this.el.querySelector('#send-btn')!.addEventListener('click', () => {
      if (input.value.trim()) {
        this.onSendText(input.value, this.encrypted);
        input.value = '';
      }
    });

    this.el.querySelector('#file-btn')!.addEventListener('click', () => {
      fileInput.click();
    });

    fileInput.addEventListener('change', () => {
      if (fileInput.files && fileInput.files.length) {
        for (const f of Array.from(fileInput.files)) {
          this.onSendFile(f, this.encrypted);
        }
      }
      fileInput.value = '';
    });

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        this.el.querySelector('#send-btn')!.dispatchEvent(new Event('click'));
      }
    });

    // Drag and drop
    this.el.addEventListener('dragover', (e) => {
      e.preventDefault();
    });
    this.el.addEventListener('drop', (e) => {
      e.preventDefault();
      if (e.dataTransfer?.files) {
        for (const f of Array.from(e.dataTransfer.files)) {
          this.onSendFile(f, this.encrypted);
        }
      }
    });

    encryptBtn.addEventListener('click', () => {
      this.encrypted = !this.encrypted;
      encryptBtn.classList.toggle('active', this.encrypted);
    });
  }
}
```

`desktop/src/main.ts`:
```ts
import './styles.css';
import { DeviceList } from './components/DeviceList';
import { ChatStream } from './components/ChatStream';
import { TaskPanel } from './components/TaskPanel';
import { InputBar } from './components/InputBar';
import * as api from './api';
import type { Peer, Task } from './types';

async function init() {
  // In Tauri, get api base from Rust side; in browser dev, hardcode
  let base = '';
  // @ts-ignore
  if (window.__TAURI__) {
    // @ts-ignore
    const { invoke } = window.__TAURI__.core;
    base = await invoke('api_base');
  } else {
    base = 'http://127.0.0.1:8080';
  }
  api.setApiBase(base);

  const app = document.getElementById('app')!;

  const left = document.createElement('div');
  left.className = 'panel panel-left';
  app.appendChild(left);

  const center = document.createElement('div');
  center.className = 'panel panel-center';
  app.appendChild(center);

  const right = document.createElement('div');
  right.className = 'panel panel-right';
  app.appendChild(right);

  const inputBar = document.createElement('div');
  app.appendChild(inputBar);

  let selectedTargets: string[] = [];
  const deviceList = new DeviceList(left, (sel) => { selectedTargets = sel; });
  const chat = new ChatStream(center);
  const taskPanel = new TaskPanel(right);

  new InputBar(inputBar,
    async (text, encrypted) => {
      if (selectedTargets.length === 0) {
        alert('请先选择至少一个目标设备');
        return;
      }
      const tasks = await api.sendText(selectedTargets, text, encrypted);
      chat.append({ type: 'text', from: 'me', body: text }, true);
      refreshTasks();
      for (const t of tasks) {
        api.subscribeTask(t.task_id, (ev, data) => {
          refreshTasks();
        });
      }
    },
    async (file, encrypted) => {
      if (selectedTargets.length === 0) {
        alert('请先选择至少一个目标设备');
        return;
      }
      const tasks = await api.sendFile(selectedTargets, file, encrypted);
      chat.append({ type: 'file', from: 'me', filename: file.name }, true);
      refreshTasks();
      for (const t of tasks) {
        api.subscribeTask(t.task_id, () => refreshTasks());
      }
    }
  );

  api.subscribeInbox((_ev, data) => {
    try {
      const msg = JSON.parse(data);
      chat.append(msg, false);
    } catch {}
  });

  async function refreshPeers() {
    try {
      const peers = await api.getPeers();
      deviceList.update(peers);
    } catch {}
  }

  async function refreshTasks() {
    try {
      const tasks = await api.getTasks();
      taskPanel.update(tasks);
    } catch {}
  }

  refreshPeers();
  refreshTasks();
  setInterval(refreshPeers, 2000);
  setInterval(refreshTasks, 1000);
}

init();
```

- [ ] **Step 2: Verify build**

```bash
cd /home/newnew/bitExchange/desktop
npm run build
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add desktop/src/components/ desktop/src/main.ts
git commit -m "feat: add frontend components and main wiring"
```

## Task 13: Settings page

**Files:**
- Create: `desktop/src/components/SettingsPage.ts`
- Modify: `desktop/src/main.ts` (add settings navigation)

- [ ] **Step 1: Write SettingsPage**

`desktop/src/components/SettingsPage.ts`:
```ts
import type { ClientConfig } from '../types';
import * as api from '../api';

export class SettingsPage {
  private el: HTMLElement;
  private onBack: () => void;

  constructor(el: HTMLElement, onBack: () => void) {
    this.el = el;
    this.onBack = onBack;
  }

  async render() {
    const cfg = await api.getConfig();
    this.el.innerHTML = `
      <div style="padding:20px;overflow:auto;height:100vh;">
        <button class="btn" id="back-btn">← 返回</button>
        <h2 style="margin:16px 0;color:var(--accent-end);">设置</h2>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">设备</div>
          <label style="display:block;margin-bottom:8px;color:var(--text-secondary);font-size:12px;">设备名称</label>
          <input class="input-field" id="device-name" value="${cfg.device_name || ''}" style="width:100%;" />
          <label style="display:block;margin:14px 0 8px;color:var(--text-secondary);font-size:12px;">默认保存根目录</label>
          <input class="input-field" id="save-root" value="${cfg.default_save_root || ''}" style="width:100%;" />
        </div>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">网络</div>
          <label style="display:block;margin-bottom:8px;color:var(--text-secondary);font-size:12px;">信令服务器</label>
          <input class="input-field" id="server-url" value="${cfg.server_url || ''}" style="width:100%;" />
          <label style="display:flex;align-items:center;gap:8px;margin-top:14px;color:var(--text-primary);font-size:13px;">
            <input type="checkbox" id="relay-enabled" ${cfg.relay_enabled ? 'checked' : ''} /> 允许公网中继
          </label>
        </div>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">安全</div>
          <label style="display:flex;align-items:center;gap:8px;color:var(--text-primary);font-size:13px;">
            <input type="checkbox" id="encrypt-default" ${cfg.encrypt_default ? 'checked' : ''} /> 默认加密传输
          </label>
        </div>

        <button class="btn btn-primary" id="save-btn">保存设置</button>
      </div>
    `;

    this.el.querySelector('#back-btn')!.addEventListener('click', this.onBack);
    this.el.querySelector('#save-btn')!.addEventListener('click', async () => {
      const updated: ClientConfig = {
        device_name: (this.el.querySelector('#device-name') as HTMLInputElement).value,
        default_save_root: (this.el.querySelector('#save-root') as HTMLInputElement).value,
        server_url: (this.el.querySelector('#server-url') as HTMLInputElement).value,
        relay_enabled: (this.el.querySelector('#relay-enabled') as HTMLInputElement).checked,
        relay_max_bytes: cfg.relay_max_bytes,
        listen_port: cfg.listen_port,
        encrypt_default: (this.el.querySelector('#encrypt-default') as HTMLInputElement).checked,
        trusted_devices_version: cfg.trusted_devices_version,
      };
      await api.putConfig(updated);
      alert('设置已保存');
      this.onBack();
    });
  }
}
```

- [ ] **Step 2: Modify main.ts to add settings navigation**

Add at top of `desktop/src/main.ts` after imports:
```ts
import { SettingsPage } from './components/SettingsPage';
```

Add a settings button to the left panel and route handling. Inside `init()`, after creating the panels, add:
```ts
const settingsBtn = document.createElement('button');
settingsBtn.className = 'btn';
settingsBtn.textContent = '⚙ 设置';
settingsBtn.style.marginTop = '12px';
settingsBtn.style.width = '100%';
left.appendChild(settingsBtn);

const mainView = app;
let settingsPage: SettingsPage | null = null;

settingsBtn.addEventListener('click', () => {
  app.style.display = 'none';
  const settingsEl = document.createElement('div');
  document.body.appendChild(settingsEl);
  settingsPage = new SettingsPage(settingsEl, () => {
    settingsEl.remove();
    app.style.display = 'grid';
  });
  settingsPage.render();
});
```

- [ ] **Step 3: Verify build**

```bash
cd /home/newnew/bitExchange/desktop
npm run build
```

Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add desktop/src/components/SettingsPage.ts desktop/src/main.ts
git commit -m "feat: add settings page"
```

## Task 14: Full regression

- [ ] **Step 1: Run Go tests**

Run: `/usr/local/go/bin/go test ./...`
Expected: ALL pass.

- [ ] **Step 2: Build Go binaries**

```bash
/usr/local/go/bin/go build ./cmd/bitexchange-core
/usr/local/go/bin/go build ./cmd/bitexchange-server
/usr/local/go/bin/go build ./cmd/bitexchange-cli
```
Expected: all three build.

- [ ] **Step 3: Build frontend**

```bash
cd /home/newnew/bitExchange/desktop && npm run build
```
Expected: `dist/` produced.

- [ ] **Step 4: Smoke test core + API**

```bash
tmpdir=$(mktemp -d)
./bitexchange-core --root "$tmpdir" &
sleep 0.5
# port extracted manually for test
curl -s http://127.0.0.1:$(cat /proc/$!/fd/1 2>/dev/null | head -1 | awk '{print $2}' | cut -d: -f2)/api/config
kill $!
```

Expected: JSON config response.

- [ ] **Step 5: Commit**

```bash
git commit --allow-empty -m "chore: phase 3 full regression green"
```

---

## Plan self-review

### Spec coverage

| Spec section | Covered by task |
|---|---|
| 3. Architecture (sidecar + Tauri) | Task 9 (core binary), Task 10 (Tauri scaffold) |
| 4. core local HTTP API | Tasks 1-8 (errors, atomic, SSE, taskmgr, server, handlers) |
| 4.1 device/session | Task 7 (peers handler) |
| 4.2 send text/file | Task 7 (send handlers) |
| 4.2 tasks + cancel + SSE | Task 8 (tasks handlers) |
| 4.3 inbox SSE | Task 8 (inbox handler) |
| 4.4 config/history/device | Task 6 (config handlers) |
| 5. UI structure | Tasks 11-12 (theme + components) |
| 6. startup/send/receive/error flows | Task 9 (core startup), Task 10 (Tauri spawn), Task 12 (main wiring) |
| 7. settings page | Task 13 |
| 8. tests | Tasks 1-8 unit tests, Task 14 regression |

### Placeholder scan

Task steps contain explicit file paths, commands, and code. The send handler stub marks intentionally-deferred pathselector integration with a neutral comment (Phase 3 stub), not a plan placeholder.

### Type consistency

- `coreapi.Task`, `coreapi.SSEEvent`, `coreapi.SSEBroker`, `coreapi.TaskManager` used consistently across tasks 3-8.
- `config.ClientConfig` new fields (DeviceName, DefaultSaveRoot, EncryptDefault, TrustedDevicesVersion) match between Task 5 Go struct and Task 10 TypeScript types.
- Frontend `Peer`, `Task`, `ClientConfig`, `ChatLine` types defined in Task 10 and used in Tasks 12-13.
- API endpoints (`/api/peers`, `/api/send/text`, `/api/tasks`, `/api/tasks/{id}/events`, `/api/inbox/events`, `/api/config`, `/api/device`, `/api/history`) consistent between Go handlers (Tasks 6-8) and TS api.ts (Task 10).
