package event

import (
	"context"

	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
)

type CompletionPublisher interface {
	PublishTaskCompleted(ctx context.Context, completed domainevent.TaskCompleted) error
}
