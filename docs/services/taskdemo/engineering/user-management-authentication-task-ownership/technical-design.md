# Technical Design: User Management, Authentication, and Task Ownership

## 1. Overview

taskdemo today exposes unauthenticated task HTTP operations. This change adds accounts (admin | user), opaque Bearer login with revocation, and task ownership, without rewriting lifecycle rules `BR-TASKDEMO-001`–`009`. `GET /health` stays public. `TaskCompleted` JSON stays `taskId` + `completedAt` (decision-log D7 in capability Q15; not a token/RBAC decision).

Stack (from `AGENTS.md` and code): Go, gorilla/mux, Postgres (`golang-migrate` under `migrations/pgsql/`), RabbitMQ publisher, four inward layers (`domain/`, `application/`, `delivery/`, `infrastructure/`).

### Goals

- Persist accounts with role and lifecycle (active, non-active, soft-deleted).
- Issue and revoke opaque Bearer tokens; deny login when the account is not active.
- Let admins add, deactivate, reactivate, and soft-delete accounts (not self, not last admin).
- Let users update their own name and password only.
- Stamp `ownerId` on create (user role only); gate get/mutate by owner; admin GET-by-id any task.
- Backfill existing task rows to the seeded admin, then require `owner_id`.

### Non-Goals

- Self-registration, extra roles, hard delete, ownership transfer, task/account lists, password reset, MFA, SSO, JWT, session cookies.
- Changing `BR-TASKDEMO-001`–`009` except create requiring an owner.
- Changing RabbitMQ routing or adding owner/actor to `TaskCompleted`.

## 2. API / Interface Contract Changes

Envelope unchanged: `status` + `data`; errors include `error.message`, `error.code`, `error.errors`.

### Unauthenticated (stay or add)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/health` | Unchanged. `security: []`. |
| POST | `/login` | New. Body `{ "email", "password" }`. Success `200` `{ "token", "tokenType": "Bearer" }`. |

### Authenticated — user role (`RequireRole user`)

| Method | Path | Notes |
|--------|------|-------|
| PATCH | `/profile` | Body `{ "name"? , "password"? }` at least one required. Identity is the caller; no account id in the path (`BR-TASKDEMO-031`). |
| POST | `/tasks` | Body still `{ "title", "dueDate"? }`. Clients must not send `ownerId`. Admin is `403`. |

### Authenticated — either role (`RequireAuth`)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/tasks/{id}` | User: own only. Admin: any id. Not-owner user → `404 TASK_NOT_FOUND`. |
| POST | `/tasks/{id}/start` | Owner only (including if caller is admin). |
| POST | `/tasks/{id}/complete` | Owner only; publish path unchanged. |
| PATCH | `/tasks/{id}/due-date` | Owner only. |

### Authenticated — admin role (`RequireRole admin`)

| Method | Path | Notes |
|--------|------|-------|
| POST | `/accounts` | Body `{ "email", "name", "password", "role": "admin"\|"user" }`. Admin-set password. `201`. |
| PATCH | `/accounts/{id}/status` | Body `{ "status": "active"\|"non_active" }`. Active → non-active revokes tokens. Non-active → active allowed. |
| DELETE | `/accounts/{id}` | Soft delete; email becomes reusable; tokens revoked. |

Task success `data` adds additive `ownerId` (UUID). OpenAPI: Bearer security on all routes except `/health` and `/login`.

### Event contract

No change. Publisher JSON remains:

```json
{ "taskId": "uuid", "completedAt": "2006-01-02T15:04:05.000Z07:00" }
```

## 3. Data Model & Schema Changes

New migration after `20261027204911_create_tasks` (golang-migrate, `cmd/migrate`).

### `accounts`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Server-generated. |
| email | TEXT NOT NULL | Stored trimmed, compared case-insensitively. |
| name | TEXT NOT NULL | |
| password_hash | TEXT NOT NULL | bcrypt (D2). |
| role | TEXT NOT NULL | `admin` \| `user` (CHECK). |
| status | TEXT NOT NULL | `active` \| `non_active` (CHECK). Soft-deleted rows keep last status. |
| deleted_at | TIMESTAMPTZ NULL | NULL = not soft-deleted. |
| created_at / updated_at | TIMESTAMPTZ NOT NULL | |

Unique email among live accounts (D4):

```sql
CREATE UNIQUE INDEX accounts_email_live_uidx
  ON accounts (LOWER(email))
  WHERE deleted_at IS NULL;
```

### `credential_tokens`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| account_id | UUID NOT NULL FK accounts(id) | |
| token_hash | TEXT NOT NULL UNIQUE | SHA-256 of the raw token (hex). Never store plaintext. |
| created_at | TIMESTAMPTZ NOT NULL | |

No expiry column in v1: revocation is explicit (D1, D5). Rows deleted (or marked revoked) on logout-equivalent paths: deactivate and soft-delete of that account.

### `tasks` (modified)

- Add `owner_id UUID NOT NULL REFERENCES accounts(id)`.
- Backfill existing rows to the seeded admin, then set NOT NULL (D6).
- Include `owner_id` in INSERT; do not update `owner_id` on conflict (ownership is immutable).

### Seed

Idempotent first admin: env `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `SEED_ADMIN_NAME` (defaults documented in `cmd/app/.env.example`). Hash at process start (or a one-shot seed command used by compose after migrate). Insert only when no live admin exists. Do not put a plaintext password in SQL.

**T1 bootstrap (migration + runtime):** The accounts migration inserts a fixed bootstrap admin row (`id` `e0000000-0000-4000-8000-000000000001`, constant `seed.BootstrapAdminID`) with default email/name and a bcrypt hash of the documented default password so legacy `tasks` rows can be backfilled before the app starts. On process start, `seed.SeedAdmin` updates that row from env (email, name, password hash). If the bootstrap row is absent and no live admin exists, it inserts a new admin from env instead.

### Rollback

Down migration: drop `tasks.owner_id`, drop `credential_tokens`, drop `accounts`. Ordered: tokens → owner_id → accounts. Data loss: all accounts and tokens; tasks remain without owner (pre-change shape).

## 4. Component Breakdown

| Layer/Component | Status | Justification |
|-----------------|--------|---------------|
| `domain/aggregate.Task` | Modified | Add `ownerID`; `NewTask` / `RestoreTask` require it. Lifecycle methods reused. |
| `domain/event.TaskCompleted` | Reused as-is | Product: no owner/actor on the event. |
| `domain/aggregate.Account` (new) | New | Role, status, soft-delete, password hash holder, profile fields. |
| `domain/enum` role + account status | New | Parallel to `enum.Status`. |
| `domain/repository.TaskRepository` | Reused as-is | Still Get/Save by id; ownership evaluated after load. |
| `domain/repository.AccountRepository` | New | Save, Get by id/email, list/count live admins for last-admin guard. |
| `domain/repository.TokenRepository` | New | Save hash, Get by hash, DeleteByAccountID. |
| `application/usecase.TaskUseCase` | Modified | Create takes actor id as owner; Get/Start/Complete/Reschedule take actor; 404 if user (or mutating admin) is not owner; admin GET skips owner match. No role matrix here (D3). |
| `application/usecase` account + profile + login | New | Admin account commands; profile self-update; verify password + issue token; revoke on status change. |
| `application/dto` | Modified + New | `TaskView.OwnerID`; `CreateTaskInput` still has no owner (handler injects). New account/login DTOs. |
| `delivery/http.Server.Router` | Modified | Public: health, login. Subrouter with auth middleware; nested admin and user groups (D3). |
| `delivery/http.TaskHandler` | Modified | Read identity from context; pass actor into use case; `taskJSON` + `ownerId`. |
| `delivery/http` auth/account/profile handlers | New | Translate envelope; do not encode RBAC beyond calling protected routes. |
| `delivery/http` auth + RBAC middleware | New | Parse Bearer, load token+account, reject inactive/soft-deleted as 401; role gate 403. |
| `delivery/http.envelope` | Modified | Map `KindUnauthenticated` → 401, `KindForbidden` → 403. |
| `pkg/errs` | Modified | New kinds and codes: `UNAUTHENTICATED`, `FORBIDDEN`, `INVALID_CREDENTIALS`, `ACCOUNT_NOT_FOUND`, `EMAIL_TAKEN`, `LAST_ADMIN`, `INVALID_ROLE`, `INVALID_STATUS`, `INVALID_PASSWORD`. |
| `infrastructure/repository/taskrepository` | Modified | Persist `owner_id`; RestoreTask with owner. Memory + Unavailable adapters. |
| `infrastructure/repository/accountrepository` | New | Postgres + Memory for `INFRA_OPTIONAL`. |
| `infrastructure/repository/tokenrepository` | New | Postgres + Memory. |
| `infrastructure/service/rabbit.go` | Reused as-is | JSON fields unchanged; tests must lock the payload. |
| `infrastructure/runtime` | Modified | Wire new repos, seed admin, inject middleware. |
| `infrastructure/config` | Modified | Seed admin env vars. |
| `docs/api/openapi.yaml` | Modified | Security scheme, new paths, `ownerId`, 401/403. |
| `test/smoke` | Modified | Login (or seed + login) then Bearer on `/tasks`. |

### Detail per component

**Identity on context (delivery only).** Middleware stores `Actor{ID, Role}` on `context.Context`. Handlers copy `ID` (and for GET, `Role`) into use-case arguments. Application must not import `delivery`. Role checks for routes live only in middleware (D3). Resource ownership is not RBAC; it stays in `TaskUseCase` after `Get`.

**Account aggregate.** `NewAccount` validates role (`admin`|`user`), email (non-empty, trimmed), name, and accepts a precomputed password hash. `Deactivate` / `Activate` / `SoftDelete` enforce transitions. `UpdateProfile(name, passwordHash)` changes only those fields. Soft-deleted accounts reject login at the auth use case even if a token row lingered.

**Password.** Hash and compare in infrastructure (or a tiny `domain/service` hasher port implemented with bcrypt) so domain tests can inject a fake hasher.

**Login.** Lookup live account by email → bcrypt compare → if not active or deleted, same `INVALID_CREDENTIALS` as bad password (no account enumeration). Insert token hash; return raw token once.

**Revocation (D5).** `Deactivate` and `SoftDelete` run in a transaction: update account, `TokenRepository.DeleteByAccountID`. Subsequent Bearer lookups fail → 401.

**Task create.** Middleware ensures role is user. Use case: `NewTask(..., ownerID)`. Owner is never taken from JSON (`DisallowUnknownFields` already rejects `ownerId` on the request body if not in the struct — keep it off the body struct).

**Task get/mutate.** Load by id. If missing → `TASK_NOT_FOUND`. If mutate and `task.OwnerID() != actor.ID` → `TASK_NOT_FOUND`. If get and actor role is admin → return; if user and not owner → `TASK_NOT_FOUND`.

## 5. Flow Sequence

### Login

1. `POST /login` (no auth middleware).
2. Login use case loads live account by email, verifies password, checks active.
3. Generate 32-byte random token; store SHA-256 hex; return raw token.

### Authenticated request

1. Logging middleware (existing `lmgorilla`).
2. Auth middleware: missing/malformed Bearer → 401; unknown/revoked hash → 401; account soft-deleted or non-active → 401 (and delete leftover tokens).
3. Role middleware (when registered): mismatch → 403.
4. Handler decodes body (existing rules) and calls the use case with actor.

### Admin deactivate

1. Admin route group.
2. Use case: reject self; reject if target is the last live admin; set non-active; delete tokens.

### Create task

1. User route group (admin never reaches the handler).
2. `NewTask` with caller id as owner; `Save`; `201` with `ownerId`.

### Complete (unchanged publish)

1. Owner check inside use case.
2. Existing `RunInTransaction`: Complete → Save → `PublishTaskCompleted` (`taskId`, `completedAt` only).

## 6. Request / Response (or Input / Output)

Create/get task success `data` (additive field):

```json
{
  "id": "uuid",
  "title": "string",
  "status": "todo | in_progress | done",
  "dueDate": "2006-01-02T15:04:05.000Z07:00 | null",
  "overdue": false,
  "createdAt": "2006-01-02T15:04:05.000Z07:00",
  "completedAt": "2006-01-02T15:04:05.000Z07:00 | null",
  "ownerId": "uuid"
}
```

Login success `data`: `{ "token": "<opaque>", "tokenType": "Bearer" }`.

## 7. Error Handling

| HTTP | Kind | Code | When |
|------|------|------|------|
| 401 | Unauthenticated | `UNAUTHENTICATED` | No/invalid/revoked token; inactive/deleted presenter. |
| 401 | Unauthenticated | `INVALID_CREDENTIALS` | Login email/password/inactive (same code). |
| 403 | Forbidden | `FORBIDDEN` | User hits admin routes; admin `POST /tasks`; user hits another profile if ever addressed by id. |
| 404 | NotFound | `TASK_NOT_FOUND` | Unknown id or not-owner (get/mutate). |
| 404 | NotFound | `ACCOUNT_NOT_FOUND` | Admin target missing / already soft-deleted. |
| 409 | Conflict | `EMAIL_TAKEN` | Live email unique violation. |
| 409 | Conflict | `LAST_ADMIN` | Deactivate/delete self or last live admin. |
| 400 | InvalidInput | existing + `INVALID_ROLE`, `INVALID_STATUS`, `INVALID_PASSWORD` | Validation. |

Propagation: domain/use case return `*errs.Error` → `writeError`. Middleware constructs the same type so localization TOML keys stay consistent (`translations/errors.en.toml`, `errors.id.toml`).

## 8. Constraints Checklist

- [x] Breaking for existing HTTP clients (auth required on `/tasks`).
- [x] `BR-TASKDEMO-001`–`009` reused; create gains owner only.
- [x] RBAC in middleware, not in task/account use-case role matrices (D3). Ownership checks remain in the task use case.
- [x] Opaque hashed Bearer, not JWT (D1). Capability lists JWT as out of scope.
- [x] bcrypt password hashing (D2).
- [x] Soft-delete + reusable email via partial unique index (D4).
- [x] Tokens revoked on deactivate and soft-delete (D5).
- [x] Legacy tasks owned by seeded admin (D6).
- [x] `TaskCompleted` payload unchanged.
- [x] No task list; admin “see all” is GET-by-id.
- [x] `GET /health` unauthenticated.
- [x] Inward dependencies unchanged (`test.arch`).

## 9. Risks & Open Questions

| Risk | Mitigation |
|------|------------|
| Smoke and example HTTP tests still POST `/tasks` without Bearer | T6 updates `test/smoke`. |
| Seed password in env is a secret | Document in `.env.example` only; never commit real values. |
| Last-admin race | Count live admins in the same transaction as deactivate/delete. |
| Memory adapter has no real revoke durability | Mirror DeleteByAccountID; sufficient for `INFRA_OPTIONAL`. |
| External AMQP consumers | Unchanged payload; still document in OpenAPI/changelog that HTTP is breaking. |

No blocking product questions remain (capability 1–16 closed).

## 10. Revision History

| Version | Date | Change | Reason |
|---------|------|--------|--------|
| v1 | 2026-10-27 | Initial design | Mode B from delta report + capability + current-state flow |
