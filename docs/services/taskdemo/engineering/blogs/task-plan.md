# Task Plan: Blogs

## Legend

- Mode A, new additive feature. Every task is Todo; no implementation was performed to generate this plan.
- Task IDs `BLOGS-001`–`BLOGS-016` are stable and used in commit messages. Append IDs on revision; never renumber or reuse.
- Complexity: S (<2h), M (2–4h), L (>4h); estimates need engineer validation.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done; preserve completed tasks on revision and mark invalid work obsolete.
- `ref` links to [technical-design.md](technical-design.md); DoD is verifiable completion evidence.
- Suggested commit format: `<type>(blogs): [BLOGS-NNN] <result>`. These IDs define future commit attribution; this request does not authorize commits.
- Execute strictly one task at a time. Each task depends on its immediate predecessor and all earlier tasks transitively. Domain/repository/use-case work is new Blogs code; modifications to shared router/runtime/catalogs are legacy integration work.
- Follow AGENTS.md: exact Go edits, gofmt, raw `$1` SQL, plain repository structs, no handwritten mocks. Verify through go test; after editing a test run its named focused test. Dedicated new test implementation follows handler/route work as requested.

## Sequential Tasks

### Migration

- [x] **BLOGS-001 — Add blog-only up/down migration** — Infrastructure schema — M
  - Depends: none. Ref: [§3.1](technical-design.md#31-blog-storage).
  - Deliverables: new `20261029000100_create_blogs.up.sql` and `.down.sql`; recheck version availability before choosing the exact filename.
  - DoD: schema includes owner FK without cascade, draft/published check, title bounds, timestamps, global slug uniqueness including deleted rows, and documented indexes. Down removes only blog-owned objects. Existing migrations/data remain intact. Validate application/reversal against a disposable database if available; final automated migration coverage is BLOGS-015. Record deferred live validation if unavailable.
  - Commit: `feat(blogs): [BLOGS-001] add blog schema`.

### Domain

- [x] **BLOGS-002 — Define blog state and access policy** — Domain/shared errors — M
  - Depends: BLOGS-001. Ref: [§4.1](technical-design.md#41-domain-model-and-html), [§4.3](technical-design.md#43-authorization-and-disclosure), [§7](technical-design.md#7-error-handling).
  - Deliverables: `domain/enum/blog_status.go`, `domain/service/blog_access.go`, additive blog error constants.
  - DoD: named actor/visibility facts express user-only creation, owner-only edit/publish, owner/admin withdrawal/deletion, live vs deleted access and visibility-before-permission denial. Codes/kinds match §7; task enum/codes/policy unchanged. `go test ./domain/... ./pkg/errs/...` passes existing checks.
  - Commit: `feat(blogs): [BLOGS-002] define blog access policy`.
  - Execution (2026-10-29): enum, access policy, additive error codes and focused TDD tests implemented. RED: `42c6f0a8d301a816baf17544d31f35ee51bf828f`. Focused tests, domain/errors DoD, `go test ./...`, `make test`, `make test.arch`, `make test.smoke`, and `make lint` passed. Security passed with user-directed `GOTOOLCHAIN=go1.26.10 make test.security` (0 affecting vulnerabilities). The earlier Go 1.26.0 security failure was also reproduced on the pre-T2 baseline; no dependency or toolchain pin changes were needed. GREEN tests and all mandatory checks passed; T2 is Done.

- [x] **BLOGS-003 — Implement HTML and slug domain policies** — Domain services — L
  - Depends: BLOGS-002. Ref: [§3.2](technical-design.md#32-slug-algorithm), [§4.1](technical-design.md#41-domain-model-and-html).
  - Deliverables: `domain/service/blog_content.go`, `blog_slug.go`; promote the existing x/net module to direct dependency when imported.
  - DoD: single shared policy performs closed allowlist rebuilding, href checks, safe serialization and normalized decoded-text counting; deterministic slug algorithm includes NFKD, fallback, truncation and 12-hex UUID suffix. No HTTP/SQL/outer-layer imports or regex-only sanitizer. Dependency updates do not opportunistically upgrade modules. Existing domain checks pass; adversarial tests follow in BLOGS-012.
  - Commit: `feat(blogs): [BLOGS-003] add content and slug policies`.
  - Execution (2026-10-29): implemented shared HTML rebuilding/readable-text policy and deterministic NFKD slug policy with focused external-package TDD tests. RED: `c7f5b8d447aee4b5c8dd00ae8004f7fe7ec14970`; command (repository root): `go test ./domain/service -v -run 'Test(ProcessBlogContent|BlogSlug)$'`, failed on inert policy outputs before implementation and passed after implementation. User approved targeted x/net upgrade from v0.60.0 to v0.62.0 to fix GO-2026-6860/GO-2026-6866; no other module versions upgraded. Focused tests, `go test ./...`, `make test`, `make test.arch`, `make test.smoke`, `make lint`, and `make test.security` passed. Security reports 0 affecting vulnerabilities (one required-module finding outside called code). Comprehensive adversarial/reparse coverage remains BLOGS-012; no later task implemented. GREEN is the commit containing this checklist update.

- [x] **BLOGS-004 — Add blog aggregate and repository port** — Domain — L
  - Depends: BLOGS-003. Ref: [§3.3](technical-design.md#33-atomic-persistence-and-public-visibility), [§4.1](technical-design.md#41-domain-model-and-html), [§4.2](technical-design.md#42-port-and-application-contracts).
  - Deliverables: `domain/aggregate/blog.go`, `domain/repository/blog.go`, named query/projection/page structs.
  - DoD: aggregate enforces title/content/state/deletion invariants and immutable IDs/owner/slug/createdAt; validated candidate assignment preserves state on rejection; time/no-op rules defined. Port covers separate insert/update, locked read, transaction and public/admin projections without importing application DTOs. `go test ./domain/...` and architecture checks pass existing cases.
  - Commit: `feat(blogs): [BLOGS-004] add aggregate and persistence port`.
  - Execution (2026-10-29): aggregate and domain repository port implemented; focused external-package TDD covers draft/title/content bounds, sanitization, publication, atomic rejection, stable identity/slug, deletion and no-op timestamps. RED: `a060d0471799c845b6c623774afffe4683292b41`; command (repository root): `go test ./domain/aggregate -v -run 'Test(NewBlog|BlogLifecycle|BlogPublishValidation)$'`, failed on inert aggregate outputs and missing validation, then passed after implementation. `go test ./...`, `make test` (race), `make test.arch`, `make test.smoke`, `make lint`, and `make test.security` passed after retrying sandbox-denied Go cache access with authorized access. Lint/security use Go 1.26.10; security reports 0 affecting vulnerabilities, with one required-module finding outside called code. The GREEN commit contains this checklist update. No adapters, use cases, routes, mocks or later tasks implemented; comprehensive domain/adversarial coverage remains BLOGS-012.

### Repository

- [x] **BLOGS-005 — Implement PostgreSQL command persistence** — Infrastructure repository — M
  - Depends: BLOGS-004. Ref: [§3.2](technical-design.md#32-slug-algorithm), [§3.3](technical-design.md#33-atomic-persistence-and-public-visibility), [§4.2](technical-design.md#42-port-and-application-contracts).
  - Deliverables: blogrepository Postgres struct, insert/get/locked-get/update/transaction methods; read methods may land in the next task before wiring the port.
  - DoD: uses pgxtx.With/From, `$1` SQL, row locks and commit/rollback; no update upsert or immutable-column writes. Known unique constraints are distinguished for collision retry; other SQL errors remain internal. No repository constructor. Existing infrastructure tests pass; real DB behavior is verified in BLOGS-015.
  - Commit: `feat(blogs): [BLOGS-005] add transactional postgres writes`.
  - Execution (2026-10-29): implemented plain Postgres command adapter using pgxtx, insert-only Create, reads including deleted rows, transaction-only FOR UPDATE, mutable-column-only Update, and commit/rollback with panic/cancellation cleanup. Only blogs_pkey/blogs_slug_key unique violations map to identifier conflict. RED: `e78157532e98401a21b4cb2c0ca1f62bf7dd4bfd`; focused command at repository root: `go test ./infrastructure/repository/blogrepository -v -run 'TestPostgres(CreateErrors|GetForUpdateRequiresTransaction)$'`, failed because inert stubs swallowed errors/accepted a lock without a transaction, then passed with implementation. `go test ./...`, `make test`, `make test.arch`, `make test.smoke`, `make lint`, and `make test.security` passed (lint/security required cache access outside sandbox). Security: zero affecting vulnerabilities. Error injection does not establish real SQL behavior; live DB locking/rollback verification remains BLOGS-015. Public/admin reads and full port conformance remain BLOGS-006. GREEN includes this checklist update; unrelated Makefile changes excluded.

- [ ] **BLOGS-006 — Implement PostgreSQL public/admin reads** — Infrastructure repository — M
  - Depends: BLOGS-005. Ref: [§2.2](technical-design.md#22-pagination-and-admin-filter), [§3.3](technical-design.md#33-atomic-persistence-and-public-visibility), [§6](technical-design.md#6-request--response).
  - Deliverables: public slug detail, public list and admin list, read projections, compile-time full port conformance.
  - DoD: joined current owner facts enforce all four visibility predicates equally on list/count/detail; order is createdAt DESC/id DESC; count and page use a read-only repeatable-read snapshot including empty pages. Admin filter active/deleted/all and current authorName work without content in lists. No SQL N+1 or dynamic unvalidated filter interpolation. Existing infrastructure checks pass.
  - Commit: `feat(blogs): [BLOGS-006] add eligible public and admin queries`.

- [ ] **BLOGS-007 — Implement optional-infrastructure memory adapter** — Infrastructure repository — M
  - Depends: BLOGS-006. Ref: [§4.2](technical-design.md#42-port-and-application-contracts), [§9](technical-design.md#9-risks--open-questions).
  - Deliverables: `blogrepository.Memory` with injected current AccountRepository and full port conformance.
  - DoD: detached copies, global slug reservation, staged transaction commit/rollback without recursive locks, deterministic pagination/filtering, and live owner facts. Document the cross-store snapshot limitation. Existing account/task memory implementations remain unchanged. Existing package checks pass; parity tests are BLOGS-014.
  - Commit: `feat(blogs): [BLOGS-007] add transactional memory adapter`.

### Use Case

- [ ] **BLOGS-008 — Implement blog inputs and command use cases** — Application — L
  - Depends: BLOGS-007. Ref: [§4.2](technical-design.md#42-port-and-application-contracts), [§4.3](technical-design.md#43-authorization-and-disclosure), [§5.1–5.2](technical-design.md#51-create).
  - Deliverables: BlogActor/input/view DTOs and Create/Edit/Publish/Unpublish/Delete methods.
  - DoD: current actor account validation precedes protected behavior; owner comes from actor; transactions load fresh locked state, authorize before idempotency, and persist only valid effective changes. Create retries only generated ID/slug collisions for five attempts in fresh statements/transactions. Clock/ID injection deterministic; no completion publisher. Domain/application existing suites pass.
  - Commit: `feat(blogs): [BLOGS-008] orchestrate blog commands`.

- [ ] **BLOGS-009 — Implement distinct read use cases** — Application — M
  - Depends: BLOGS-008. Ref: [§2.2](technical-design.md#22-pagination-and-admin-filter), [§4.2–4.3](technical-design.md#42-port-and-application-contracts), [§5.3](technical-design.md#53-reads-and-account-changes).
  - Deliverables: ListPublic/GetPublic/GetPrivate/ListAdmin/GetAdmin and read DTO mapping.
  - DoD: public methods take no actor and use eligible queries; protected reads validate current actor; private owner/admin reads hide deleted, admin surface reads deleted. Unknown slug/missing/hidden share BLOG_NOT_FOUND; infrastructure failures propagate. No owner-private list or public UUID fallback. Existing application checks pass.
  - Commit: `feat(blogs): [BLOGS-009] separate public private and admin reads`.

### Handler / Route

- [ ] **BLOGS-010 — Add HTTP translation, routes, catalogs and runtime wiring** — Delivery/infrastructure — L
  - Depends: BLOGS-009. Ref: [§2.1–2.2](technical-design.md#21-routes-and-security-boundaries), [§4](technical-design.md#4-component-breakdown), [§6–7](technical-design.md#6-request--response).
  - Deliverables: `delivery/http/blog.go`, Server/Router additions, runtime composition, English/Indonesian messages.
  - DoD: all ten operations registered with explicit public/protected/admin boundaries; public GET before broad protected groups, user-only POST create. Strict JSON presence/null/EOF behavior, UUID/slug distinction, page/filter parsing and exact response fields/statuses match design. Compose Postgres and memory against the same account source as auth; no config/event changes. `go test ./...`, existing architecture/smoke tests pass before documentation/test expansion.
  - Commit: `feat(blogs): [BLOGS-010] expose and compose blog routes`.

### API Contract Documentation

- [ ] **BLOGS-011 — Update API contract and endpoint documentation** — Documentation — M
  - Depends: BLOGS-010. Ref: [§2.3](technical-design.md#23-contract-documentation), [§6–7](technical-design.md#6-request--response).
  - Deliverables: `docs/api/openapi.yaml`, `docs/api/endpoints.md`, `docs/api/CHANGELOG.md`, version 1.2.0.
  - DoD: all paths/methods, schemas and envelope/error codes match implementation/design; public GET explicitly uses security: [], protected methods inherit/declare the existing Bearer scheme. Page defaults/limits/filter/order, HTML text-count rules, immutable slug and idempotency documented. Endpoint guide accurately retains current account/task authentication. YAML parses, local references resolve and path/method inventory matches router; no new contract platform required.
  - Commit: `docs(api): [BLOGS-011] document blogs contract`.

### Tests

- [ ] **BLOGS-012 — Verify domain behavior and sanitizer** — Domain tests — L
  - Depends: BLOGS-011. Ref: [§3.2](technical-design.md#32-slug-algorithm), [§4.1–4.3](technical-design.md#41-domain-model-and-html), [tactical §5.1](tactical-strategy.md#51-unit-and-application-tests).
  - Deliverables: external-package aggregate/domain service tests, named table cases and controlled time/IDs.
  - DoD: boundaries, Unicode, draft/published validation, immutable slug/timestamps, deletion/idempotency/access matrix and adversarial HTML/reparse cases pass. Rejected operations leave state unchanged. Run focused named tests immediately after edits, then `go test ./domain/... -race -v`.
  - Commit: `test(blogs): [BLOGS-012] verify domain and html policy`.

- [ ] **BLOGS-013 — Generate mocks and verify use-case orchestration** — Application tests/tooling — L
  - Depends: BLOGS-012. Ref: [§4.2–4.3](technical-design.md#42-port-and-application-contracts), [§5](technical-design.md#5-flow-sequence), [tactical §5.1](tactical-strategy.md#51-unit-and-application-tests).
  - Deliverables: register BlogRepository in `.mockery.yml`, generated mock under `test/mocks/`, external-package use-case tests using existing AccountRepository mock.
  - DoD: generation follows repo Mockery configuration, no handwritten mocks; actor lifecycle, exact denial mapping, authorized no-op, five-attempt collision handling, transaction/commit errors, read surface separation and DTO field policy are verified. Focused named tests and `go test ./application/... -race -v` pass.
  - Commit: `test(blogs): [BLOGS-013] verify use cases with generated mocks`.

- [ ] **BLOGS-014 — Verify HTTP contract and memory behavior** — Delivery/adapter tests — L
  - Depends: BLOGS-013. Ref: [§2](technical-design.md#2-api--interface-contract-changes), [§4.2–4.3](technical-design.md#42-port-and-application-contracts), [§6–7](technical-design.md#6-request--response), [tactical §5.2](tactical-strategy.md#52-http-and-contract-tests).
  - Deliverables: router/handler tests, memory transaction/visibility/pagination tests; contract inventory assertions where useful.
  - DoD: all ten operations, anonymous/invalid/revoked/owner/admin headers, 400/401/403/404/409 mapping, localized envelope, omitted/null/trailing JSON, page overflow/duplicates and admin filters covered. Memory tests prove detached copies, rollback after staged update, deletion reservation, account deactivation/reactivation visibility and tied ordering. Compare OpenAPI security/field declarations with runtime behavior. Focused test reruns and affected package suites pass.
  - Commit: `test(blogs): [BLOGS-014] verify routes and memory parity`.

- [ ] **BLOGS-015 — Verify migrations and PostgreSQL behavior** — Integration tests — L
  - Depends: BLOGS-014. Ref: [§3](technical-design.md#3-data-model--schema-changes), [tactical §5.3](tactical-strategy.md#53-postgresql-and-migrations).
  - Deliverables: integration-tagged blog migration/repository tests under `test/infrastructure/` with isolated fixture/database handling.
  - DoD: up/down/up preserves existing account/task/token data; constraints/slug reservation/indexes, eligibility joins, exact totals, tied ordering, rollback and coordinated row-lock concurrency verified. Collision retry recovers from actual constraint violations in fresh transactions. Older fixed-version migration tests remain valid; serialize schema-changing tests or isolate databases. Run focused integration tests then the integration suite on disposable live DB; record skips explicitly if unavailable and do not call live behavior verified.
  - Commit: `test(blogs): [BLOGS-015] verify migrations and postgres locking`.

- [ ] **BLOGS-016 — Add lifecycle smoke coverage and complete acceptance checks** — Smoke/regression — M
  - Depends: BLOGS-015. Ref: [§8–9](technical-design.md#8-constraints-checklist), [tactical §5.4](tactical-strategy.md#54-regression-and-final-verification), [rollout](tactical-strategy.md#6-rollout--rollback).
  - Deliverables: INFRA_OPTIONAL blog lifecycle smoke, final check evidence, capability-to-test mapping for all 24 approved rules (050–054, 060–067, 070–075, 080–084).
  - DoD: smoke covers draft/private/public publication and deletion; existing health/login/accounts/profile/task/TaskCompleted regressions pass. Run focused smoke after edits, then `go test ./...`, `make test`, `make test.arch`, `make test.smoke`, `make lint`; include isolated integration evidence from BLOGS-015. Report failures, missing tools and skips. Inspect final diff for scope and contract consistency. Do not mark unavailable required verification as passed; keep acceptance outstanding until resolved.
  - Commit: `test(blogs): [BLOGS-016] verify lifecycle and regressions`.

## Dependency Graph

```text
BLOGS-001 migration
  → BLOGS-002 → BLOGS-003 → BLOGS-004 domain
  → BLOGS-005 → BLOGS-006 → BLOGS-007 repositories
  → BLOGS-008 → BLOGS-009 use cases
  → BLOGS-010 handler / route / runtime
  → BLOGS-011 API documentation
  → BLOGS-012 → BLOGS-013 → BLOGS-014 → BLOGS-015 → BLOGS-016 tests / acceptance
```

No parallel tasks or hidden prerequisite feature work. Any discovered deviation from the verified baseline must be reconciled explicitly before the dependent task starts.
