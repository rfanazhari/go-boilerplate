package service

import (
	"context"
	"sync"

	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
)

type RecordingPublisher struct {
	mu     sync.Mutex
	Events []domainevent.TaskCompleted
}

func (p *RecordingPublisher) PublishTaskCompleted(_ context.Context, completed domainevent.TaskCompleted) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Events = append(p.Events, completed)
	return nil
}
