package taskrepository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Memory struct {
	mu    sync.Mutex
	tasks map[uuid.UUID]*aggregate.Task
}

func (m *Memory) ensure() {
	if m.tasks == nil {
		m.tasks = map[uuid.UUID]*aggregate.Task{}
	}
}

func (m *Memory) clone(task *aggregate.Task) *aggregate.Task {
	return aggregate.RestoreTask(task.ID(), task.Title(), task.Status(), task.DueDate(), task.CreatedAt(), task.CompletedAt())
}

func (m *Memory) Get(_ context.Context, id uuid.UUID) (*aggregate.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	task, ok := m.tasks[id]
	if !ok {
		return nil, errs.New(errs.CodeTaskNotFound, errs.KindNotFound, fmt.Errorf("task %s not found", id))
	}
	return m.clone(task), nil
}

func (m *Memory) Save(_ context.Context, task *aggregate.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	m.tasks[task.ID()] = m.clone(task)
	return nil
}

func (m *Memory) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
