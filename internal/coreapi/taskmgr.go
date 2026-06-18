package coreapi

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var taskIDCounter uint64

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
	mu      sync.Mutex
	tasks   map[string]*Task
	order   []string
	limit   int
	brokers map[string]*SSEBroker
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
	if !ok {
		m.mu.Unlock()
		return
	}
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
	seq := atomic.AddUint64(&taskIDCounter, 1)
	return "task-" + time.Now().Format("20060102150405.000000000") + "-" + strconv.FormatUint(seq, 10)
}
