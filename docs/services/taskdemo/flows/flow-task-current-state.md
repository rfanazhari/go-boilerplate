# Flow: Task tracking (current state)

HTTP task API — `TaskHandler` / `TaskUseCase`

Files:

- `../../../../delivery/http/router.go` — `Server.Router()`
- `../../../../delivery/http/task.go` — `Create()`, `Start()`, `Complete()`, `Reschedule()`, `Get()`
- `../../../../application/usecase/task.go` — `TaskUseCase`
- `../../../../domain/aggregate/task.go` — `Task`
- `../../../../infrastructure/repository/taskrepository/postgres.go` — `Postgres`
- `../../../../infrastructure/service/rabbit.go` — `RabbitPublisher`
- `../../../../infrastructure/runtime/runtime.go` — `Start()`
- `../../../../migrations/pgsql/20261027204911_create_tasks.up.sql`

Markers used below:

- `[NO USER/OWNER]` — the type, query, or payload has no user, account, or owner concept.
- `[NEEDS USER CONTEXT]` — the call currently receives only `r.Context()` plus a task id or body. A token-derived identity has no field to land in.
- `[EVENT PAYLOAD]` — RabbitMQ `TaskCompleted` body, as published today.

---

## Summary

Any HTTP client can create, start, complete, reschedule, and read a task. The service does not authenticate the caller and does not record who owns the task. Success is a JSON envelope whose `data` is the task view: create returns `201`; start, complete, reschedule, and get return `200`.

The only integration event is `TaskCompleted`. `Complete` saves the task and then publishes that event synchronously before the database transaction commits. This repository declares the RabbitMQ exchange and queue and publishes to them. It does not consume the queue.

`[NO USER/OWNER]` runs through the handler, use-case inputs, aggregate, repository key, table, and event. `[NEEDS USER CONTEXT]` is the HTTP edge and every use-case method that today identifies a task only by UUID. `[EVENT PAYLOAD]` is two JSON fields, `taskId` and `completedAt`.

---

## Full Flow

### Router and middleware

`Server.Router()` builds one `mux.Router`. There is no subrouter and no route group.

```
Server.Router()
  │
  ├─ [PRE-CHECK] router.Use(lmgorilla.Middleware(s.Log))
  │    └─ only middleware; applies to /health and every /tasks route
  │    └─ logs the request; startup masks Authorization, Cookie, token, password
  │    └─ does not read a token or attach a user to the context
  │
  ├─ GET  /health                     Server.health          [NO USER/OWNER]
  ├─ POST /tasks                      TaskHandler.Create     [NEEDS USER CONTEXT]
  ├─ POST /tasks/{id}/start           TaskHandler.Start      [NEEDS USER CONTEXT]
  ├─ POST /tasks/{id}/complete        TaskHandler.Complete   [NEEDS USER CONTEXT]
  ├─ PATCH /tasks/{id}/due-date       TaskHandler.Reschedule [NEEDS USER CONTEXT]
  └─ GET  /tasks/{id}                 TaskHandler.Get        [NEEDS USER CONTEXT]
       │
       ├─ unmatched method → 405 METHOD_NOT_ALLOWED
       └─ unmatched path   → 404 ROUTE_NOT_FOUND

Composition (infrastructure/runtime.Start)
  │
  ├─ INFRA_OPTIONAL=false
  │    ├─ DATABASE_URL required → Postgres
  │    └─ RABBITMQ_URL required → RabbitPublisher
  │         └─ DeclareTaskCompleted(exchange, queue) before listen
  └─ INFRA_OPTIONAL=true
       ├─ Memory repository, replaced by Postgres only if the pool pings
       └─ RecordingPublisher, replaced by RabbitPublisher only if declare succeeds
```

`GET /health` stays on the same router and returns `200` with `data: {}`. The OpenAPI document marks that route `security: []`. Task routes declare no security scheme. Nothing in `../../../../delivery` inspects `Authorization`.

Logging is the only middleware slot. A later authenticator would be a new `router.Use` or a new route group; neither exists. Health and task routes cannot be separated without changing this registration.

### Shared handler behavior

```
writeSuccess
  └─ Envelope{ status: "success", data: taskJSON }     [NO USER/OWNER]

writeError
  ├─ KindInvalidInput → 400
  ├─ KindNotFound     → 404
  ├─ KindConflict     → 409
  ├─ KindUnavailable  → 503
  ├─ CodeMethodNotAllowed → 405
  └─ otherwise        → 500
       └─ data is {}, error.message / error.code / error.errors[]
       └─ message is localized from Accept-Language, fallback en
```

`taskJSON` fields: `id`, `title`, `status`, `dueDate`, `overdue`, `createdAt`, `completedAt`. No owner.

`pathID` parses `mux.Vars(r)["id"]` as a UUID. Failure is `400 INVALID_IDENTIFIER`.

`decodeJSON` rejects unknown JSON fields and a second JSON value. Empty body on create is treated as an empty struct (missing title then fails in the domain).

Timestamps are UTC, truncated to milliseconds. Fractional seconds beyond three digits are rejected.

### POST /tasks — Create()

```
TaskHandler.Create(w, r)
  │
  ├─ [PRE-CHECK] decodeJSON → createBody{title, dueDate}     [NO USER/OWNER]
  │    └─ [NEEDS USER CONTEXT] Authorization is ignored
  ├─ [PRE-CHECK] parseOptionalDue(dueDate, required=false)
  │    ├─ omitted or null → nil due date
  │    └─ non-timestamp → 400 INVALID_DUE_DATE
  │
  └─ [SYNC] TaskUseCase.Create(r.Context(), CreateTaskInput)
       │
       ├─ now = Clock.Now() UTC, truncated to milliseconds
       ├─ id  = uuid.New() (runtime does not inject IDs)
       ├─ [BRANCHING] aggregate.NewTask
       │    ├─ trim title; empty → 400 INVALID_TITLE
       │    ├─ title longer than 100 Unicode code points → 400 INVALID_TITLE
       │    ├─ due date strictly before now → 400 INVALID_DUE_DATE
       │    └─ due date equal to now is allowed
       │    └─ status = todo; createdAt = now; completedAt unset
       │    └─ [NO USER/OWNER] NewTask(id, title, dueDate, now)
       │
       └─ [DB] TaskRepository.Save
            ├─ Postgres INSERT ... ON CONFLICT (id) DO UPDATE
            ├─ updated_at = time.Now().UTC() at save time
            └─ failure → 500 INTERNAL_ERROR
                 └─ no event is published
```

Success: `201`, task view. `overdue` is computed from `now` and is false for a due date equal to `now`.

### POST /tasks/{id}/start — Start()

```
TaskHandler.Start(w, r)
  │
  └─ mutate()
       ├─ [PRE-CHECK] pathID
       ├─ [PRE-CHECK] if ContentLength > 0, body must decode to {}
       │    └─ non-empty object → 400 MALFORMED_REQUEST
       │    └─ ContentLength <= 0 skips the body check
       │
       └─ [SYNC] TaskUseCase.Start(r.Context(), id)          [NEEDS USER CONTEXT]
            └─ mutate(id)
                 ├─ [DB] Get(id)                             [NO USER/OWNER]
                 │    └─ no row → 404 TASK_NOT_FOUND
                 ├─ [BRANCHING] Task.Start()
                 │    ├─ status done → 409 TASK_FINAL
                 │    ├─ status not todo → 409 INVALID_TRANSITION
                 │    └─ status todo → status = in_progress
                 └─ [DB] Save
                      └─ no event is published
```

Success: `200`, same task view. Start does not change `completedAt` or `dueDate`.

### POST /tasks/{id}/complete — Complete()

```
TaskHandler.Complete(w, r)
  │
  └─ mutate()  (same path-id and empty-body checks as Start)
       │
       └─ [SYNC] TaskUseCase.Complete(r.Context(), id)       [NEEDS USER CONTEXT]
            │
            └─ [DB] RunInTransaction
                 ├─ [DB] Get(id)                             [NO USER/OWNER]
                 │    └─ no row → 404 TASK_NOT_FOUND (transaction rolls back)
                 ├─ [BRANCHING] Task.Complete(now)
                 │    ├─ status done → 409 TASK_FINAL
                 │    ├─ status not in_progress → 409 INVALID_TRANSITION
                 │    └─ status in_progress
                 │         ├─ status = done
                 │         ├─ completedAt = now
                 │         └─ returns event.TaskCompleted{TaskID, CompletedAt}
                 │              └─ [EVENT PAYLOAD] no actor, no owner
                 ├─ [DB] Save (row still uncommitted on Postgres)
                 └─ [SYNC] CompletionPublisher.PublishTaskCompleted
                      ├─ failure → 503 EVENT_UNAVAILABLE, transaction rolls back
                      └─ success → transaction commits, then 200
```

`Start` and `Reschedule` do not use `RunInTransaction`. Only complete does.

Postgres stores the `pgx.Tx` in the context. `Get` and `Save` use that transaction. The publisher receives the same context and does not enlist in the SQL transaction.

### PATCH /tasks/{id}/due-date — Reschedule()

```
TaskHandler.Reschedule(w, r)
  │
  ├─ [PRE-CHECK] pathID                                    [NEEDS USER CONTEXT]
  ├─ [PRE-CHECK] decodeJSON → rescheduleBody{dueDate}
  ├─ [PRE-CHECK] parseOptionalDue(dueDate, required=true)
  │    ├─ field omitted → 400 VALIDATION_FAILED
  │    ├─ JSON null → nil due date (clears the due date)
  │    └─ bad timestamp → 400 INVALID_DUE_DATE
  │
  └─ [SYNC] TaskUseCase.Reschedule(r.Context(), RescheduleTaskInput)
       └─ mutate(id)
            ├─ [DB] Get(id)                                [NO USER/OWNER]
            ├─ [BRANCHING] Task.Reschedule(dueDate, now)
            │    ├─ status done → 409 TASK_FINAL
            │    ├─ non-nil due date strictly before now → 400 INVALID_DUE_DATE
            │    └─ nil clears dueDate; equal-to-now is allowed
            └─ [DB] Save
                 └─ no event is published
```

Success: `200`. `overdue` is recomputed from the new due date and the use-case clock. A done task cannot be rescheduled.

### GET /tasks/{id} — Get()

```
TaskHandler.Get(w, r)
  │
  ├─ [PRE-CHECK] pathID                                    [NEEDS USER CONTEXT]
  └─ [SYNC] TaskUseCase.Get(r.Context(), id)
       ├─ [DB] Get(id)                                     [NO USER/OWNER]
       │    └─ no row → 404 TASK_NOT_FOUND
       └─ overdue = dueDate set AND status != done AND now > dueDate
            └─ no Save, no event
```

Success: `200`. Any caller who knows the UUID receives the task. There is no list endpoint.

### Repository

```
TaskRepository
  ├─ Get(ctx, id uuid.UUID)                 [NO USER/OWNER]
  ├─ Save(ctx, task *aggregate.Task)        [NO USER/OWNER]
  └─ RunInTransaction(ctx, fn)

Postgres.Get
  └─ SELECT id, title, status, due_date, created_at, completed_at
       FROM tasks WHERE id = $1
       └─ updated_at is not loaded
       └─ RestoreTask(...) — no owner argument

Postgres.Save
  └─ INSERT (id, title, status, due_date, created_at, updated_at, completed_at)
     ON CONFLICT (id) DO UPDATE title, status, due_date, updated_at, completed_at
       └─ created_at is not updated on conflict

Memory
  └─ map[uuid.UUID]*aggregate.Task
       └─ RunInTransaction calls fn(ctx) with no rollback

Unavailable
  └─ initial adapter only; Start returns before listen if required infra is missing
```

`infrastructure/repository/taskrepository/model.TaskRow` includes `UpdatedAt` and is not used by `Postgres`.

### TaskCompleted publisher

```
Task.Complete(now)
  └─ domain/event.TaskCompleted{ TaskID, CompletedAt }     [EVENT PAYLOAD]
       │
       └─ [SYNC] RabbitPublisher.PublishTaskCompleted
            ├─ JSON taskCompletedMessage
            │    ├─ taskId      string (UUID)
            │    └─ completedAt string (UTC, millisecond layout)
            ├─ ContentType application/json
            ├─ DeliveryMode persistent
            ├─ Timestamp time.Now().UTC() (broker timestamp, not CompletedAt)
            ├─ mandatory=true, immediate=false
            └─ PublishWithContext(ctx, exchange, routingKey=queue, ...)

DeclareTaskCompleted (process startup, not per request)
  ├─ exchange declare: name from RABBITMQ_EXCHANGE, default taskdemo.events
  │    └─ type direct, durable, not auto-delete
  ├─ queue declare: name from RABBITMQ_QUEUE, default taskdemo.task_completed
  │    └─ durable, not auto-delete, not exclusive
  └─ bind queue to exchange with routing key equal to the queue name

Consumer
  └─ none in this repository
```

`RecordingPublisher` (optional infra) appends the domain event in memory and does not open AMQP. `UnavailablePublisher` returns `EVENT_UNAVAILABLE` and is not left in place on a serving process.

`[EVENT PAYLOAD]` impact of ownership: the domain struct and the JSON object both stop at `taskId` and `completedAt`. An owner or actor is not published. No in-repo consumer reads the body. An external consumer of `taskdemo.task_completed` sees those two fields only.

### Table `tasks`

One migration: `../../../../migrations/pgsql/20261027204911_create_tasks.up.sql`. Down drops the index, then the table. `cmd/migrate` runs `m.Up()` against `DATABASE_URL`.

| Column | Type | Null | Written by |
| --- | --- | --- | --- |
| id | UUID, primary key | no | use case (`uuid.New`) |
| title | TEXT | no | domain, trimmed |
| status | TEXT | no | `todo`, `in_progress`, `done` |
| due_date | TIMESTAMPTZ | yes | domain |
| created_at | TIMESTAMPTZ | no | domain clock at create |
| updated_at | TIMESTAMPTZ | no | repository `time.Now().UTC()` on every save |
| completed_at | TIMESTAMPTZ | yes | domain clock at complete |

Index: `tasks_status_INDEX` on `status`.

`[NO USER/OWNER]` there is no user table, no `owner_id`, and no foreign key. Lookup and upsert match `id` alone.

---

## Request

No task operation reads a header for identity. `Accept-Language` affects error text only.

| Parameter | Type | Source | Description |
| --- | --- | --- | --- |
| title | string | body, `POST /tasks` | Required after trim. 1–100 Unicode code points. `[NO USER/OWNER]` |
| dueDate | RFC3339 string or null | body, `POST /tasks` | Optional. Omitted and `null` both mean no due date. `[NO USER/OWNER]` |
| dueDate | RFC3339 string or null | body, `PATCH /tasks/{id}/due-date` | Required key. `null` clears the due date. `[NO USER/OWNER]` |
| id | UUID | path, every `/tasks/{id}` route | Server-generated. Not an owner key. `[NEEDS USER CONTEXT]` |
| body | empty or `{}` | body, start and complete | A non-empty object is `400` when `ContentLength > 0`. `[NO USER/OWNER]` |
| Authorization | — | header | Not read. Log masking would hide it if present. `[NEEDS USER CONTEXT]` |
| Accept-Language | string | header | Localizes `error.message` only. |

`CreateTaskInput` is `Title` and `DueDate`. `RescheduleTaskInput` is `ID` and `DueDate`. `Start`, `Complete`, and `Get` take only `context.Context` and `uuid.UUID`.

---

## Response

| Status | Condition |
| --- | --- |
| 201 | Create saved a new `todo` task. |
| 200 | Start, complete, reschedule, or get succeeded. Health also returns 200 with `data: {}`. |
| 400 | Malformed JSON, unknown field, bad UUID, empty or overlong title, bad or past due date, missing due date on reschedule. Codes: `MALFORMED_REQUEST`, `INVALID_IDENTIFIER`, `INVALID_TITLE`, `INVALID_DUE_DATE`, `VALIDATION_FAILED`. |
| 404 | `TASK_NOT_FOUND` for an unknown id. `ROUTE_NOT_FOUND` for an unknown path. |
| 405 | `METHOD_NOT_ALLOWED`. |
| 409 | `TASK_FINAL` when status is `done`. `INVALID_TRANSITION` when start is not from `todo` or complete is not from `in_progress`. |
| 503 | `EVENT_UNAVAILABLE` when complete's publish fails. The Postgres transaction is rolled back. |
| 500 | `INTERNAL_ERROR` for an unexpected repository or decode failure. |

Success response format:

```json
{
  "status": "success",
  "data": {
    "id": "uuid",
    "title": "string",
    "status": "todo | in_progress | done",
    "dueDate": "2006-01-02T15:04:05.000Z07:00 | null",
    "overdue": false,
    "createdAt": "2006-01-02T15:04:05.000Z07:00",
    "completedAt": "2006-01-02T15:04:05.000Z07:00 | null"
  }
}
```

`[NO USER/OWNER]` `data` has no owner field. Clients do not send `createdAt` or `completedAt`.

Error response format:

```json
{
  "status": "error",
  "data": {},
  "error": {
    "message": "localized text",
    "code": "ERROR_CODE",
    "errors": [{ "message": "localized text", "code": "ERROR_CODE" }]
  }
}
```

`[EVENT PAYLOAD]` published only on a committed complete:

```json
{
  "taskId": "uuid",
  "completedAt": "2006-01-02T15:04:05.000Z07:00"
}
```

---

## Dependencies

| Component | File |
| --- | --- |
| `Server.Router`, health, method/not-found handlers | `../../../../delivery/http/router.go` |
| `TaskHandler` | `../../../../delivery/http/task.go` |
| JSON envelope and localization | `../../../../delivery/http/envelope.go` |
| Timestamp parse/format | `../../../../delivery/http/time.go` |
| `TaskUseCase` | `../../../../application/usecase/task.go` |
| `CreateTaskInput`, `RescheduleTaskInput`, `TaskView` | `../../../../application/dto/task.go` |
| `CompletionPublisher` | `../../../../application/event/publisher.go` |
| `Task` aggregate | `../../../../domain/aggregate/task.go` |
| `TaskCompleted` | `../../../../domain/event/task_completed.go` |
| `Status` (`todo`, `in_progress`, `done`) | `../../../../domain/enum/status.go` |
| `TaskRepository` | `../../../../domain/repository/task.go` |
| `Clock`, `SystemClock` | `../../../../domain/service/clock.go` |
| Error codes and kinds | `../../../../pkg/errs/codes.go` |
| `Postgres` | `../../../../infrastructure/repository/taskrepository/postgres.go` |
| `Memory` | `../../../../infrastructure/repository/taskrepository/memory.go` |
| `Unavailable` repository | `../../../../infrastructure/repository/taskrepository/unavailable.go` |
| Unused `TaskRow` | `../../../../infrastructure/repository/taskrepository/model/task.go` |
| `RabbitPublisher`, `DeclareTaskCompleted` | `../../../../infrastructure/service/rabbit.go` |
| `RecordingPublisher` | `../../../../infrastructure/service/recording.go` |
| `UnavailablePublisher` | `../../../../infrastructure/service/unavailable.go` |
| Env config, exchange and queue names | `../../../../infrastructure/config/config.go` |
| Process composition | `../../../../infrastructure/runtime/runtime.go` |
| Log manager; masks `Authorization` | `../../../../cmd/app/main.go` |
| Request logging middleware | `github.com/SALT-Indonesia/salt-pkg/logmanager/integrations/lmgorilla` |
| Migration up/down | `../../../../migrations/pgsql/20261027204911_create_tasks.up.sql`, `migrations/pgsql/20261027204911_create_tasks.down.sql` |
| Migration command | `../../../../cmd/migrate/main.go` |
| RabbitMQ consumer | none |

---

## Important Notes

1. `[NO USER/OWNER]` The current slice has no account, role, token, or owner. `../../../../AGENTS.md` leaves auth and ownership out of scope. The aggregate, DTOs, HTTP body, SQL row, repository key, and domain event all identify a task by UUID only.
2. `[NEEDS USER CONTEXT]` These are the boundaries that today carry no caller identity: `Server.Router` (single middleware, no route group), each `TaskHandler` method (`r.Context()` only), `TaskUseCase.Create` / `Start` / `Complete` / `Reschedule` / `Get`, and `TaskRepository.Get` / `Save`. `GET /health` is the route that stays unauthenticated.
3. `[NEEDS USER CONTEXT]` `Create` is the write that would have to attach an owner, because `NewTask` and `Save` persist the row with no owner column. `Get`, `Start`, `Complete`, and `Reschedule` load and update by `id` alone, so they cannot tell one caller from another.
4. `[EVENT PAYLOAD]` Complete is the only publisher. The body is `taskId` and `completedAt`. Create, start, reschedule, and get emit nothing. There is no consumer in this module, so a payload change affects `domain/event.TaskCompleted`, `taskCompletedMessage`, and any external subscriber of queue `taskdemo.task_completed` on direct exchange `taskdemo.events`. The routing key is the queue name.
5. Publish is synchronous inside `RunInTransaction`, after `Save` and before `Commit`. A publish error rolls the Postgres transaction back and the client receives `503`. The AMQP publish is not part of the SQL transaction: a successful publish followed by a failed commit can leave a message whose row was rolled back. `Memory.RunInTransaction` does not roll back at all.
6. `updated_at` exists only in Postgres. The domain aggregate and the HTTP view do not expose it. `overdue` is not stored; it is derived on each response from the use-case clock.
7. Start and complete inspect the body only when `ContentLength > 0`. A chunked request with `ContentLength == -1` does not go through the empty-object check.
8. Status lifecycle enforced in the aggregate: create → `todo`; start only from `todo`; complete only from `in_progress`; `done` rejects start, complete, and reschedule. Due date equal to now is valid. Overdue is `now` strictly after `dueDate` while status is not `done`.
