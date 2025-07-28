package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("task not found")
	ErrTooManyItems  = errors.New("task already has 3 items")
	ErrTaskFinalized = errors.New("task is finalized and cannot be modified")
)

type Store interface {
	Create() *Task
	Get(id string) (*Task, error)
	// AddItem appends url to task. Returns ready==true when after adding задача содержит 3 items и переведена в StatusPending.
	AddItem(id string, url string) (ready bool, err error)
	// Update выполняет атомарное изменение задачи под блокировкой.
	Update(id string, fn func(t *Task)) error
	// ActiveCount возвращает количество задач, которые ещё не перешли в финальный статус (done|error).
	ActiveCount() int
}

type memoryStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{
		tasks: make(map[string]*Task),
	}
}

func (m *memoryStore) Create() *Task {
	task := &Task{
		ID:        newID(),
		Status:    StatusNew,
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

func (m *memoryStore) AddItem(id string, url string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return false, ErrNotFound
	}
	if len(t.Items) >= 3 {
		return false, ErrTooManyItems
	}
	if t.Status != StatusNew && t.Status != StatusPending {
		return false, ErrTaskFinalized
	}

	t.Items = append(t.Items, Item{
		URL:    url,
		Status: ItemPending,
	})

	ready := false
	if len(t.Items) == 3 {
		t.Status = StatusPending
		ready = true
	}
	t.UpdatedAt = time.Now()
	return ready, nil
}

// Update выполняет атомарное изменение задачи.
func (m *memoryStore) Update(id string, fn func(t *Task)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	fn(t)
	t.UpdatedAt = time.Now()
	return nil
}

// ActiveCount подсчитывает задачи, не завершённые окончательно.
func (m *memoryStore) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, t := range m.tasks {
		if t.Status != StatusDone && t.Status != StatusError {
			n++
		}
	}
	return n
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
