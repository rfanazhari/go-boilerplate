# Tactical Strategy: Blogs

## 1. Implementation Strategy

Mode A, additive only. [Technical design](technical-design.md) is the target contract; [capability](../../business/capabilities/blogs.md) is the business authority and [delta](../../delta-report/delta-report-blogs.md) identifies the current baseline. Root AGENTS.md and source substitute for absent CLAUDE.md. Existing account/auth implementation is a prerequisite already present, not a new Blogs task.

Implement in a single dependency chain: migration, domain rules/ports, repository adapters, use cases, handlers/routes/runtime, API contract documentation, then tests and final checks. Do not expose routes before composition is ready. Each task ends with a verifiable artifact; final tests demonstrate feature correctness. The requested test phase follows handler work; it does not waive existing test checks or focused reruns whenever a test is edited.

Avoid refactoring task/auth behavior to implement Blogs. Reuse mechanisms and create Blogs-specific rules. The feature requires no new environment variables, RabbitMQ topology, account write cascade, or deployment service. Retain runtime INFRA_OPTIONAL behavior with the new memory adapter and current account source.

## 2. Code Reuse Map

| Component | Verified location | Reuse / adaptation |
|---|---|---|
| Bearer/current account authentication | `delivery/http/auth.go` | Reuse Auth.Middleware and authenticate unchanged for protected groups |
| Role gate and actor extraction | `delivery/http/auth.go` | RequireRole(user) for create, admin for admin reads; map Actor to BlogActor |
| Current account lookup and enums | `domain/repository/account.go`, `domain/enum/role.go`, `account_status.go` | Reuse identity/status facts for use-case checks and memory public eligibility |
| Aggregate style and input/view separation | `domain/aggregate/task.go`, `application/dto/task.go`, `application/usecase/task.go` | Follow encapsulation/DTO/clock pattern; create different lifecycle/validation/permissions |
| Shared envelope/localization | `delivery/http/envelope.go`, `infrastructure/i18n/bundle.go` | Reuse JSON/status mapping and localization unchanged; add messages only |
| Input/UUID/time helpers | `delivery/http/task.go`, `delivery/http/time.go` | Reuse UUID/time helpers; adapt strict blog JSON presence/EOF checks, not task decoding behavior |
| Error types and codes | `pkg/errs/codes.go` | Reuse kinds/wrapping and Error definition in this file; add blog codes |
| SQL/plain struct pattern | `infrastructure/repository/accountrepository/postgres.go`, `taskrepository/postgres.go` | Parameterized SQL and Postgres{Pool}; new blog insert/update/locking/query behavior |
| Shared transaction context | `infrastructure/repository/pgxtx/context.go` | Use With/From, as account adapter does; task adapter uses its own private context |
| Memory clone pattern | `infrastructure/repository/accountrepository/memory.go`, `taskrepository/memory.go` | Follow detached-copy storage; replace no-op transaction pattern with staged blog transactions |
| Runtime composition | `infrastructure/runtime/runtime.go` | Add BlogRepository/BlogUseCase/BlogHandler in both adapter modes |
| Migration application | `cmd/migrate`, `migrations/pgsql/` | Append timestamped pair; runner unchanged |
| Unicode normalization | `golang.org/x/text` in `go.mod` | Existing direct dependency for deterministic slug normalization |
| HTML parser | Resolved `golang.org/x/net/html` v0.60.0 in module cache/go.mod | Promote existing indirect dependency to direct; explicit allowlist rebuild is new policy, parser alone is insufficient |
| Clock, testify, Mockery | `domain/service/clock.go`, `test/testdata/testdata.go`, `.mockery.yml` | Reuse deterministic clock and named table cases; generate BlogRepository mock |
| Architecture/smoke tooling | `test/architecture/architecture_test.go`, `test/smoke/`, `Makefile` | Existing checks plus Blogs regressions |

The existing task title/status/permissions and CompletionPublisher are not reusable Blogs policies. Keep their behavior intact.

## 3. Net-New Code Inventory

| Item | Layer/component | Type / notes |
|---|---|---|
| `20261029000100_create_blogs.{up,down}.sql` | Infrastructure schema | Blog-only table, constraints, slug uniqueness, listing indexes |
| `domain/enum/blog_status.go` | Domain | Distinct draft/published enum |
| `domain/aggregate/blog.go` | Domain | Creation, candidate edit, publication, unpublication, soft deletion, immutable identifiers |
| `domain/service/blog_content.go` | Domain | Closed HTML policy, safe serialization and normalized text counting |
| `domain/service/blog_slug.go` | Domain | Fixed deterministic normalization/fallback/UUID suffix |
| `domain/service/blog_access.go` | Domain | Named actor/visibility facts, owner/admin operations and deletion rules |
| `domain/repository/blog.go` | Domain port | Commands, transactions, locking, public/admin projections and pagination |
| `infrastructure/repository/blogrepository/postgres.go` | Infrastructure | pgx SQL adapter, joined public queries, snapshot paging and row locks |
| `infrastructure/repository/blogrepository/memory.go` | Infrastructure | Mutex/copies, staged transactions, live owner eligibility and matching read semantics |
| `application/dto/blog.go`, `application/usecase/blog.go` | Application | Named actor/inputs/views; protected actor checks, atomic commands and distinct read methods |
| `delivery/http/blog.go` | Delivery | Strict blog JSON, query validation, slug vs UUID input, response shapes |
| Router/runtime edits | Delivery/infrastructure | Public routes before broad protected routes; role groups and both adapter modes |
| New codes/translations | Shared/delivery | INVALID_BLOG_TITLE/CONTENT/PAGINATION/FILTER, BLOG_NOT_FOUND/DELETED/IDENTIFIER_CONFLICT |
| Mockery config + generated BlogRepository mock | Test tooling | Generation follows existing config; no handwritten mocks |
| Domain/application/HTTP/memory/SQL/migration/smoke tests | Tests | Behavior matrix below; real DB tests tagged integration |
| API documentation updates | `docs/api/` | OpenAPI 1.2.0, endpoint guide, changelog; explicit public security overrides |

All paths in this inventory are proposed except reuse-map evidence. Verify no naming collision immediately before implementation.

## 4. Branching / PR Strategy

Use one feature branch with one coherent review containing the ordered commits. No branch, commit, push, or PR is requested by this documentation task. If later authorized, use stable `BLOGS-001`–`BLOGS-016` IDs from [task plan](task-plan.md), e.g. `feat(blogs): [BLOGS-001] add blog schema` or `test(blogs): [BLOGS-015] verify SQL visibility and locking`. Never renumber IDs on revision; append new IDs, preserve completed work and mark obsolete tasks explicitly.

Migration commits precede dependent application changes. Intermediate commits are execution checkpoints, not separately deployable releases; merge/deploy only the validated complete feature. Existing pre-feature binaries can run with the additive blog table present. If reviews are split later, schema/domain first, adapters/use cases second, delivery/docs/tests last, with no partial public launch.

## 5. Testing Strategy

Use named table-driven structs, testify assertions, controlled clocks/IDs, external `_test` packages for domain/application, and Mockery-generated mocks. After each test edit run its focused `go test ./path -v -run TestName`. Do not use go build/go run as verification. The detailed test tasks occur after route/contract tasks per the requested ordering.

### 5.1 Unit and application tests

| Rules / concern | Required evidence |
|---|---|
| 050–054 identity/ownership | User create stamps actor; admin create 403; forged owner fields rejected; all protected methods deny missing/inactive/deleted actors; direct use-case calls cannot bypass checks |
| 060–067 lifecycle/moderation | New draft; owner-only edit/publish; admin can withdraw/delete but cannot edit/publish another author's work; other user denials distinguish visible 403 from hidden 404 |
| 070–075 visibility | Anonymous and header-bearing public reads behave identically; drafts/deleted/ineligible-owner excluded; owner/admin header never elevates public access; owner cannot read deleted; admin reads deleted only via admin surface |
| 080 title | Empty/whitespace, 9/10/200/201 code points, multibyte and combining characters, trim, invalid UTF-8 |
| 081–083 content | Draft empty/99 accepted, max1000; published 99/100/1000/1001; whitespace-only; valid published edit immediately changes content; rejected edit leaves all data/status/timestamps unchanged |
| HTML | Every allowed tag, disallowed tags/attrs, script/style/foreign subtree removal, malformed nesting, entity/ampersand round trips, block/br whitespace, nested unsupported blocks, encoded dangerous href, relative/protocol-relative URL, duplicate href, comments; reparse stored output and assert closed policy |
| 084 slug | Identical titles with distinct IDs, NFKD accents, non-Latin fallback, base truncation, separators, lowercase 12-hex suffix, exact lookup, rename stability, delete reservation, five collision attempts, retry only known constraints |
| Idempotency/state | Repeat authorized publish/unpublish unchanged; admin repeat delete 200 unchanged; owner repeat delete 404; deleted admin edit/publish/unpublish 409; unauthorized no-op still denied |
| Transactions | Callback/update/commit errors propagate; no early success; no persistence on validation denial; no task completion publisher dependency |

Use generated BlogRepository/AccountRepository mocks to isolate orchestration. Exercise real HTML/domain services rather than mocking the policy under test. Test detached object copies and staged rollback in memory, including an explicit callback error after a staged update.

### 5.2 HTTP and contract tests

Cover all ten operations, response field sets and success codes, malformed/null/unknown/trailing JSON, missing-vs-null patch fields, invalid UUID, malformed/unknown slug, and shared envelope. Exercise English/Indonesian errors. Public GET with no token, invalid token, revoked token, owner token and admin token must use the same eligible set. Protected routes require valid identity. Test method/path matching so GET `/blogs` does not capture POST or fall into required authentication, and `/private`/`/admin` cannot be handled as slug routes.

Page tests: defaults, 1/100 bounds, zero/negative/101, empty/non-decimal/duplicate/unknown queries, integer/offset overflow, equal timestamps with descending UUIDs, filtered totals and out-of-range empty pages. Admin tests exercise all deletion filters and default, draft/published and active/ineligible owners; list items omit content. Verify OpenAPI route/security/schema declarations against these assertions and actual router behavior; the spec file alone is not runtime proof.

### 5.3 PostgreSQL and migrations

Add integration-tagged scenarios under `test/infrastructure/` using an isolated disposable database. Test current account version → blog version → account version → blog version, retaining pre-existing account/task/token rows across blog up/down. Verify FK, status/title constraints, global slug uniqueness after delete, and index existence. Do not assume the older fixed-version migration test covers Blogs.

Test repository insert/update and immutable fields, rollback, joined list/detail eligibility, exact count+items including out-of-range pages, deleted admin reads, SQL parameterization, tied order, and current author names. Change owner active → non_active → active and soft-delete owner; verify list and detail change without blog status updates. Coordinate concurrent edit/delete and publish/unpublish calls with barriers; confirm row locks prevent a stale write resurrecting a deleted blog. A real unique violation must not leave collision retry using an aborted transaction.

Migration tests that move schema versions must not run concurrently with repository tests. Use dedicated test databases or serialize migration cases and restore the required latest schema before adapter cases. A skipped live test means unverified, not passed.

### 5.4 Regression and final verification

Keep existing health/login, task ownership/lifecycle, account management/profile, token revocation, localization and TaskCompleted tests passing. Extend INFRA_OPTIONAL smoke with create → private draft → public hidden → publish → slug read/list → unpublish/republish → delete → hidden, using controlled fixtures and no new public bootstrap endpoint.

Final commands: `go test ./...`, `make test`, `make test.arch`, `make test.smoke`, `make lint`. On isolated live infrastructure also run `go test -tags=integration ./test/infrastructure/...`; separate migration/repository runs with `-run` if sharing a database. Report pass/fail/skip and tool availability faithfully. This planning change itself needs document/reference checks, not application test execution.

## 6. Rollout & Rollback

Rollout: apply additive blog migration after accounts, deploy the fully validated application, verify public/owner/admin smoke scenarios and owner deactivation visibility. Existing compose migration runner and `db`/`rabbitmq` hostnames remain. No new feature flag/config is required. Use existing HTTP logging to inspect unexpected 5xx and authorization/visibility regressions without logging tokens/raw blog bodies.

If private/deleted content leaks, restrict the new blog routes or revert the feature application release immediately; a visibility leak is not an acceptable partial rollout. Application rollback retains the blog table and data, while restoring the earlier router/runtime/catalog version. Reverting Blogs does not revert account migrations or task behavior.

Run down migration only for an explicitly requested data-discarding rollback after backup/export, with blog-serving binaries stopped. Drop only blog-owned objects; retain accounts/tasks/tokens. A later up migration recreates empty blogs and cannot restore deleted data or prior slug reservations. Production data preservation favors leaving the additive schema during rollback. Revisit destructive rollback if retention/history requirements change (decision-log D1).
