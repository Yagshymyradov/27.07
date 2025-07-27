package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("task not found")
	ErrTooManyUtems = errors.New("task already has 3 items")
	ErrTaskFinalized = errors.New("taskt is finalized and cannot be modified")
)

type Store interface {
	Create() *Task
	Get(id string) (*Task, error)
	AddItem(id string, item string) error
}

type memoryStore struct {
	mu sync.RWMutex
	tasks map[string]*Task
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{
		tasks: make(map[string]*Task),
	}
}

func (m *memoryStore) Create() *Task {
	task := &Task{
		ID: newID(),
		Status: StatusNew,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.mu.Lock()
	m.tasks[task.ID] = task
	m.mu.Unlock()
	return task
}

func (m *memoryStore) Get(id string) (*Task, error) {
	m.mu.RLock()
	t, ok := m.tasks[id]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (m *memoryStore) AddItem(id string, item string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	if len(t.Items) >= 3 {
		return ErrTooManyUtems
	}
	if t.Status != StatusNew && t.Status != StatusPending {
		return ErrTaskFinalized
	}

	t.Items = append(t.Items, Item{
		URL: item,
		Status: ItemPending,
	})

	if len(t.Items) == 3 {
		t.Status = StatusPending
	}
	t.UpdatedAt = time.Now()
	return nil
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}