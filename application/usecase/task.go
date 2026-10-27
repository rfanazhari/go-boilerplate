package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	appevent "github.com/rfanazhari/go-boilerplate/application/event"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type TaskUseCase struct {
	Tasks     repository.TaskRepository
	Publisher appevent.CompletionPublisher
	Clock     service.Clock
	IDs       func() uuid.UUID
}

func (u TaskUseCase) now() time.Time {
	if u.Clock == nil {
		return time.Now().UTC().Truncate(time.Millisecond)
	}
	return u.Clock.Now().UTC().Truncate(time.Millisecond)
}

func (u TaskUseCase) nextID() uuid.UUID {
	if u.IDs == nil {
		return uuid.New()
	}
	return u.IDs()
}

func view(task *aggregate.Task, now time.Time) dto.TaskView {
	return dto.TaskView{
		ID:          task.ID(),
		OwnerID:     task.OwnerID(),
		Title:       task.Title(),
		Status:      string(task.Status()),
		DueDate:     task.DueDate(),
		Overdue:     task.Overdue(now),
		CreatedAt:   task.CreatedAt(),
		CompletedAt: task.CompletedAt(),
	}
}

func taskNotFoundErr() error {
	return errs.New(errs.CodeTaskNotFound, errs.KindNotFound, errors.New("task not found"))
}

func (u TaskUseCase) authorizeGet(actor dto.TaskActor, task *aggregate.Task) error {
	if actor.Role == enum.RoleAdmin {
		return nil
	}
	if task.OwnerID() != actor.ID {
		return taskNotFoundErr()
	}
	return nil
}

func (u TaskUseCase) authorizeMutate(actor dto.TaskActor, task *aggregate.Task) error {
	if task.OwnerID() != actor.ID {
		return taskNotFoundErr()
	}
	return nil
}

func (u TaskUseCase) Create(ctx context.Context, actor dto.TaskActor, input dto.CreateTaskInput) (dto.TaskView, error) {
	now := u.now()
	task, err := aggregate.NewTask(u.nextID(), actor.ID, input.Title, input.DueDate, now)
	if err != nil {
		return dto.TaskView{}, err
	}
	if err := u.Tasks.Save(ctx, task); err != nil {
		return dto.TaskView{}, wrapInfra(err)
	}
	return view(task, now), nil
}

func (u TaskUseCase) Start(ctx context.Context, actor dto.TaskActor, id uuid.UUID) (dto.TaskView, error) {
	return u.mutate(ctx, actor, id, func(task *aggregate.Task, _ time.Time) error {
		return task.Start()
	})
}

func (u TaskUseCase) Complete(ctx context.Context, actor dto.TaskActor, id uuid.UUID) (dto.TaskView, error) {
	var result dto.TaskView
	err := u.Tasks.RunInTransaction(ctx, func(txCtx context.Context) error {
		now := u.now()
		task, err := u.Tasks.Get(txCtx, id)
		if err != nil {
			return mapGetError(err)
		}
		if err := u.authorizeMutate(actor, task); err != nil {
			return err
		}
		completed, err := task.Complete(now)
		if err != nil {
			return err
		}
		if err := u.Tasks.Save(txCtx, task); err != nil {
			return wrapInfra(err)
		}
		if err := u.Publisher.PublishTaskCompleted(txCtx, *completed); err != nil {
			return errs.New(errs.CodeEventUnavailable, errs.KindUnavailable, err)
		}
		result = view(task, now)
		return nil
	})
	if err != nil {
		return dto.TaskView{}, err
	}
	return result, nil
}

func (u TaskUseCase) Reschedule(ctx context.Context, actor dto.TaskActor, input dto.RescheduleTaskInput) (dto.TaskView, error) {
	return u.mutate(ctx, actor, input.ID, func(task *aggregate.Task, now time.Time) error {
		return task.Reschedule(input.DueDate, now)
	})
}

func (u TaskUseCase) Get(ctx context.Context, actor dto.TaskActor, id uuid.UUID) (dto.TaskView, error) {
	task, err := u.Tasks.Get(ctx, id)
	if err != nil {
		return dto.TaskView{}, mapGetError(err)
	}
	if err := u.authorizeGet(actor, task); err != nil {
		return dto.TaskView{}, err
	}
	return view(task, u.now()), nil
}

func (u TaskUseCase) mutate(ctx context.Context, actor dto.TaskActor, id uuid.UUID, fn func(*aggregate.Task, time.Time) error) (dto.TaskView, error) {
	now := u.now()
	task, err := u.Tasks.Get(ctx, id)
	if err != nil {
		return dto.TaskView{}, mapGetError(err)
	}
	if err := u.authorizeMutate(actor, task); err != nil {
		return dto.TaskView{}, err
	}
	if err := fn(task, now); err != nil {
		return dto.TaskView{}, err
	}
	if err := u.Tasks.Save(ctx, task); err != nil {
		return dto.TaskView{}, wrapInfra(err)
	}
	return view(task, now), nil
}

func mapGetError(err error) error {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		return err
	}
	return wrapInfra(err)
}

func wrapInfra(err error) error {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		return err
	}
	return errs.New(errs.CodeInternalError, errs.KindInternal, err)
}
