package aggregate_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var testOwnerID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

type taskNewCase struct {
	name      string
	ownerID   uuid.UUID
	title     string
	dueDate   *time.Time
	now       time.Time
	wantErr   bool
	wantCode  string
	wantTitle string
}

type taskTransitionCase struct {
	name     string
	setup    func(*aggregate.Task) error
	action   func(*aggregate.Task, time.Time) error
	wantErr  bool
	wantCode string
	wantStat enum.Status
}

func TestNewTask(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Millisecond)

	tests := []taskNewCase{
		{
			name:      "reject missing owner id",
			ownerID:   uuid.Nil,
			title:     "Prepare report",
			now:       now,
			wantErr:   true,
			wantCode:  errs.CodeValidationFailed,
		},
		{
			name:      "create with title prepare report and no due date",
			ownerID:   testOwnerID,
			title:     "Prepare report",
			now:       now,
			wantTitle: "Prepare report",
		},
		{
			name:     "reject empty title",
			ownerID:  testOwnerID,
			title:    "",
			now:      now,
			wantErr:  true,
			wantCode: errs.CodeInvalidTitle,
		},
		{
			name:     "reject whitespace only title",
			ownerID:  testOwnerID,
			title:    " \u00a0\t\n",
			now:      now,
			wantErr:  true,
			wantCode: errs.CodeInvalidTitle,
		},
		{
			name:      "trim unicode whitespace and store trimmed title",
			ownerID:   testOwnerID,
			title:     "  Prepare report  ",
			now:       now,
			wantTitle: "Prepare report",
		},
		{
			name:     "reject title longer than 100 code points",
			ownerID:  testOwnerID,
			title:    strings.Repeat("あ", 101),
			now:      now,
			wantErr:  true,
			wantCode: errs.CodeInvalidTitle,
		},
		{
			name:      "accept title of 100 code points",
			ownerID:   testOwnerID,
			title:     strings.Repeat("あ", 100),
			now:       now,
			wantTitle: strings.Repeat("あ", 100),
		},
		{
			name:      "accept due date equal to now",
			ownerID:   testOwnerID,
			title:     "Prepare report",
			dueDate:   &now,
			now:       now,
			wantTitle: "Prepare report",
		},
		{
			name:     "reject due date strictly before now",
			ownerID:  testOwnerID,
			title:    "Prepare report",
			dueDate:  &past,
			now:      now,
			wantErr:  true,
			wantCode: errs.CodeInvalidDueDate,
		},
		{
			name:      "accept future due date",
			ownerID:   testOwnerID,
			title:     "Prepare report",
			dueDate:   &future,
			now:       now,
			wantTitle: "Prepare report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
			task, err := aggregate.NewTask(id, tt.ownerID, tt.title, tt.dueDate, tt.now)
			if tt.wantErr {
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, tt.wantCode, appErr.Code)
				assert.Nil(t, task)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, task)
			assert.Equal(t, enum.Todo, task.Status())
			assert.Equal(t, tt.ownerID, task.OwnerID())
			assert.Equal(t, tt.wantTitle, task.Title())
			assert.Equal(t, tt.now.UTC().Truncate(time.Millisecond), task.CreatedAt())
			assert.Nil(t, task.CompletedAt())
			assert.Equal(t, utf8.RuneCountInString(task.Title()) <= 100, true)
		})
	}
}

func TestTaskLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	tests := []taskTransitionCase{
		{
			name: "start from todo",
			action: func(task *aggregate.Task, _ time.Time) error {
				return task.Start()
			},
			wantStat: enum.InProgress,
		},
		{
			name: "complete todo is invalid",
			action: func(task *aggregate.Task, at time.Time) error {
				_, err := task.Complete(at)
				return err
			},
			wantErr:  true,
			wantCode: errs.CodeInvalidTransition,
			wantStat: enum.Todo,
		},
		{
			name: "start then complete",
			setup: func(task *aggregate.Task) error {
				return task.Start()
			},
			action: func(task *aggregate.Task, at time.Time) error {
				ev, err := task.Complete(at)
				if err != nil {
					return err
				}
				if ev.TaskID != task.ID() {
					return assert.AnError
				}
				return nil
			},
			wantStat: enum.Done,
		},
		{
			name: "start twice is invalid",
			setup: func(task *aggregate.Task) error {
				return task.Start()
			},
			action: func(task *aggregate.Task, _ time.Time) error {
				return task.Start()
			},
			wantErr:  true,
			wantCode: errs.CodeInvalidTransition,
			wantStat: enum.InProgress,
		},
		{
			name: "modify done is final",
			setup: func(task *aggregate.Task) error {
				if err := task.Start(); err != nil {
					return err
				}
				_, err := task.Complete(now)
				return err
			},
			action: func(task *aggregate.Task, at time.Time) error {
				return task.Reschedule(&at, at)
			},
			wantErr:  true,
			wantCode: errs.CodeTaskFinal,
			wantStat: enum.Done,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", nil, now)
			require.NoError(t, err)
			if tt.setup != nil {
				require.NoError(t, tt.setup(task))
			}
			err = tt.action(task, now)
			if tt.wantErr {
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, tt.wantCode, appErr.Code)
				assert.Equal(t, tt.wantStat, task.Status())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStat, task.Status())
			if tt.wantStat == enum.Done {
				require.NotNil(t, task.CompletedAt())
				assert.Equal(t, now.UTC().Truncate(time.Millisecond), *task.CompletedAt())
			}
		})
	}
}

func TestTaskOverdueAndReschedule(t *testing.T) {
	createdAt := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	due := createdAt.Add(time.Hour)

	t.Run("in progress becomes overdue after due date", func(t *testing.T) {
		task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", &due, createdAt)
		require.NoError(t, err)
		require.NoError(t, task.Start())
		assert.False(t, task.Overdue(due))
		assert.True(t, task.Overdue(due.Add(time.Millisecond)))
	})

	t.Run("done is not overdue after due date", func(t *testing.T) {
		task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", &due, createdAt)
		require.NoError(t, err)
		require.NoError(t, task.Start())
		_, err = task.Complete(due)
		require.NoError(t, err)
		assert.False(t, task.Overdue(due.Add(time.Hour)))
	})

	t.Run("missing due date is never overdue", func(t *testing.T) {
		task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", nil, createdAt)
		require.NoError(t, err)
		assert.False(t, task.Overdue(createdAt.Add(24*time.Hour)))
	})

	t.Run("reschedule to past is rejected", func(t *testing.T) {
		task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", &due, createdAt)
		require.NoError(t, err)
		past := createdAt.Add(-time.Second)
		err = task.Reschedule(&past, createdAt)
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidDueDate, appErr.Code)
	})

	t.Run("clear due date", func(t *testing.T) {
		task, err := aggregate.NewTask(uuid.New(), testOwnerID, "Prepare report", &due, createdAt)
		require.NoError(t, err)
		require.NoError(t, task.Reschedule(nil, createdAt))
		assert.Nil(t, task.DueDate())
	})
}
