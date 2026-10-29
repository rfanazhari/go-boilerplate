# Tactical Strategy: Blogs

## 1. Implementation Strategy

Additive module. No change to task routes, task lifecycle, account contracts, or the `TaskCompleted` payload.

Ship on one branch in the task-plan order: migration, domain, repository, use case, HTTP and contract, then the cross-package verification. Each task is one commit. The commit subject is the ticket line in `task-plan.md` (`BLOG-T1` … `BLOG-T6`).

Public reads and management writes belong in the same release. An empty `blogs` table is safe to migrate before the handlers exist. There is no feature flag and no new environment variable.

`INFRA_OPTIONAL=true` uses the memory blog adapter so smoke tests do not need Postgres. Production compose stays Postgres and RabbitMQ; RabbitMQ is still only for task completion.

## 2. Code Reuse Map

Verified against the current tree. “Reused as-is” means the existing code is called and not modified. “Wrap” means the mechanism stays and the blog code supplies new policy.

| Component | Location | Used for | Fit and limitations |
| --- | --- | --- | --- |
| `Auth.Middleware`, `authenticate`, `RequireRole` | `delivery/http/auth.go` | 401 for missing, bad, inactive, or soft-deleted tokens; 403 for user versus admin routes | Fits protected `/blog-records`. Does not belong on `GET /blogs`. Cannot express author-or-admin. |
| `ActorFromContext` | `delivery/http/auth.go` | Handler copies `ID` and `Role` into `dto.BlogActor` | Delivery type must not be imported by domain or application. `dto.TaskActor` is task-specific. |
| Router registration order | `delivery/http/router.go` | Public routes on the root router; everything under the authenticated subrouter requires a token | `/health` and `/login` already do this. A slug route under `/blogs` must not be the management GET. |
| `writeSuccess`, `writeError`, `decodeJSON`, `formatTime` | `delivery/http/envelope.go`, `task.go`, `time.go` | Envelope, unknown-field rejection, timestamps | `decodeJSON` does not sanitize HTML. `pathID` reports `INVALID_IDENTIFIER` with task wording; blog ids need a separate helper. |
| Error kinds | `pkg/errs/codes.go` | 400/401/403/404/409/500 mapping already in `writeError` | Add blog codes. Do not change existing message text. |
| `AccountRepository.GetByID`, account status and deletion | `domain/repository/account.go`, `domain/aggregate/account.go` | Author name on management reads; public SQL joins `accounts` | A per-id lookup does not implement a paged public list. |
| `Clock`, `SystemClock` | `domain/service/clock.go` | `created_at` / `updated_at` and `deleted_at` | Same truncation as tasks. |
| `pgxtx.With` / `From` | `infrastructure/repository/pgxtx/context.go` | Blog transactions and `SELECT … FOR UPDATE` | Account and token adapters already use it. Task Postgres uses a private context key; new code uses `pgxtx`. |
| Migration runner | `cmd/migrate`, `migrations/pgsql/` | Next version after `20261027225508` | Do not edit the accounts migration or its seed. |
| Runtime composition | `infrastructure/runtime/runtime.go` | Placeholder unavailable, memory when `INFRA_OPTIONAL`, Postgres when the pool is up | Copy that order for blogs. No new config. |
| Mockery, testify, architecture test | `.mockery.yml`, `test/architecture/architecture_test.go` | Generate `BlogRepository`; keep inward imports | Do not hand-write mocks. |
| Task aggregate, `TaskUseCase`, task repositories, `CompletionPublisher` | `domain/aggregate/task.go`, `application/usecase/task.go`, `infrastructure/repository/taskrepository`, `infrastructure/service/rabbit.go` | Left in place for tasks | Different status, title bounds, and permissions. Blog publish must not call the publisher. Task memory’s lock-free `RunInTransaction` is not the blog transaction. |

## 3. Net-New Code Inventory

Each row is a responsibility. A value object owned by `Blog` stays a value object even though it lives in `domain/aggregate`.

| Item | Layer/Component | Package/Path | Type | Ownership / Notes |
| --- | --- | --- | --- | --- |
| `Blog` | domain | `domain/aggregate/blog.go` | aggregate root | Owns id, owner id, status, `deleted_at`, timestamps. Holds the three values below. |
| `BlogTitle` | domain | `domain/aggregate/blog_title.go` | value object | Trim and 10–200 runes. Not the task title. |
| `BlogContent` | domain | `domain/aggregate/blog_content.go` | value object | Sanitized HTML plus plain-text length. Owned by `Blog`. |
| `BlogSlug` | domain | `domain/aggregate/blog_slug.go` | value object | Normalization and shape. Suffix is supplied by the use case. |
| `PublicationStatus` | domain | `domain/enum/publication_status.go` | enum | `draft`, `published`. |
| `HTMLSanitizer` | domain | `domain/service/html_sanitizer.go` | port | `Sanitize(raw string) string`. Implemented in infrastructure. |
| `BlogRepository` | domain | `domain/repository/blog.go` | port | Persistence and the public/admin queries. |
| Public eligibility | repository queries | `GetPublicBySlug`, `ListPublic` | behavior of those methods | Not a separate type. Same predicate on list and detail. |
| `BlogUseCase` | application | `application/usecase/blog.go` | use case | Commands, disclosure, paging, slug retry. |
| Blog DTOs | application | `application/dto/blog.go` | input and view structs | No validation constructors. |
| Postgres blog adapter | infrastructure | `infrastructure/repository/blogrepository/postgres.go` | adapter | Plain struct, `$1` SQL, `pgxtx`. |
| Memory blog adapter | infrastructure | `infrastructure/repository/blogrepository/memory.go` | adapter | Same rules; mutex held for the transaction callback. |
| Unavailable blog adapter | infrastructure | `infrastructure/repository/blogrepository/unavailable.go` | adapter | Same placeholder role as `taskrepository.Unavailable`. |
| bluemonday policy | infrastructure | `infrastructure/service/html_sanitizer.go` | adapter of `HTMLSanitizer` | Allowlist in technical design §4. |
| Blog handler | delivery | `delivery/http/blog.go` | HTTP translation | No permission matrix. |
| Blogs migration | infrastructure | `migrations/pgsql/<version>_create_blogs.*` | schema | Up and down in technical design §3. |
| Blog error codes | shared | `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | codes and copy | New keys only. |

## 4. Branching / PR Strategy

One branch, six commits, one pull request after T6 is green.

| Commit | Ticket | Contains |
| --- | --- | --- |
| 1 | `BLOG-T1` | Migration pair and the integration migration test |
| 2 | `BLOG-T2` | Domain model, port, sanitizer port, error codes, translations, domain tests |
| 3 | `BLOG-T3` | Adapters, sanitizer implementation, adapter tests |
| 4 | `BLOG-T4` | Use case, DTOs, Mockery config, generated mock, use case tests |
| 5 | `BLOG-T5` | Handler, router, runtime, OpenAPI, changelog, endpoints section |
| 6 | `BLOG-T6` | Smoke and any small fix the full suite requires |

Do not merge after T1 alone if the team expects the branch to serve traffic: the table would exist with no routes, which is harmless, and the capability would be incomplete. Prefer one PR of all six commits.

Safe rollback of that PR is the previous application binary. Run the down migration only when dropping `blogs` is acceptable (decision log D1, technical design §9).

## 5. Testing Strategy

Commands from the repository instructions:

```sh
go test ./...
make test
make test.arch
make lint
go test -tags=integration ./test/infrastructure/...
```

Focused checks after each task are in `task-plan.md`. Tests are table-driven, in `_test` packages, with testify. Domain and use case tests build blogs through constructors and `test/testdata` factories, not struct literals. Mocks come from Mockery.

The cases below are the proof plan. They are not claims that those tests already exist.

### Unit Tests

- **Domain:** title trim and bounds, including `Catatan Go` (10) and 200 versus 201 runes; plain-text table in technical design §3; draft content empty and 1000; publish 99, 100, 1000, 1001; slug fallback `blog` and hyphen collapsing; `Revise` does not change slug; publish and unpublish idempotence does not change `updated_at`; soft delete keeps status; `Publish` on a deleted blog returns `BLOG_DELETED` and changes nothing; `RestoreBlog` returns a short title and a deleted row unchanged.
- **Sanitizer:** `p`, `br`, `h1`–`h5`, and `a` with an absolute `https` href remain; `script`, `img`, `onclick`, `javascript:`, `mailto:`, and a path-relative href do not remain; stored result is the sanitized string.
- **Memory repository:** list and detail agree for draft, deleted, inactive author, and soft-deleted author; page size is applied after the filter; sort is `created_at DESC, id DESC`; a deleted slug cannot be saved again; the transaction callback does not interleave with another save.
- **Use case:** admin create is `FORBIDDEN` and does not save; owner id is the actor; saved content is the fake sanitizer output; invalid published edit does not save; idempotent publish does not save; admin edit and admin publish are `FORBIDDEN`; admin unpublish succeeds; owner delete then owner get is `BLOG_NOT_FOUND`; admin delete twice succeeds and the second call does not save; stranger on a public blog is `FORBIDDEN`; stranger on a draft is `BLOG_NOT_FOUND`; public reads hide an inactive author and show the blog again after the account is active, with status still `published`; `pageSize` 101 is `VALIDATION_FAILED`; omitted page is 1 and omitted size is 20.

### Integration Tests

Run with `go test -tags=integration ./test/infrastructure/ -count=1`. Skip when Postgres is unreachable.

- Happy path: migrate up, insert a blog, read it back, public join hides a draft and a blog whose author is `non_active`, unique slug rejects a duplicate including a deleted row, down migration removes `blogs` and leaves `accounts`.
- Error cases: there is no new HTTP status beyond 400, 401, 403, 404, and 409. The integration package proves the database constraints behind 404 (no row) and the unique slug. HTTP mapping is the smoke pass.

### Regression Tests

- `go test ./domain/aggregate/ ./application/usecase/ ./delivery/http/ -count=1` — existing task and account cases still pass.
- `go test ./test/smoke/ -count=1` — task create/start/complete/get still require Bearer; `/health` stays open.
- `make test.arch` — domain does not import delivery or infrastructure; the new sanitizer port is the other way around.
- `make lint`.

### HTTP smoke (T6)

Use `INFRA_OPTIONAL` memory adapters.

| Case | Expected |
| --- | --- |
| `GET /blogs` and `GET /blogs/{slug}` with no token, eligible blog | 200 |
| Same GETs with a garbage `Authorization` | still 200 for an eligible blog |
| Draft slug, deleted slug, inactive author’s slug | 404 `BLOG_NOT_FOUND`, and the draft is absent from the list |
| `POST /blog-records` without a token | 401 |
| Admin `POST /blog-records` | 403 |
| User create | 201, `status` `draft`, slug set, `ownerId` is that user |
| Owner publish with content length 100 | 200 `published`, then public detail 200 |
| Owner publish with content length 99 | 400, public detail 404 |
| Admin `PATCH` and admin publish of that blog | 403 |
| Admin unpublish | 200, public detail 404 |
| Owner publish again | 200, public detail 200 |
| Owner delete, then owner `GET /blog-records/{id}` | first 200 with `data: {}`, then 404 |
| Admin `GET /blog-records/{id}` and `GET /blog-records?deleted=only` | 200 and the deleted blog is listed |
| Admin `PATCH` of the deleted blog | 409 `BLOG_DELETED` |
| User `GET /blog-records` | 403 |
| `GET /health`, login, and an authenticated task flow | unchanged |

## 6. Rollout & Rollback

- Rollout: deploy the migration with the existing `cmd/migrate` step, then the application. No flag. Watch public `GET /blogs` for unexpected 500s and for 200 responses that include drafts (a predicate bug). Confirm `POST /tasks` still returns 401 without a token and 201 for a user.
- Rollback: deploy the previous application. Leave `blogs` in place if it holds writings. Run the down migration only when the table is empty or product accepts losing every blog row. Down must not drop `accounts`, `credential_tokens`, or `tasks`.
