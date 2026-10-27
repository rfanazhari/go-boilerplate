# Delta Report: User Management, Authentication, and Task Ownership
Generated: 2026-10-27
Source: flow-task-current-state.md → user-management-authentication-task-ownership.md

## 1. Change Type
<!-- Mark one -->
- [ ] Additive only (no breaking change)
- [x] Breaking change (affects existing behavior)
- [ ] Mixed (partly additive, partly breaking)

Justification: Account, role, authentication, and profile work are net-new. Existing task HTTP operations already exist and stay in place, but they become identity-bound: callers who today succeed with only a UUID must authenticate, create must stamp an owner (`BR-TASKDEMO-040`), and start / complete / reschedule / get must reject access to tasks the caller does not own (`BR-TASKDEMO-042`). That is a breaking change for any current HTTP client and for any stored task row that has no owner. Task lifecycle rules `BR-TASKDEMO-001`–`009` are not rewritten except that create now requires an owner. `GET /health` remains unauthenticated.

## 2. What Changed

### BR mapping (`BR-XXX` → NEW / CHANGED / REUSED)

| Rule | Classification | Rationale |
|---|---|---|
| `BR-TASKDEMO-001` Title 1–100 after trim | **REUSED** | Unchanged; still enforced in `aggregate.NewTask`. |
| `BR-TASKDEMO-002` New task starts as todo | **REUSED** | Restated by `BR-TASKDEMO-041` for the owned-create path; current `NewTask` already sets `todo`. |
| `BR-TASKDEMO-003` Due date not in the past | **REUSED** | Unchanged on create and reschedule. |
| `BR-TASKDEMO-004` Start only from todo | **REUSED** | Unchanged; ownership is an extra gate before mutate. |
| `BR-TASKDEMO-005` Complete only from in_progress | **REUSED** | Unchanged. |
| `BR-TASKDEMO-006` Done cannot be modified | **REUSED** | Unchanged. |
| `BR-TASKDEMO-007` Overdue rule | **REUSED** | Unchanged; still derived, not stored. |
| `BR-TASKDEMO-008` System timestamps | **REUSED** | Clients still must not send `createdAt` / `completedAt`. |
| `BR-TASKDEMO-009` Publish `TaskCompleted` | **REUSED** | Complete still saves then publishes inside the transaction. Capability does not require a new event or a payload change. |
| `BR-TASKDEMO-010` Acting requires authenticated identity | **NEW** | Flow has no token read, no user on context; only logging middleware. Applies to create-task, profile, and admin account work. Health stays open. |
| `BR-TASKDEMO-011` User role denied admin account work | **NEW** | No roles exist. |
| `BR-TASKDEMO-020` Role is exactly admin or user | **NEW** | No account type. |
| `BR-TASKDEMO-021` Only admin adds accounts | **NEW** | No account table or add-account flow. |
| `BR-TASKDEMO-022` Only admin active → non-active | **NEW** | No account status. |
| `BR-TASKDEMO-023` Only admin soft-deletes accounts | **NEW** | No soft-delete. |
| `BR-TASKDEMO-030` User updates own profile | **NEW** | No profile. Fields: name and password only. |
| `BR-TASKDEMO-031` User cannot update another profile | **NEW** | No profile. |
| `BR-TASKDEMO-040` Create assigns owner = authenticated user | **CHANGED** | `POST /tasks` exists (`TaskHandler.Create` → `TaskUseCase.Create` → `NewTask` → `Save`) but identity is ignored and the row has no owner. Create must take a caller identity and persist it. |
| `BR-TASKDEMO-041` User creates task in todo (owned) | **REUSED** | Todo-on-create is already `BR-TASKDEMO-002`. Ownership is covered by `BR-TASKDEMO-040`. Do not change lifecycle meaning. |
| `BR-TASKDEMO-042` User may view/change only owned tasks | **CHANGED** | `GET /tasks/{id}`, start, complete, and reschedule exist and load by UUID only. After this rule, a non-owner user is rejected as not found (HTTP 404). Admin may GET any id; admin mutate of others' tasks is also 404. |

### New
- Account aggregate (role admin|user; lifecycle active, non-active, soft-deleted).
- Authenticated identity on requests other than `GET /health`.
- Admin-only: add account (role chosen by admin; password set by admin), set active → non-active, reactivate non-active → active (open question 7), soft-delete (email reusable after soft-delete, open question 3).
- First admin via seed (open question 1).
- User self-profile update (name and password only; role, status, email locked).
- Login denied and issued tokens revoked when the account is non-active or soft-deleted (open question 2).
- Owner on new tasks; `POST /tasks` is user-role only (admin create is 403).
- Authz on existing task read/mutate for the user role.
- Seed / bootstrap of the first admin; no self-registration.

### Modified
- All `/tasks` routes: currently public; must require an authenticated identity (`BR-TASKDEMO-010` for create; `BR-TASKDEMO-042` for get/start/complete/reschedule).
- `CreateTaskInput` / `NewTask` / `tasks` row / `taskJSON`: must carry owner (column and likely response field).
- `TaskUseCase.Start` / `Complete` / `Reschedule` / `Get` and `TaskRepository.Get`: must evaluate owner (and, for admin read, “see all”) instead of id-only access.
- HTTP success `data` for tasks: additive `owner` (or equivalent) would change the documented OpenAPI task view; clients that reject unknown fields would break. Current HTTP decoder already rejects unknown **request** fields; responses adding a field are additive for typical clients.
- Router: today one mux with a single logging `router.Use`; health and tasks cannot be split without changing registration. Auth middleware or a protected route group is required.
- Error surface: 401 unauthenticated; 403 forbidden role (including admin create-task); 404 not-owner and unknown task id.

### Removed
- None of the current task routes, envelope shape, or lifecycle transitions are withdrawn.
- Implicit product behavior removed: anonymous task create/read/mutate; UUID-as-capability-token.

## 3. Impact Analysis
| Layer | Impact | Notes |
|-------|--------|-------|
| Transport/Handler | Modified + New | Existing `TaskHandler` methods need caller identity (`[NEEDS USER CONTEXT]`). New handlers for login (opaque Bearer), admin account commands, and profile update. `GET /health` and login stay unauthenticated. Envelope reused; 401 / 403 added; not-owner uses existing 404. |
| Use Case | Modified + New | `TaskUseCase.Create` must set owner; other task methods must authorize. New use cases for accounts, profile, and credential verification. Task lifecycle orchestration (save, complete + publish) reused. |
| Domain / Service | Modified + New | `Task` aggregate and `NewTask` / `RestoreTask` gain owner. Status machine, overdue, timestamps, `TaskCompleted` type reused as-is unless a later decision adds owner/actor to the event (not required by the capability). New account/role/status model. Clock reused. |
| Repository | Modified + New | `tasks` needs `owner_id` (FK once accounts exist) plus backfill. New account persistence (including soft-delete and unique email rules). Memory adapters used in `INFRA_OPTIONAL` must mirror both. |
| Proto/Contract | Modified + New | OpenAPI: Bearer security on `/tasks` except health and login; task `data` gains `ownerId`. AMQP `TaskCompleted` stays `taskId` + `completedAt`. |

## 4. Reuse Map
| Component | Status | Notes |
|-----------|--------|-------|
| `Task` lifecycle (`Start`, `Complete`, `Reschedule`, overdue, title/due rules) | Reused as-is | Do not restyle `BR-TASKDEMO-001`–`009`. |
| `TaskUseCase` complete + `RunInTransaction` + `CompletionPublisher` | Reused as-is | Ownership check wraps Get; publish path unchanged. |
| `domain/event.TaskCompleted` and Rabbit JSON | Reused as-is | Capability silent on payload; keep `taskId` + `completedAt` unless product later requires owner. |
| JSON envelope, localization, timestamp parse | Reused as-is | |
| `GET /health` | Reused as-is | Still unauthenticated. |
| `Server.Router` single-mux + log middleware only | Needs wrapping | Add auth middleware or protected group; keep health outside. |
| `TaskHandler.Create` / `Start` / `Complete` / `Reschedule` / `Get` | Needs wrapping | Pass authenticated identity; keep decode/pathID/empty-body rules. |
| `CreateTaskInput`, `TaskView`, `NewTask`, `RestoreTask` | Replace (extend) | Add owner; request body still must not accept owner from the client. |
| `TaskRepository.Get(ctx, id)` | Needs wrapping | Either authorize in use case after Get, or add owner-aware lookup. Id-only Get remains useful for admin “see all”. |
| Postgres `tasks` table / `Postgres.Save` upsert | Needs wrapping | New column; existing `ON CONFLICT (id)` unchanged in spirit. |
| Memory / Unavailable task adapters | Needs wrapping | Same owner field as Postgres. |
| `lmgorilla` request log (already masks Authorization) | Reused as-is | |
| Account, credential, session/token, profile, seed admin | Replace | Nothing to reuse; types do not exist. |

## 5. Net-New Inventory
- [ ] Opaque Bearer authenticator, login, hashed token store, revoke on deactivate/soft-delete — Layer: Transport + infrastructure — Reason: `BR-TASKDEMO-010` + open questions 2 and 10.
- [ ] Account model (role, active / non-active / soft-delete, email, name, password) — Layer: Domain — Reason: `BR-TASKDEMO-020`–`023`.
- [ ] Admin add-account (admin-chosen role, admin-set password) — Layer: Use case + Transport — Reason: `BR-TASKDEMO-021` + open questions 8–9.
- [ ] Admin deactivate / reactivate / soft-delete, including “cannot target self or last admin” — Layer: Use case — Reason: `BR-TASKDEMO-022`–`023` + open questions 4 and 7.
- [ ] User profile update (name, password) — Layer: Use case + Transport — Reason: `BR-TASKDEMO-030`–`031` + open question 5.
- [ ] Login + token revocation on deactivate/soft-delete — Layer: Use case + infrastructure — Reason: open question 2; no login flow in current task doc.
- [ ] Seed of first admin — Layer: Infrastructure / migrate — Reason: open question 1.
- [ ] Task `owner` on aggregate, DTO, HTTP view, SQL, memory — Layer: Domain through repository — Reason: `BR-TASKDEMO-040`.
- [ ] Owner authorization: user get/mutate own only; admin GET-by-id any; admin cannot create or mutate others — Layer: Use case — Reason: `BR-TASKDEMO-040`–`042` + open questions 6 and 11.
- [ ] Data migration: `owner_id` NOT NULL, backfill existing rows to seeded admin — Layer: Repository / migrate — Reason: open question 13.

## 6. Open Questions
All items below are decided (capability open questions 1–16). None remain blocking.

- [x] First admin via seed.
- [x] Non-active / soft-deleted: login denied, tokens revoked.
- [x] Email reusable after soft-delete.
- [x] Admin cannot deactivate/delete self or last admin.
- [x] Profile fields: name and password only.
- [x] Admin may GET any task by id; no list endpoint.
- [x] Non-active may return to active.
- [x] New account role: admin-chosen.
- [x] Credentials: admin-set password.
- [x] Auth: opaque Bearer token, hashed store, login endpoint; health and login unauthenticated.
- [x] HTTP: 401 no identity; 403 forbidden role (including admin `POST /tasks`); 404 not-owner and unknown id.
- [x] Admin cannot create tasks; cannot start/complete/reschedule others' tasks.
- [x] Task JSON adds `ownerId`; clients do not send owner.
- [x] `TaskCompleted` payload unchanged (`taskId`, `completedAt`).
- [x] Legacy tasks: set `owner_id` to the seeded admin, then NOT NULL.

## 7. Recommendation

Safe to pass to engineering-tripack-generator. Keep `BR-TASKDEMO-001`–`009`, the envelope, health, and complete-and-publish; wrap task use cases with identity and owner. Split: (1) accounts + seed + opaque Bearer login/revoke, (2) `owner_id` + backfill to seed admin + create by user role, (3) owner checks on get/mutate with admin GET-all-by-id, (4) profile and admin account commands.

### Breaking change (clients and consumers)

| Surface | Break | Detail |
|---|---|---|
| HTTP `/tasks` (all methods) | Yes | Today unauthenticated success; afterwards missing identity must fail (`BR-TASKDEMO-010` on create; identity required to evaluate `BR-TASKDEMO-042` on other ops). |
| HTTP create | Yes | Same body (`title`, `dueDate`) but success now implies an owner; anonymous create disappears. |
| HTTP get/start/complete/reschedule | Yes | UUID is no longer a sufficient capability; another user’s task is rejected. |
| Task success JSON | Additive | `ownerId` added on `data`. |
| Error codes / status | Yes | 401 and 403 are new; not-owner reuses 404. |
| OpenAPI `security: []` on tasks | Yes | Task routes currently have no security scheme. |
| RabbitMQ `TaskCompleted` | No, if payload unchanged | In-repo there is no consumer; external queue consumers keep `taskId` + `completedAt`. Adding fields is additive for typical consumers; removing/renaming would break. |
| `GET /health` | No | Stays public. |

### Data migration

| Item | Need |
|---|---|
| `tasks.owner_id` | New non-null FK (or equivalent) once accounts exist. Current migration `20261027204911_create_tasks` has no owner column. |
| Existing task rows | Backfill `owner_id` to the seeded admin, then enforce NOT NULL. |
| Accounts | New table(s); seed at least one admin. |
| Tokens / sessions | New store if revocation is required on deactivate/soft-delete. |
| Down migration | Drop owner FK/column; dropping accounts while tasks reference them needs an ordered down. |
