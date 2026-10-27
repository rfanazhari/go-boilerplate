package aggregate

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/event"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var (
	errEmptyTitle     = errors.New("title is empty")
	errTitleTooLong   = errors.New("title exceeds 100 code points")
	errDueDatePast    = errors.New("due date is in the past")
	errCannotStart    = errors.New("task can be started only from todo")
	errCannotComplete = errors.New("task can be completed only from in progress")
	errTaskFinal      = errors.New("done task cannot be modified")
)

type Task struct {
	id          uuid.UUID
	title       string
	status      enum.Status
	dueDate     *time.Time
	createdAt   time.Time
	completedAt *time.Time
}

func normalizeInstant(value time.Time) time.Time {
	return value.UTC().Truncate(time.Millisecond)
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := normalizeInstant(*value)
	return &copied
}

func NewTask(id uuid.UUID, title string, dueDate *time.Time, now time.Time) (*Task, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return nil, errs.New(errs.CodeInvalidTitle, errs.KindInvalidInput, errEmptyTitle)
	}
	if utf8.RuneCountInString(trimmed) > 100 {
		return nil, errs.New(errs.CodeInvalidTitle, errs.KindInvalidInput, errTitleTooLong)
	}
	createdAt := normalizeInstant(now)
	normalizedDue, err := validateDueDate(dueDate, createdAt)
	if err != nil {
		return nil, err
	}
	return &Task{
		id:        id,
		title:     trimmed,
		status:    enum.Todo,
		dueDate:   normalizedDue,
		createdAt: createdAt,
	}, nil
}

func RestoreTask(id uuid.UUID, title string, status enum.Status, dueDate *time.Time, createdAt time.Time, completedAt *time.Time) *Task {
	return &Task{
		id:          id,
		title:       title,
		status:      status,
		dueDate:     cloneTime(dueDate),
		createdAt:   normalizeInstant(createdAt),
		completedAt: cloneTime(completedAt),
	}
}

func validateDueDate(dueDate *time.Time, now time.Time) (*time.Time, error) {
	if dueDate == nil {
		return nil, nil
	}
	normalized := normalizeInstant(*dueDate)
	if normalized.Before(now) {
		return nil, errs.New(errs.CodeInvalidDueDate, errs.KindInvalidInput, errDueDatePast)
	}
	return &normalized, nil
}

func (t *Task) ID() uuid.UUID {
	return t.id
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) Status() enum.Status {
	return t.status
}

func (t *Task) DueDate() *time.Time {
	return cloneTime(t.dueDate)
}

func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

func (t *Task) CompletedAt() *time.Time {
	return cloneTime(t.completedAt)
}

func (t *Task) Start() error {
	if t.status == enum.Done {
		return errs.New(errs.CodeTaskFinal, errs.KindConflict, errTaskFinal)
	}
	if t.status != enum.Todo {
		return errs.New(errs.CodeInvalidTransition, errs.KindConflict, errCannotStart)
	}
	t.status = enum.InProgress
	return nil
}

func (t *Task) Complete(now time.Time) (*event.TaskCompleted, error) {
	if t.status == enum.Done {
		return nil, errs.New(errs.CodeTaskFinal, errs.KindConflict, errTaskFinal)
	}
	if t.status != enum.InProgress {
		return nil, errs.New(errs.CodeInvalidTransition, errs.KindConflict, errCannotComplete)
	}
	completedAt := normalizeInstant(now)
	t.status = enum.Done
	t.completedAt = &completedAt
	return &event.TaskCompleted{TaskID: t.id, CompletedAt: completedAt}, nil
}

func (t *Task) Reschedule(dueDate *time.Time, now time.Time) error {
	if t.status == enum.Done {
		return errs.New(errs.CodeTaskFinal, errs.KindConflict, errTaskFinal)
	}
	normalized, err := validateDueDate(dueDate, normalizeInstant(now))
	if err != nil {
		return err
	}
	t.dueDate = normalized
	return nil
}

func (t *Task) Overdue(now time.Time) bool {
	if t.dueDate == nil || t.status == enum.Done {
		return false
	}
	return normalizeInstant(now).After(*t.dueDate)
}
