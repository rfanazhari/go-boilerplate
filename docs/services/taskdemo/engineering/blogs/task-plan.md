# Task Plan: Blogs

## Legend

- ID: T1, T2, ... (stable — never renumber on revision)
- Complexity: S (< 2h) | M (2–4h) | L (> 4h)
- Status: [ ] Todo | [~] In Progress | [x] Done | ~~obsolete~~
- ref: technical-design.md section this task implements
- DoD: definition of done (verifiable)
- **Target:** `new` = files that do not exist yet; `legacy` = existing files this task must change
- **Ticket / commit:** use the ticket as the commit subject prefix. One commit per task. Subject states why.
- Work order is strictly sequential: migration → domain → repository → use case → handler/route → cross-cutting test.
- Layer tests ship with the layer that introduces the behavior, so that task can be verified before the next one starts. T6 is the closing pass across packages, smoke, architecture, and lint. Do not leave a layer’s own tests for T6.

Mode A. No blog code exists yet. Decisions D1–D5 are active; no task is blocked.

## Tasks

### Sequential (must be done in order)

- [x] **T1. Migration: blogs table, global slug uniqueness, separate soft delete** — infrastructure/migrate — M
      — depends: none — ref: design §3, decision-log D1 D3
      — **ticket:** `BLOG-T1`
      — **commit:** `BLOG-T1: add blogs table so publication and soft delete stay separate`
      — **Target:** `new`
      — **Tujuan:** Store a blog owned by an account, with draft/published status, `deleted_at`, and a slug that stays unique after soft delete. Do not change `accounts` or `tasks`.
      — **File (new):** `migrations/pgsql/<version>_create_blogs.up.sql`, `migrations/pgsql/<version>_create_blogs.down.sql` with `<version>` greater than `20261027225508`; `test/infrastructure/migration_blogs_test.go` (build tag `integration`)
      — **File (legacy):** none
      — **Test yang harus lulus:** `go test -tags=integration ./test/infrastructure/ -run TestBlogsMigration -count=1` (skips when Postgres is unreachable). Up creates `blogs` and the unique slug index. Down drops only `blogs` and leaves `accounts` and `tasks` in place. The existing accounts migration test still targets its own version.
      — **DoD:** Up and down SQL match design §3. No blog seed row. No task or account column changes.

- [x] **T2. Domain: blog aggregate, title, content, slug, publication status** — domain — L
      — depends: T1 — ref: design §3 §4 §6, decision-log D1 D3
      — **ticket:** `BLOG-T2`
      — **commit:** `BLOG-T2: enforce blog lifecycle and content rules in the domain`
      — **Target:** `new` for blog types; `legacy` for shared error codes and translations
      — **Tujuan:** A new blog is a draft owned by its creator id. Title, sanitized-content length, slug shape, publish/unpublish, and soft delete are enforced by the blog model. Task and account aggregates stay unchanged.
      — **File (new):** `domain/aggregate/blog.go`, `domain/aggregate/blog_title.go`, `domain/aggregate/blog_content.go`, `domain/aggregate/blog_slug.go`, and matching `*_test.go` in package `aggregate_test`; `domain/enum/publication_status.go`; `domain/repository/blog.go`; `domain/service/html_sanitizer.go`
      — **File (legacy):** `pkg/errs/codes.go`; `translations/errors.en.toml`; `translations/errors.id.toml`
      — **Test yang harus lulus:** `go test ./domain/aggregate/ ./domain/enum/ -count=1`. Cover title trim and rune bounds 10/200/201, content plain-text examples from design §3, draft versus published bounds, slug fallback `blog` and immutability, publish/unpublish idempotence, soft-deleted mutation refusal, and `RestoreBlog` loading a short or deleted row without rewriting it. Existing task and account tests still pass.
      — **DoD:** New content can be built only through the sanitizer port plus domain measurement. `INVALID_TITLE` and task status values are not reused. `go test ./domain/aggregate/ -count=1` passes.

- [x] **T3. Repository: Postgres, memory, and HTML sanitizer** — infrastructure — L
      — depends: T2 — ref: design §3 §4 §5, decision-log D1 D3 D5
      — **ticket:** `BLOG-T3`
      — **commit:** `BLOG-T3: persist blogs and apply the same public visibility in Postgres and memory`
      — **Target:** `new`
      — **Tujuan:** Save and load blogs. Public list and public detail use one eligibility rule (published, not deleted, author active and not deleted) before paging. Admin list supports the deletion filter. Memory and Postgres agree. HTML allowlist is the bluemonday policy in design §4.
      — **File (new):** `infrastructure/repository/blogrepository/postgres.go`, `memory.go`, `unavailable.go`; `infrastructure/service/html_sanitizer.go`, `infrastructure/service/html_sanitizer_test.go`; repository tests in `infrastructure/repository/blogrepository` (`_test` package) and an integration test under `test/infrastructure/` for the Postgres adapter
      — **File (legacy):** none in this task (runtime wiring is T5). `.mockery.yml` may wait until T4.
      — **Test yang harus lulus:** `go test ./infrastructure/service/ ./infrastructure/repository/blogrepository/ -count=1`. Sanitizer cases: allowed tags kept, other tags and attributes removed, `href` only for absolute `http`/`https`. Memory repository: public list and detail hide the same fixtures (draft, deleted, inactive author, deleted author); paging happens after that filter; slug unique including deleted; `RunInTransaction` holds the mutex across the callback. Integration Postgres test, when `DATABASE_URL` is reachable: round-trip, unique slug, `SELECT … FOR UPDATE` inside `pgxtx`, public join. Skips when Postgres is down.
      — **DoD:** Both adapters implement `domain/repository.BlogRepository`. Plain structs, raw SQL `$1` placeholders, no repository constructor. Public queries join `accounts` rather than filtering only in Go after `LIMIT`.

- [x] **T4. Use case: commands, disclosure, and list paging** — application — L
      — depends: T3 — ref: design §4 §5 §6 §7, decision-log D1 D2 D4 D5
      — **ticket:** `BLOG-T4`
      — **commit:** `BLOG-T4: orchestrate blog commands and reads without changing task behavior`
      — **Target:** `new`, plus Mockery config
      — **Tujuan:** Create (user role only, caller becomes owner, status draft, slug minted), revise, publish, unpublish, soft delete, private/admin get, public list/detail, and admin list. Map hidden blogs to `BLOG_NOT_FOUND`, forbidden operations on a visible blog to `FORBIDDEN`, and admin edits of a deleted blog to `BLOG_DELETED`. Persist the sanitizer output.
      — **File (new):** `application/dto/blog.go`; `application/usecase/blog.go`; `application/usecase/blog_test.go`; factories in `test/testdata/testdata.go` if the existing helpers are the right home; generated `test/mocks/blog_repository_mock.go`
      — **File (legacy):** `.mockery.yml` (add `BlogRepository`; generate, do not hand-write)
      — **Test yang harus lulus:** `go test ./application/usecase/ -run Blog -count=1`. Include: admin create forbidden; owner and timestamps assigned; slug unchanged on rename; invalid published edit writes nothing; publish content 99/100/1000/1001; idempotent publish and unpublish; admin cannot edit or publish; admin can unpublish; owner and admin delete; second owner delete is not found; second admin delete succeeds with no write; stranger gets 403 on a public blog and 404 on a draft; public get of a draft is not found; inactive author hidden from public reads and visible again when active; page defaults and `pageSize` 101 rejected. Task use case tests still pass.
      — **DoD:** No import of `delivery` or `infrastructure`. No call into `TaskUseCase`. No `CompletionPublisher`. Client-supplied owner, slug, and timestamps are not accepted.

- [x] **T5. HTTP: public routes, management routes, composition, contract** — delivery — L
      — depends: T4 — ref: design §2 §5 §7, decision-log D2 D3 D4
      — **ticket:** `BLOG-T5`
      — **commit:** `BLOG-T5: expose public slug reads and authenticated blog management over HTTP`
      — **Target:** `new` handler; `legacy` router, runtime, API docs
      — **Tujuan:** Anonymous `GET /blogs` and `GET /blogs/{slug}`. Authenticated `/blog-records` for create (user), admin list (admin), and id-based get/update/publish/unpublish/delete. Existing health, login, account, and task routes keep their current auth. Envelope and status codes match design §2.
      — **File (new):** `delivery/http/blog.go`
      — **File (legacy):** `delivery/http/router.go`; `infrastructure/runtime/runtime.go`; `docs/api/openapi.yaml`; `docs/api/CHANGELOG.md`; `docs/api/endpoints.md` (add a Blogs section only)
      — **Test yang harus lulus:** `go test ./delivery/http/ -count=1` (existing auth tests still pass; do not add delivery unit tests). Package compiles with the new routes. OpenAPI marks public blog operations `security: []` and leaves global bearer security on management operations. Create is `201`; other blog successes are `200`.
      — **DoD:** Public blog routes are registered on the root router before the authenticated subrouter. Management routes are only under `/blog-records` on that subrouter. `POST /blog-records` uses `RequireRole(user)`. `GET /blog-records` uses `RequireRole(admin)`. Blog UUID parsing returns `INVALID_BLOG_IDENTIFIER`, not `pathID`. Runtime wires Postgres, memory (`INFRA_OPTIONAL`), unavailable, sanitizer, clock, and the handler. No new environment variable.

- [x] **T6. Verification: smoke, architecture, full test, lint** — test — M
      — depends: T5 — ref: design §6 §8, tactical-strategy §5
      — **ticket:** `BLOG-T6`
      — **commit:** `BLOG-T6: prove public visibility, moderation, and existing task routes still hold`
      — **Target:** `new` smoke coverage; `legacy` only if a smoke helper must learn the new routes
      — **Tujuan:** Prove the HTTP contract and the regression surface the layer tests do not compose: anonymous public read, token ignored on `/blogs`, management requires a token, admin cannot create, draft and deleted blogs stay off the public channel, and task/account routes still behave.
      — **File (new):** `test/smoke/blog_api_test.go` (or the existing smoke package’s equivalent)
      — **File (legacy):** `test/smoke/example_api_test.go` only if shared setup must stay green; do not weaken task assertions
      — **Test yang harus lulus:**
        - `go test ./test/smoke/ -run Blog -count=1` using `INFRA_OPTIONAL` memory adapters
        - `go test ./... -count=1`
        - `make test.arch`
        - `make lint`
        - `go test -tags=integration ./test/infrastructure/ -count=1` when Postgres is reachable; otherwise record the skip
      — **DoD:** Smoke covers the cases in tactical-strategy §5 (happy path, 400/401/403/404/409, public versus management). `make test.arch` still forbids outward imports. Lint is clean. Task completion and account routes are unchanged.

## Dependency Graph

```
T1 (migration, BLOG-T1)
 └─ T2 (domain, BLOG-T2)
     └─ T3 (repository, BLOG-T3)
         └─ T4 (use case, BLOG-T4)
             └─ T5 (handler/route, BLOG-T5)
                 └─ T6 (cross-cutting test, BLOG-T6)
```

No parallel tasks. T6 starts only after T5 is done.
