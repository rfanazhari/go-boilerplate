# Tactical Strategy: User Management, Authentication, and Task Ownership

## 1. Implementation Strategy

**Breaking change, additive schema, wrap existing task lifecycle.** Do not restyle `Task.Start` / `Complete` / `Reschedule` or the complete-and-publish transaction. Add identity around those paths.

Order of work matches independently testable slices: schema → domain → account use cases → login/tokens → HTTP RBAC → task ownership (including a regression lock on Rabbit JSON).

Composition stays in `infrastructure/runtime`. Memory repositories remain the `INFRA_OPTIONAL` path. Production compose still waits for healthy `db` and `rabbitmq`, then migrate, then start (seed admin on start).

## 2. Code Reuse Map

| Component | Location | Used for |
|-----------|----------|----------|
| `Task.Start`, `Complete`, `Reschedule`, `Overdue`, title/due rules | `domain/aggregate/task.go` | Unchanged lifecycle (`BR-TASKDEMO-001`–`009`) |
| `TaskUseCase` complete + `RunInTransaction` + `CompletionPublisher` | `application/usecase/task.go`, `application/event/publisher.go` | Save-then-publish; ownership wraps `Get` only |
| `event.TaskCompleted` + `taskCompletedMessage` | `domain/event/task_completed.go`, `infrastructure/service/rabbit.go` | Payload stays `taskId`, `completedAt` |
| JSON envelope, `decodeJSON` unknown-field reject, timestamp parse | `delivery/http/envelope.go`, `task.go`, `time.go` | All new handlers |
| `GET /health` | `delivery/http/router.go` | Public, same mux |
| `lmgorilla.Middleware` | `Server.Router` | Keep as first `router.Use`; already masks `Authorization` |
| `TaskRepository` Get/Save/RunInTransaction | `domain/repository/task.go` | Id lookup remains; no owner filter in SQL |
| Postgres upsert `ON CONFLICT (id)` | `infrastructure/repository/taskrepository/postgres.go` | Add `owner_id` on INSERT only |
| Memory / Unavailable task adapters | `taskrepository/memory.go`, `unavailable.go` | Same owner field as Postgres |
| Clock | `domain/service/clock.go` | Unchanged |
| golang-migrate + `cmd/migrate` | `migrations/pgsql`, `cmd/migrate` | New up/down files |
| Table-driven `_test` packages + Mockery | `application/usecase/task_test.go`, `.mockery.yml` | Pattern for new use-case tests |
| `pkg/errs.Error` + i18n TOML | `pkg/errs/codes.go`, `translations/errors.*.toml` | New codes |

## 3. Net-New Code Inventory

| Item | Layer/Component | Type | Notes |
|------|-----------------|------|-------|
| `accounts` + `credential_tokens` + `tasks.owner_id` | migrate | schema | Partial unique email; backfill owner |
| `Account` aggregate | domain | aggregate | Role, status, soft-delete, profile |
| Role / account-status enums | domain | enum | |
| `AccountRepository`, `TokenRepository` | domain + infrastructure | ports + Postgres/Memory | Plain structs, raw SQL `$1` |
| Password hasher (bcrypt) | infrastructure (port in domain/application) | service | Cost documented in design |
| Login / account / profile use cases | application | use case | |
| Opaque token issue + SHA-256 store | application + infrastructure | auth | D1 |
| Auth + `RequireRole` middleware | delivery | middleware | D3 |
| Account, login, profile handlers | delivery | HTTP | Envelope |
| Context actor | delivery | context key | Handlers pass id/role into use cases |
| Seed admin | infrastructure/runtime + config | bootstrap | Env-only |
| OpenAPI security + new paths | `docs/api` | contract | Changelog bump |

## 4. Branching / PR Strategy

Prefer **one feature branch**, commits aligned to task-plan IDs (T1…T6) so each commit is reviewable and `go test` green. Merge order is the task order: a half-migrated `owner_id` must not ship without seed + backfill.

Do not split HTTP RBAC and task ownership across releases: after T1–T4, `/tasks` is still public until T5–T6. If a PR must ship early, stop **before** requiring auth on `/tasks` or ship T5+T6 together.

## 5. Testing Strategy

Commands from `AGENTS.md`: `go test ./path -v -run TestName` after each edit; `go test ./...`; `make test`; `make test.arch`; `make test.smoke`; `make lint`. Domain/application tests in `_test` packages, table-driven named structs, testify, controllable clocks. Mocks via Mockery only.

### Unit Tests

- Account aggregate: role, transitions, soft-delete, profile locks role/email/status.
- Task aggregate: `NewTask` requires owner; lifecycle still BR-001–009.
- Account use case: add, deactivate, reactivate, last-admin, self-target, email reuse after soft-delete.
- Login: bad password, inactive, soft-deleted, success issues token; deactivate deletes tokens.
- Task use case: create sets owner; user get/mutate foreign → `TASK_NOT_FOUND`; admin get foreign OK; admin mutate foreign → `TASK_NOT_FOUND`.
- Middleware: missing Bearer 401; user on admin route 403; admin on `POST /tasks` 403.
- Rabbit publisher: JSON marshal equals `taskId` + `completedAt` only.

### Integration Tests

- Happy path: seed/login as user → create → start → complete → get; AMQP payload unchanged (`go test -tags=integration ./test/infrastructure/...` when DB/broker up).
- Error cases: 401 on `/tasks` without token; 403 admin create; 404 not-owner; 409 email taken; 409 last admin; login after deactivate 401/INVALID_CREDENTIALS; reused email after soft-delete succeeds.
- Migration: apply up on a DB that already has a `tasks` row; `owner_id` equals seeded admin.

### Regression Tests

- Existing domain lifecycle tables still pass.
- Complete still 503 when publish fails (transaction rollback).
- `GET /health` 200 without Authorization.
- `make test.arch` layer import rules.
- Decode still rejects unknown JSON fields on create (client cannot send `ownerId`).

## 6. Rollout & Rollback

- **Rollout:** migrate up (accounts, tokens, owner backfill) → deploy binary that seeds admin if missing and requires Bearer on `/tasks`. Clients must obtain a token. Breaking HTTP; Rabbit consumers unchanged.
- **Rollback:** revert binary first only if old binary can run against new schema (`owner_id` unused) — current `Save`/`Get` SQL will **fail** once the column exists if the old binary does not select/insert it. Therefore migrate and binary **must** deploy together. True rollback: revert binary and migrate down (drops owner_id, tokens, accounts). Tasks survive without owners; accounts are lost.
- **Monitor:** 401/403 rates after cutover; login failures; seed-admin start errors.
