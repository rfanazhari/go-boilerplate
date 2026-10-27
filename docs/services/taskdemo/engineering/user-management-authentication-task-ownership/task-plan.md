# Task Plan: User Management, Authentication, and Task Ownership

## Legend

- ID: T1, T2, ... (stable — never renumber on revision)
- Complexity: S (< 2h) | M (2–4h) | L (> 4h)
- Status: [ ] Todo | [~] In Progress | [x] Done | ~~obsolete~~
- ref: technical-design.md section this task implements
- DoD: definition of done (verifiable)
- **Target:** `new` = files that do not exist yet; `legacy` = existing files this task must change
- Work order is **bottom-up**: migration → domain → user management → auth → middleware RBAC → task ownership (including RabbitMQ payload lock)

## Tasks

### Sequential (must be done in order)

- [x] **T1. Schema: accounts, tokens, unique live email, seed admin, task owner backfill** — infrastructure/migrate — L  
      — depends: none — ref: design §3, decision-log D4 D5 D6  
      — **Tujuan:** Tabel akun dan token ada; email unik hanya untuk akun yang belum soft-delete; admin pertama bisa di-seed; baris `tasks` lama mendapat `owner_id` lalu kolom wajib.  
      — **File (new):** `migrations/pgsql/<timestamp>_accounts_tokens_task_owner.up.sql`, `*.down.sql`  
      — **File (legacy):** `cmd/app/.env.example` (`SEED_ADMIN_*`); `infrastructure/config/config.go`; optionally `infrastructure/runtime/runtime.go` if seed runs at process start rather than SQL  
      — **Test yang harus lulus:** migration up/down on a database that already has at least one `tasks` row (`go test -tags=integration ./test/infrastructure/...` or a focused migrate test); unique email succeeds twice after soft-delete (can wait for T3 if seed-only here); config parses new env vars  
      — **DoD:** Up migration applies; down drops tokens, `owner_id`, accounts in that order; existing task rows have `owner_id` = seeded admin and NOT NULL; `.env.example` documents seed vars without real secrets

- [x] **T2. Domain: Account aggregate and Task owner** — domain — M  
      — depends: T1 — ref: design §4  
      — **Tujuan:** Model akun (role, status, soft-delete, profil) dan task wajib punya owner di agregat, tanpa mengubah mesin status BR-001–009.  
      — **File (new):** `domain/aggregate/account.go`, `domain/aggregate/account_test.go`; `domain/enum/role.go`; `domain/enum/account_status.go`; `domain/repository/account.go`; `domain/repository/token.go`; hasher port if placed under `domain/service`  
      — **File (legacy):** `domain/aggregate/task.go`, `domain/aggregate/task_test.go`; `domain/event/task_completed.go` (no field changes — confirm)  
      — **Test yang harus lulus:** `go test ./domain/aggregate -v` — NewTask rejects missing owner; existing title/due/start/complete/reschedule tables still pass; Account: only admin|user; deactivate/activate/soft-delete; profile cannot change role/email/status  
      — **DoD:** `NewTask` / `RestoreTask` carry `ownerID`; `TaskCompleted` still `TaskID` + `CompletedAt` only

- [x] **T3. User management: persist accounts, admin commands, self profile** — application + infrastructure — L  
      — depends: T2 — ref: design §4–5, D2 D4  
      — **Tujuan:** Admin dapat menambah akun (role dan password dipilih admin), menonaktifkan, mengaktifkan kembali, dan soft-delete (bukan diri sendiri, bukan admin terakhir); user mengubah nama/password sendiri. Belum ada HTTP.  
      — **File (new):** `application/usecase/account.go`, `application/usecase/account_test.go`; `application/dto/account.go`; `infrastructure/repository/accountrepository/postgres.go`, `memory.go`; bcrypt hasher implementation; Mockery mocks for new ports  
      — **File (legacy):** `.mockery.yml` if new interfaces; `pkg/errs/codes.go` (`EMAIL_TAKEN`, `LAST_ADMIN`, `ACCOUNT_NOT_FOUND`, `INVALID_ROLE`, `INVALID_STATUS`, `INVALID_PASSWORD`); `translations/errors.en.toml`, `translations/errors.id.toml`  
      — **Test yang harus lulus:** `go test ./application/usecase -v -run Account` (or equivalent table tests) — add account; duplicate live email 409; email reusable after soft-delete; last-admin and self-target 409; profile updates name/password hash only. Hasher unit test for bcrypt compare. Optional: `test/infrastructure` account repository round-trip  
      — **DoD:** Memory + Postgres account adapters; no delivery routes yet

- [ ] **T4. Auth: login, hashed opaque token, revoke on deactivate/soft-delete** — application + infrastructure — M  
      — depends: T3 — ref: design §5, D1 D2 D5  
      — **Tujuan:** Login akun active menerbitkan token opaque; token disimpan ter-hash; login ditolak untuk non-active/soft-deleted dengan kode yang sama seperti password salah; deactivate/soft-delete menghapus semua token akun itu.  
      — **File (new):** `application/usecase/login.go`, `application/usecase/login_test.go`; `application/dto/login.go`; `infrastructure/repository/tokenrepository/postgres.go`, `memory.go`  
      — **File (legacy):** account deactivate/soft-delete use case (call token delete in the same transaction); `pkg/errs` (`INVALID_CREDENTIALS`, `UNAUTHENTICATED`)  
      — **Test yang harus lulus:** `go test ./application/usecase -v -run Login` — success stores hash not raw token; wrong password / inactive / deleted → `INVALID_CREDENTIALS`; after deactivate, GetByHash misses the old token  
      — **DoD:** Token repository DeleteByAccountID used by T3 lifecycle; still no HTTP

- [ ] **T5. Middleware RBAC and HTTP for login, accounts, profile** — delivery — M  
      — depends: T4 — ref: design §2 §7, D3  
      — **Tujuan:** Identity dari Bearer di middleware; role admin vs user di subrouter, bukan di use case; health dan login tetap publik.  
      — **File (new):** `delivery/http/auth.go` (middleware + login handler); `delivery/http/auth_test.go`; `delivery/http/account.go`; `delivery/http/profile.go`  
      — **File (legacy):** `delivery/http/router.go`; `delivery/http/envelope.go` (401/403 kinds); `pkg/errs` kinds `KindUnauthenticated`, `KindForbidden`; `infrastructure/runtime/runtime.go` (wire handlers); `docs/api/openapi.yaml`; `docs/api/CHANGELOG.md`  
      — **Test yang harus lulus:** `go test ./delivery/http -v` — no token → 401; user `POST /accounts` → 403; admin `POST /accounts` → 201; `GET /health` 200 without Authorization; login 200 then Bearer on profile  
      — **DoD:** Admin group and user group registered; task routes still **not** required to be correct for ownership (that is T6) but may already sit behind RequireAuth if that simplifies the mux — if so, existing smoke tests will fail until T6 and must be updated in T6 the same PR slice

- [ ] **T6. Task ownership on HTTP/use case and RabbitMQ payload lock** — application + delivery + infrastructure — L  
      — depends: T5 — ref: design §2 §4 §5, D3 D6  
      — **Tujuan:** Create meng-stamp owner = caller (user only); get/mutate hormati `BR-TASKDEMO-042`; admin GET any id; admin mutate/create ditolak (403 create via middleware, 404 mutate); JSON task menambah `ownerId`; event RabbitMQ **tidak** menambah field.  
      — **File (new):** none required beyond tests  
      — **File (legacy):** `application/dto/task.go`; `application/usecase/task.go`, `application/usecase/task_test.go`; `delivery/http/task.go`; `delivery/http/router.go` (`POST /tasks` on user group); `infrastructure/repository/taskrepository/postgres.go`, `memory.go`, `unavailable.go`, `model/task.go`; `infrastructure/service/rabbit.go` (verify only); `test/smoke/example_api_test.go`; `docs/api/openapi.yaml`  
      — **Test yang harus lulus:**  
        - `go test ./application/usecase -v -run Task` — create sets owner; user foreign get/mutate → `TASK_NOT_FOUND`; admin get foreign OK; admin mutate foreign → `TASK_NOT_FOUND`; complete still publishes `TaskCompleted` with only TaskID + CompletedAt  
        - `go test ./infrastructure/service -v` or existing publisher tests — JSON keys exactly `taskId`, `completedAt`  
        - `go test ./test/smoke -v` — login as seeded/user then full create/start/complete/get with Bearer; `/tasks` without token 401; admin create 403  
        - `go test ./domain/aggregate ./application/... ./delivery/http -v`; `make test.arch`  
      — **DoD:** OpenAPI task view has `ownerId`; clients cannot send owner; Rabbit payload unchanged; smoke uses auth; `make lint` clean

## Dependency Graph

```
T1 (migration)
 └─ T2 (domain)
     └─ T3 (user management)
         └─ T4 (auth / tokens / revoke)
             └─ T5 (middleware RBAC + account/login HTTP)
                 └─ T6 (task ownership + RabbitMQ payload lock)
```
