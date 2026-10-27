package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/mocks"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

type usecaseCase struct {
	name     string
	run      func(t *testing.T, uc usecase.TaskUseCase, tasks *mocks.MockTaskRepository, pub *mocks.MockCompletionPublisher)
	wantCode string
}

func newUseCase(t *testing.T, clock *testdata.Clock) (usecase.TaskUseCase, *mocks.MockTaskRepository, *mocks.MockCompletionPublisher, uuid.UUID) {
	t.Helper()
	tasks := mocks.NewMockTaskRepository(t)
	pub := mocks.NewMockCompletionPublisher(t)
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	uc := usecase.TaskUseCase{
		Tasks:     tasks,
		Publisher: pub,
		Clock:     clock,
		IDs:       func() uuid.UUID { return id },
	}
	return uc, tasks, pub, id
}

func TestTaskUseCaseScenarios(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()

	tests := []usecaseCase{
		{
			name: "create start complete publishes event",
			run: func(t *testing.T, uc usecase.TaskUseCase, tasks *mocks.MockTaskRepository, pub *mocks.MockCompletionPublisher) {
				var stored *aggregate.Task
				tasks.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, task *aggregate.Task) {
					stored = aggregate.RestoreTask(task.ID(), task.Title(), task.Status(), task.DueDate(), task.CreatedAt(), task.CompletedAt())
				}).Return(nil)
				created, err := uc.Create(ctx, dto.CreateTaskInput{Title: testdata.TitlePrepareReport()})
				require.NoError(t, err)
				assert.Equal(t, string(enum.Todo), created.Status)

				tasks.EXPECT().Get(mock.Anything, created.ID).Return(stored, nil)
				tasks.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, task *aggregate.Task) {
					stored = aggregate.RestoreTask(task.ID(), task.Title(), task.Status(), task.DueDate(), task.CreatedAt(), task.CompletedAt())
				}).Return(nil)
				started, err := uc.Start(ctx, created.ID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.InProgress), started.Status)

				tasks.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
					return fn(txCtx)
				})
				tasks.EXPECT().Get(mock.Anything, created.ID).Return(stored, nil)
				tasks.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, task *aggregate.Task) {
					stored = aggregate.RestoreTask(task.ID(), task.Title(), task.Status(), task.DueDate(), task.CreatedAt(), task.CompletedAt())
				}).Return(nil)
				pub.EXPECT().PublishTaskCompleted(mock.Anything, mock.Anything).Run(func(_ context.Context, completed domainevent.TaskCompleted) {
					assert.Equal(t, created.ID, completed.TaskID)
					assert.Equal(t, now, completed.CompletedAt)
				}).Return(nil)
				completed, err := uc.Complete(ctx, created.ID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.Done), completed.Status)
				require.NotNil(t, completed.CompletedAt)
			},
		},
		{
			name: "complete todo does not publish",
			run: func(t *testing.T, uc usecase.TaskUseCase, tasks *mocks.MockTaskRepository, pub *mocks.MockCompletionPublisher) {
				task, err := aggregate.NewTask(uuid.New(), testdata.TitlePrepareReport(), nil, now)
				require.NoError(t, err)
				tasks.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
					return fn(txCtx)
				})
				tasks.EXPECT().Get(mock.Anything, task.ID()).Return(task, nil)
				_, err = uc.Complete(ctx, task.ID())
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errs.CodeInvalidTransition, appErr.Code)
			},
			wantCode: errs.CodeInvalidTransition,
		},
		{
			name: "missing task",
			run: func(t *testing.T, uc usecase.TaskUseCase, tasks *mocks.MockTaskRepository, _ *mocks.MockCompletionPublisher) {
				id := uuid.New()
				tasks.EXPECT().Get(mock.Anything, id).Return(nil, errs.New(errs.CodeTaskNotFound, errs.KindNotFound, errors.New("missing")))
				_, err := uc.Get(ctx, id)
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errs.CodeTaskNotFound, appErr.Code)
			},
			wantCode: errs.CodeTaskNotFound,
		},
		{
			name: "publish failure rolls back via transaction error",
			run: func(t *testing.T, uc usecase.TaskUseCase, tasks *mocks.MockTaskRepository, pub *mocks.MockCompletionPublisher) {
				task, err := aggregate.NewTask(uuid.New(), testdata.TitlePrepareReport(), nil, now)
				require.NoError(t, err)
				require.NoError(t, task.Start())
				tasks.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
					return fn(txCtx)
				})
				tasks.EXPECT().Get(mock.Anything, task.ID()).Return(task, nil)
				tasks.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				pub.EXPECT().PublishTaskCompleted(mock.Anything, mock.Anything).Return(errors.New("broker down"))
				_, err = uc.Complete(ctx, task.ID())
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errs.CodeEventUnavailable, appErr.Code)
			},
			wantCode: errs.CodeEventUnavailable,
		},
		{
			name: "overdue after time advances",
			run: func(t *testing.T, _ usecase.TaskUseCase, _ *mocks.MockTaskRepository, pub *mocks.MockCompletionPublisher) {
				clock := testdata.NewClock(now)
				uc, tasks, _, id := newUseCase(t, clock)
				due := now.Add(time.Hour)
				tasks.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				created, err := uc.Create(ctx, dto.CreateTaskInput{Title: testdata.TitlePrepareReport(), DueDate: &due})
				require.NoError(t, err)
				task, err := aggregate.NewTask(id, testdata.TitlePrepareReport(), &due, now)
				require.NoError(t, err)
				require.NoError(t, task.Start())
				clock.Advance(2 * time.Hour)
				tasks.EXPECT().Get(mock.Anything, created.ID).Return(task, nil)
				view, err := uc.Get(ctx, created.ID)
				require.NoError(t, err)
				assert.True(t, view.Overdue)
				_ = pub
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := testdata.NewClock(now)
			uc, tasks, pub, _ := newUseCase(t, clock)
			tt.run(t, uc, tasks, pub)
		})
	}
}
