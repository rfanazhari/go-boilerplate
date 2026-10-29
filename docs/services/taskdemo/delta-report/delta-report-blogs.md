# Delta Report: Blogs
Generated: 2026-10-29
Updated: 2026-10-29 — user decisions OQ-01–14 incorporated
Source: [flow-task-current-state.md](../flows/flow-task-current-state.md) → [blogs.md](../business/capabilities/blogs.md)

## 1. Change Type

- [x] Additive only (no breaking change)
- [ ] Breaking change (affects existing behavior)
- [ ] Mixed (partly additive, partly breaking)

Justification: Blogs adds a separate capability and does not request changes to existing task endpoints, task lifecycle rules, account contracts, or the `TaskCompleted` event. Shared router, runtime composition, error catalogs, API documentation, and migration history need additive changes. Account lifecycle changes gain an observable consequence for blog visibility, but the target does not require altering existing task behavior. Classification assumes the existing task/account contracts are preserved; the approved Blogs decisions must be applied without changing shared task/account policy.

### Baseline reconciliation and evidence

The supplied flow is a historical, pre-authentication snapshot. It describes anonymous tasks, no owner, one migration, and no protected subrouter. The checked-out source already contains authentication, accounts, and task ownership. This report compares the requested documents while using the source to establish what is actually reusable. Differences already implemented by the account capability are prerequisites already present, not net-new Blogs work. Source inspection establishes code presence, not runtime verification; no application tests were run for this documentation change.

Repository paths below are relative to the project root unless linked.

| Subject | Supplied flow snapshot | Current source evidence | Consequence for this delta |
|---|---|---|---|
| Authentication | Logging only; `Authorization` ignored | `delivery/http/auth.go`: `Auth.Middleware`, `authenticate`, `ActorFromContext`; `application/usecase/login.go`: opaque bearer token issuance and hashing | Reuse token authentication; no new login or token format is required. All protected Blogs operations require an active, non-deleted account. |
| Route groups | Health and tasks on one router | `delivery/http/router.go`: public health/login, protected routes, admin/user subrouters | Extend registration for Blogs public and protected access. |
| Ownership | No owner in task/domain/HTTP | `application/usecase/task.go`, `delivery/http/task.go`: actor input, ownership checks, `ownerId` output | Reuse the identity propagation pattern, not task-specific permissions. |
| Errors | No 401/403 mapping | `delivery/http/envelope.go`, `pkg/errs/codes.go`: unauthenticated/forbidden kinds and HTTP mappings | Reuse error infrastructure; blog-specific codes are new; disclosure now follows approved OQ-14. |
| Database | Tasks-only migration | `migrations/pgsql/20261027225508_accounts_tokens_task_owner.up.sql`: accounts, credential tokens, task owner FK | Add a later Blogs migration; do not recreate accounts or backfill task owners again. |
| API documentation | Tasks unsecured | `docs/api/openapi.yaml`: bearer security, account/login paths, task ownership | Extend the actual contract after blog API decisions; do not regenerate from the historical flow. |

The sibling [account capability](../business/capabilities/user-management-authentication-task-ownership.md) also retains stale planned/known-gap text despite the source above. Its account definitions and approved decisions supply context; its task-only restrictions do not automatically apply to Blogs. The historical flow and account capability remain unchanged; the Blogs capability now records the user-approved decisions. The report is saved in the existing service-level `delta-report/` directory because the two inputs occupy different directories and the repository already uses that location for delta reports.

## 2. What Changed

### New

- **Owned writing:** authenticated creation assigns the caller as owner; editing title/content is owner-only, including against a non-owner admin (`BR-TASKDEMO-050`–`054`). Only the non-admin `user` role may create. All management/private/admin access requires an active, non-deleted account.
- **Publication lifecycle:** new blogs start as `draft`; publication status is only `draft` or `published`. Only the owner may publish; owner or admin may unpublish (`060`–`065`, `067`). Task states and transitions cannot represent this lifecycle.
- **Soft deletion and moderation:** owner/admin may soft-delete; deletion is a separate condition from publication status. A non-owner admin can withdraw or delete, but cannot edit or publish another author's writing (`052`, `063`, `065`–`067`).
- **Public and privileged reads:** anonymous list/detail expose only eligible published blogs. Draft reads require owner/admin through private access. Deleted blogs are hidden from every non-admin, including the owner; admin reads use admin access (`070`–`074`). Public access never gains private privileges because a caller happens to be owner/admin.
- **Owner account dependency:** public list/detail must also exclude blogs whose owner is non-active or soft-deleted (`075`). Checking the reader's token does not satisfy this rule.
- **Blog validation:** title length 10–200; publishing requires content length 100–1000; specified HTML tags are permitted (`080`–`083`). Approved policy: trimmed Unicode title; sanitized text length for content; draft content may be empty but at most 1000 characters; valid published edits are immediately visible, invalid edits are rejected.
- **Title-based public lookup:** eligible public detail uses a title-derived identifier (`084`). Approved identifier: globally unique title-based slug with a server-generated short ID suffix, fixed at creation and reserved after soft delete. Normalization/fallback/collision details belong in the tripack.
- **A new collection read:** public blog listing is required even though task listing remains out of scope. Approved public list uses page/pageSize (default size 20, maximum 100), createdAt descending with ID tie-breaker, and ID/slug/title/author name/creation time fields. Full content is detail-only. Search, other public filters, and owner-private listing are deferred.

### Modified

- **Shared composition:** add blog handlers/use cases/adapters to `infrastructure/runtime/runtime.go` and route registration to `delivery/http/router.go`.
- **Shared documentation and error catalogs:** extend `docs/api/openapi.yaml`, error code definitions, and English/Indonesian translations for approved blog operations and errors.
- **Schema history and verification coverage:** append blog migrations and related tests/mocks; reuse existing migration tooling.
- **Account lifecycle integration:** deactivation/deletion must affect public blog visibility. Approved OQ-08 requires current account eligibility at public read time without changing blog publication state. Reactivating an eligible owner makes their non-deleted published blogs visible again. No account-to-blog write cascade is required.
- **Existing task behavior:** None required by Blogs. Existing source already protects tasks; that change predates this delta.

### Removed

None. The target describes another module, so omission of task operations, due dates, overdue calculation, and completion events is not a request to remove them. Blog soft delete does not authorize task deletion or account hard deletion.

## 3. Impact Analysis

| Layer | Impact | Notes |
|-------|--------|-------|
| Transport/Handler (`delivery/`) | New + Modified | New blog request/response translation and access surfaces; extend the shared router. Reuse envelope, localization, logging, identity extraction. Public list/detail must remain accessible without login. HTTP paths, verbs, payloads, and operation success codes are not yet specified. |
| Use Case (`application/`) | New; existing account writes unchanged | Blog orchestration, actor-to-owner assignment, permissions, read policies, publication/deletion flows. Account writes remain unchanged under the approved read-time visibility policy. Do not invoke `TaskUseCase` to implement Blogs. |
| Domain (`domain/`) | New; existing account concepts reused | Blog lifecycle, ownership/moderation policy, title/content constraints, and separate deletion condition. Reuse role/account-status vocabulary and clock. Keep domain rules independent of HTTP, SQL, and delivery context types. |
| Service/Integration | Reused as-is + New HTML processing | Existing authentication/token/password services suffice for identity. HTML processing requires the approved server-side sanitization behavior; this does not imply a separate root layer or service. No blog message publisher, queue, consumer, notification, or outbox is required by the target or the approved initial scope. |
| Repository (`domain/` ports, `infrastructure/` adapters) | New; existing account lookup reused | Blog persistence, public list and title-derived lookup, authorized private/admin reads. Public reads require current owner eligibility; account lookup is available, but there is no existing combined blog/account query. Both list and detail must apply all visibility conditions. |
| Database/Migrations | New | Blog storage with owner relation, publication state, deletion condition, content, and approved public identifier requirements. Use new ordered up/down migrations. Global slug uniqueness is required; index details follow the approved listing and lookup behavior; no task-row backfill is needed for an initially empty module. |
| Contract/OpenAPI | New + Modified | Document public, private, and admin capabilities once their API contract is settled. Existing OpenAPI has global bearer security: anonymous blog operations need explicit `security: []`, as health/login already do. No protobuf change is indicated. |
| Runtime/Configuration | Modified; configuration reused | Wire blog dependencies into existing runtime and DB pool. Preserve environment-only configuration and optional-infrastructure operation. New memory behavior is needed if Blogs is supported in `INFRA_OPTIONAL`; no new external dependency/configuration is currently required. |
| Tests/Tooling | New + extended coverage | Add blog behavior and visibility coverage, generated repository mocks, adapter/migration tests, and router regressions. Existing architecture checks and Go/test conventions remain applicable. |

### Existing behavior affected and implementation implications

| Existing behavior/component | Required or conditional effect | Implementation implication |
|---|---|---|
| Protected routing | New anonymous blog reads coexist with protected management | Register public access so it does not pass through required-auth middleware. Check route matching/fallthrough because current protected subrouters use broad `NewRoute()` registration. Keep existing health/login and task/account route behavior covered. |
| Active-account authentication | Middleware rejects non-active/deleted accounts for every protected request and may delete their tokens | Direct reuse matches approved OQ-02: deny all protected Blogs operations for those accounts. Anonymous public reads remain available. No shared-auth exception is needed. |
| Account deactivate/soft-delete | Existing account writes revoke tokens; new blog readers must also stop seeing the author's published work | Anonymous readers do not run auth, so token revocation alone is insufficient. Approved OQ-08 requires read-time account eligibility. Avoid visibility based only on account state captured when publishing. |
| Account reactivation | Existing source supports reactivation | Approved OQ-08: existing non-deleted published blogs reappear when the owner is active and not soft-deleted again. No republish or stored publication-state change is required. |
| Task owner/admin permissions | No required change | Tasks hide unauthorized ownership access as 404 and let admin read any task. Blogs has a different moderation matrix; reuse patterns without copying that policy or overriding the approved Blogs 401/403/404 disclosure policy. |
| Task validation/lifecycle | No required change | Task title remains trimmed 1–100 Unicode code points; task `done` remains final. Do not widen a shared validator or reuse `todo/in_progress/done` for Blogs. |
| Shared HTTP errors | Existing mapping/envelope can stay intact; codes/messages are additive | Preserve `status`, `data: {}`, and `error.message/code/errors`. Use approved 400 validation, 401 invalid identity, 403 forbidden operation on a visible resource, 404 absent/hidden resource, and 409 state conflict. Publish/unpublish are idempotent; repeated admin delete succeeds, repeated non-admin owner delete returns 404. Blog-specific code names belong in the tripack. |
| Migration runner/deployment | Existing `cmd/migrate` applies later up migrations | Preserve existing migration files and account/task data. New down migration must remove only blog-owned objects before any prerequisite account rollback. Current account-migration test explicitly targets versions, so blog migrations need their own coverage rather than an assumption that the older test covers them. |
| Task completion + AMQP | No required change | Preserve `taskId`/`completedAt` payload and task completion behavior. Blog “publish” means changing publication state, not invoking `CompletionPublisher`. Do not copy the task's synchronous AMQP-before-commit workflow into Blogs. |

Conceptually, public eligibility is the conjunction of **published**, **blog not soft-deleted**, **owner active**, and **owner not soft-deleted**. Owner/admin identity does not override this conjunction on a public surface. Deleted-blog access is an admin-read exception on an admin surface, not a restore/edit/publish permission. The report identifies these obligations without selecting joins, endpoint layouts, caching, or transaction design. Any future caching must preserve the same visibility on list and detail when the blog or owner changes.

## 4. Reuse Map

“Reused as-is” describes existing logic usable for Blogs. “Needs wrapping” means the mechanism/pattern remains useful but needs blog-specific inputs, policy, or integration. “Replace” means use a new Blogs counterpart rather than copying task-specific behavior; existing task code stays in place.

| Component | Status | Notes |
|-----------|--------|-------|
| `Auth.Middleware`, `Auth.authenticate`, `LoginUseCase`, token repositories | Reused as-is | Opaque bearer credentials, current account lookup, status/deletion rejection, revocation infrastructure already exist. No JWT/new credential store is requested. |
| `ActorFromContext`, `Actor` | Needs wrapping | Extract identity at the HTTP edge and map it to application input. `dto.TaskActor` is task-specific; domain/application must not import the delivery actor type. Never take ownership from client-supplied owner fields. |
| `RequireRole` | Needs wrapping | Useful for strictly admin reads/moderation surfaces. It cannot express owner-or-admin, nor owner-only editing/publishing. Approved OQ-01 permits reusing the `RoleUser` create gate: admins cannot create blogs. |
| Four root layers and current handler → use case → aggregate/port → adapter pattern | Needs wrapping | Reuse dependency direction and composition style with Blogs types. Ports currently live under `domain/repository`; adapters remain in infrastructure. No additional root layer is needed. |
| `writeSuccess`, `writeError`, `Envelope`, error kinds, localization bundle | Reused as-is | Existing mappings include 400/401/403/404/409/500/503. Extend message/code catalogs as necessary; existing task codes must retain their meaning. |
| `decodeJSON` | Reused as-is for JSON translation | Existing unknown-field/malformed-input handling is available; blog-specific required/optional fields and validation are new. It does not sanitize HTML. |
| `pathID`, timestamp formatting helpers | Needs wrapping | UUID parsing is reusable for the approved private/admin UUID lookup. It cannot parse the required title-derived public identifier. Time helpers are usable if the eventual blog view exposes timestamps. |
| Task aggregate, status enum, title validator, DTOs and `TaskUseCase` | Replace for Blogs | New blog equivalents are needed: different states, title bounds, content, deletion, and permissions. Do not alter task equivalents. |
| `TaskUseCase.authorizeGet` / `authorizeMutate` | Replace for Blogs | Task permissions do not implement public visibility, deleted-blog exceptions, or owner/admin moderation. |
| `AccountRepository.GetByID`, account status/deletion accessors, role enum | Reused as-is | Current account records provide eligibility facts, including deleted state. Efficient public listing still needs new query behavior; calling the repository alone does not supply a collection policy. |
| Postgres pool, raw `$1` SQL style, plain adapter structs, `pgxtx.With/From` | Needs wrapping | Reuse infrastructure mechanisms in a blog adapter. Transaction context plumbing is available; atomicity/concurrency for approved blog transitions still needs verification. |
| Task/account Postgres and memory repositories | Replace for Blogs | Reuse implementation patterns, not interfaces/data stores. New blog storage and read methods are required. The documented task memory transaction has no rollback; copying that behavior does not establish transactional correctness. |
| `cmd/migrate`, ordered `migrations/pgsql/*.up.sql` / `*.down.sql` | Reused as-is | Append blog migration pair(s) after current prerequisites. Existing account seed/task owner backfill must not be repeated. |
| Runtime, logging middleware, optional-infrastructure composition | Needs wrapping | Add dependencies and routes while keeping established startup behavior. Existing log masking remains available. |
| Clock/test clock, testify table-driven tests, Mockery, architecture checks | Reused as-is | Add new behavior cases and generate new port mocks through `.mockery.yml`; do not handwrite mocks. |
| `CompletionPublisher`, Rabbit publisher | Reused as-is for tasks only | Not a Blogs dependency under the requested capability. |

## 5. Net-New Inventory

- [ ] Blog domain model and publication/deletion rules — Layer: Domain — Reason: task aggregates do not model drafts, HTML content, or soft-deleted writing (`060`–`066`, `080`–`083`).
- [ ] Blog ownership/moderation permissions — Layer: Domain/Application — Reason: edit/publish owner-only; unpublish/delete owner-or-admin; creation is restricted to non-admin role user (`050`–`054`, `062`–`067`).
- [ ] Blog command orchestration and application inputs/views — Layer: Application — Reason: create, edit, publish, unpublish, soft delete are new operations with new validation boundaries.
- [ ] Public, private, and admin read policies — Layer: Application/Repository — Reason: anonymous visibility, draft privacy, admin-only deleted reads, and owner-account eligibility are new (`070`–`075`). Public slug access, UUID private/admin access, and separate admin deleted-blog list/detail are approved; exact HTTP registration is tripack work.
- [ ] Title/content validation and agreed HTML handling — Layer: Domain and supporting implementation — Reason: blog bounds differ from tasks; HTML allowlist and sanitization-before-length-validation are approved (`080`–`083`).
- [ ] Title-derived public identifier behavior — Layer: Domain/Application/Repository/Delivery — Reason: task UUID lookup cannot satisfy `084`; global uniqueness, fixed slug, and no reuse are approved; normalization, fallback, suffix sizing, and collision handling are tripack details.
- [ ] Blog repository port, Postgres adapter, and optional-infrastructure memory adapter — Layer: Domain/Infrastructure — Reason: no blog persistence, public listing, or eligible-public-detail query exists.
- [ ] Blog schema and ordered up/down migrations — Layer: Infrastructure — Reason: owner-linked blog records and publication/deletion data are new. Identifier constraints follow approved decisions.
- [ ] Blog handlers, JSON views, route registrations, and OpenAPI additions — Layer: Delivery/Contract — Reason: no Blogs API contract exists yet; the capability defines business behavior rather than endpoints.
- [ ] Blog error codes/messages where existing generic errors do not suffice — Layer: Shared errors/Delivery — Reason: validation, identifier conflicts, and visibility denials need codes/messages following approved OQ-14, including English/Indonesian messages.
- [ ] Blog runtime composition and generated mocks — Layer: Infrastructure/Test tooling — Reason: new dependencies must be wired in production and optional-infrastructure execution.
- [ ] Blog verification coverage — Layer: Tests — Reason: new permissions and visibility can leak drafts/deleted posts if tested only as ordinary get-by-ID operations. Cover user-only creation (admin denied), public list/detail equivalence, admin denied edit/publish of another author's blog, deleted owner denial/admin read, account lifecycle visibility, and approved validation/identifier edge cases.
- [ ] Account-aware public visibility queries — Layer: Application/Infrastructure — Reason: approved OQ-08 requires current owner eligibility on list/detail; retain stored publication status and make eligible published blogs visible on owner reactivation without account-write cascades.

## 6. Open Questions

None blocking tripack generation. The user approved OQ-02–14 and replaced the OQ-01 proposal on 2026-10-29: **admins cannot create blogs; authorship is restricted to non-admin role user**. The authoritative decision record is the [Blogs capability, Approved decisions](../business/capabilities/blogs.md#approved-decisions--2026-10-29).

| Decisions | Approved outcome / implementation implication | Blocking |
|---|---|---|
| OQ-01–02 | User-role-only creation; active/non-deleted identity required for all protected management/private/admin access. Reuse existing auth and role gate. Public reads remain anonymous. | No |
| OQ-03–04 | Draft content may be empty, maximum 1000; published requires 100–1000. Title trimmed, Unicode code points. Content uses sanitized text, decoded entities, normalized whitespace and block/br separators, then trim and Unicode count. | No |
| OQ-05 | Closed p/br/h1–h5/a allowlist; only href on a, absolute HTTP/HTTPS. Sanitize server-side before length validation/storage and return sanitized content. | No |
| OQ-06–07 | Unique title-based slug with short server ID suffix at create; unchanged on rename and never reused after deletion. Public detail uses slug only; management/private/admin use UUID. | No |
| OQ-08 | Keep publication status; evaluate owner eligibility at read time. Eligible non-deleted published blogs reappear on reactivation. | No |
| OQ-09 | Admin list/detail with deletion filter; deleted blogs are read-only. No restore/edit/republish of deleted blogs, hard delete, or additional retention policy in initial scope. | No |
| OQ-10 | Owner may republish after admin unpublish without approval. Reasons/notifications excluded. Admin soft delete prevents owner republishing. | No |
| OQ-11–12 | Authorized publish/unpublish are idempotent for non-deleted blogs. Repeat admin delete succeeds; repeat non-admin owner delete is 404. Valid owner edits to published content are immediately public; invalid edits leave data/state unchanged. | No |
| OQ-13 | Public page/pageSize, default size 20, maximum 100; createdAt DESC with ID tie-breaker. List fields: ID, slug, title, author name, creation time; full content on detail. Search/other public filters and owner-private list deferred. Admin list remains required. | No |
| OQ-14 | 401 invalid identity; 403 forbidden operation on a visible blog; 404 absent/hidden blog or unknown slug; 400 validation; 409 state conflict. | No |

### Details to define in the tripack

- Specify HTTP paths, verbs, request/response structures, success status codes, and the admin deletion filter, preserving the shared envelope and the approved public/private/admin separation.
- Specify slug normalization, Unicode handling, empty-base fallback, short suffix format/length, and collision handling. These are technical details under the approved unique, fixed title-based slug policy, not approval for a different identifier lifecycle.
- Specify exact sanitizer behavior for disallowed/malformed HTML consistent with the approved allowlist and processing order; choose suitable implementation dependencies during design.
- Specify pagination parameter validation, default first page, ID tie-break direction, and field names consistently across contract and tests.

These details may be formulated during tripack generation. They are not unresolved business blockers and no implementation has been started by this report update.

## 7. Recommendation

Proceed to generate the implementation tripack. Blogs remains an additive module using existing four-layer architecture, bearer authentication, user-role create gate, account concepts, error envelope/localization, Postgres pool, and migration tooling. All fourteen business questions have approved answers, with user-only authorship overriding the earlier suggestion that admins could author. Build new Blogs lifecycle, ownership/moderation policies, sanitization/validation, fixed unique slug lookup, admin deleted-blog reads, and public list/detail with current owner eligibility; no account-write cascade or new messaging integration is needed. Define remaining API/normalization details within the tripack and preserve existing task/account contracts and task completion events. Capability status stays planned and implementation evidence remains empty until code is implemented and verified.
