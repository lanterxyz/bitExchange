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
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			t.Fatalf("task id should not be empty")
		}
		if seen[id] {
			t.Fatalf("duplicate task id %q", id)
		}
		seen[id] = true
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
