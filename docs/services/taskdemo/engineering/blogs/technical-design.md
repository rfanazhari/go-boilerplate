# Technical Design: Blogs

## 1. Overview

Mode A: a new, additive feature. This is a design, not evidence of implementation. The baseline already implements accounts, opaque Bearer authentication, and task ownership; the historical task flow is not the current implementation baseline.

Sources: [Blogs capability](../../business/capabilities/blogs.md), [approved OQ-01–14](../../business/capabilities/blogs.md#approved-decisions--2026-10-29), and [delta report](../../delta-report/delta-report-blogs.md). Root `CLAUDE.md` is absent; root `AGENTS.md`, current code, and existing engineering documents supply project context. Output follows the existing `engineering/<capability>/` convention.

Verified stack: Go 1.26.0, gorilla/mux, pgx/v5, PostgreSQL, golang-migrate timestamped SQL pairs, go-i18n English/Indonesian catalogs, testify, and Mockery. Four root layers retain inward dependencies. Repository ports currently live in `domain/repository`; reuse that convention. Blogs does not depend on the task completion publisher.

### Goals

- User-role accounts author drafts, edit, publish, unpublish, and soft-delete their own blogs.
- Admins read and moderate blogs through the approved permission matrix.
- Anonymous readers list eligible blogs and open detail through a stable title-derived slug.
- Public eligibility reflects the current owner account state without changing stored publication status.
- Specify schema, HTML policy, concurrency, HTTP contracts, and sequential implementation tasks.

### Non-Goals

Owner-private list, search, other public filters, restore, hard delete, retention automation, moderation reasons/notifications, editorial approval, slug redirects, and blog events. Existing task/account contracts and task business rules remain intact. This delivery writes documents only.

## 2. API / Interface Contract Changes

### 2.1 Routes and security boundaries

| Method | Path | Access | Success / data |
|---|---|---|---|
| GET | `/blogs` | Anonymous | 200 / public page |
| GET | `/blogs/{slug}` | Anonymous | 200 / public detail |
| POST | `/blogs` | Authenticated role user only | 201 / managed detail |
| GET | `/private/blogs/{id}` | Authenticated owner; admin reads live blogs too | 200 / managed detail |
| PATCH | `/private/blogs/{id}` | Authenticated owner only | 200 / managed detail |
| POST | `/private/blogs/{id}/publish` | Authenticated owner only | 200 / managed detail |
| POST | `/private/blogs/{id}/unpublish` | Authenticated owner or admin | 200 / managed detail |
| DELETE | `/private/blogs/{id}` | Authenticated owner or admin | 200 / `{}` |
| GET | `/admin/blogs` | Authenticated admin only | 200 / admin page |
| GET | `/admin/blogs/{id}` | Authenticated admin only | 200 / admin detail, including deleted |

Protected access requires a current active, non-deleted account via existing `Auth.Middleware`. Admin creation is 403. Map `ActorFromContext` into a new application `BlogActor`, never into `TaskActor`. Authorization also runs in use cases; middleware alone is insufficient for ownership decisions.

Register the two public GET routes directly on the root router before the broad protected subrouter. Register POST `/blogs` within the existing user group. Register `/private/blogs/...` within protected, and `/admin/blogs/...` within the existing admin group. Public and protected access differ by method/path, never by optional authentication on one handler. Public GET ignores even invalid/revoked Bearer headers and applies public visibility uniformly. There is no public UUID lookup or slug-to-UUID fallback (D2, D3).

### 2.2 Pagination and admin filter

Both lists accept `page` (default 1) and `pageSize` (default 20, inclusive range 1–100). Each supplied value must be one nonempty ASCII decimal integer with no sign or whitespace, fit signed 64-bit, and be positive. Reject duplicate keys, unknown query keys, and overflow in `(page - 1) * pageSize` with 400 `INVALID_BLOG_PAGINATION`. Public accepts no other keys. Admin additionally accepts one `deletion=active|deleted|all`, default `active`; invalid/duplicate deletion values return 400 `INVALID_BLOG_FILTER`.

`active` here means `blogs.deleted_at IS NULL`, not owner-account status. Admin list includes draft/published blogs regardless of owner eligibility. `deleted` selects only deleted rows; `all` includes both. Admin detail accepts any existing UUID without requiring a query filter.

Order is `created_at DESC, id DESC` everywhere. Filtering occurs before counting and pagination. Return `{items: [], page, pageSize, total}` with integer metadata and exact total for that request. Out-of-range pages return empty items and the filtered total. PostgreSQL count and items use one read-only repeatable-read transaction, including for empty/out-of-range pages. Memory captures blog records and owner facts for that read. Offset pagination has deterministic ties but can shift between requests when data changes; there is no cross-request snapshot guarantee (D4).

### 2.3 Contract documentation

A dedicated task updates `docs/api/openapi.yaml`, `docs/api/endpoints.md`, and `docs/api/CHANGELOG.md` to API version 1.2.0. Public operations explicitly set `security: []`; all protected operations use the existing Bearer scheme. Document every route, role, visibility rule, pagination/filter, nullable field, HTML length semantics, error, and idempotent operation. Keep current task/account contracts; the endpoint guide currently lags OpenAPI and must accurately describe existing authentication when extended.

## 3. Data Model & Schema Changes

### 3.1 Blog storage

Append `migrations/pgsql/20261029000100_create_blogs.up.sql` and matching `.down.sql`. This version is later than current `20261027225508_accounts_tokens_task_owner`; verify it remains unused when implementing.

| Column | PostgreSQL type / constraints | Meaning |
|---|---|---|
| `id` | UUID primary key | Server-generated UUID v4 |
| `owner_id` | UUID NOT NULL, FK accounts(id), no cascade | Immutable author identity |
| `slug` | TEXT NOT NULL, global unique constraint `blogs_slug_key` | Immutable, reserved even after deletion |
| `title` | TEXT NOT NULL; char_length between 10 and 200 | Trimmed domain-validated title |
| `content` | TEXT NOT NULL, default empty string | Sanitized HTML fragment |
| `status` | TEXT NOT NULL; CHECK draft or published | Separate from deletion |
| `deleted_at` | TIMESTAMPTZ nullable | First successful soft-delete instant |
| `created_at` | TIMESTAMPTZ NOT NULL | Creation instant |
| `updated_at` | TIMESTAMPTZ NOT NULL | Last effective change |

Application/domain validation remains authoritative for trimming, normalized text length, sanitized HTML, and role permissions; SQL character checks are a secondary guard, not HTML validation. No publishedAt, revision history, deletion actor, or optimistic version column is required.

Indexes: global unique slug including deleted rows; `(owner_id)` for the FK/owner lookups; partial `(created_at DESC, id DESC) WHERE status = 'published' AND deleted_at IS NULL` for public listing; full `(created_at DESC, id DESC)` for admin ordering. Owner eligibility remains a join predicate, not a partial index condition across tables. Existing account PK supports the join.

No backfill is needed: the new table starts empty. Existing migration files, seeded accounts, tokens, and tasks are untouched. Down drops only blogs and its indexes/constraints; it destroys blog data and must never run as an automatic application rollback. Drop blogs before rolling back the prerequisite accounts migration.

### 3.2 Slug algorithm

At creation, trim/validate the title and generate UUID v4. Normalize the title with Unicode NFKD (`golang.org/x/text/unicode/norm`, already a direct dependency), remove combining marks, lowercase ASCII A–Z, retain ASCII a–z and 0–9, replace every other run with `-`, then trim `-`. Limit the base to 80 ASCII characters and trim a trailing hyphen after truncation. If empty, use `blog`. Non-Latin titles remain valid and use the fallback if they have no surviving base; transliteration is outside initial scope.

Append `-` plus the first 12 lowercase hex characters of the UUID after removing hyphens. Maximum length is 93 characters. Example: `Catatan Go` with UUID prefix `123456789abc` becomes `catatan-go-123456789abc`. Same titles are allowed. The suffix is an identifier aid, not a secret or an authorization control.

Use the global unique constraint as the final concurrency guard. If insert violates `blogs_slug_key` (or the generated ID primary key), discard that candidate, generate a fresh UUID/slug, and retry the entire insert in a fresh transaction/autocommit statement, at most five total attempts. Never retry other SQL errors. Exhaustion returns 409 `BLOG_IDENTIFIER_CONFLICT`; nothing is created. Do not retry a failed statement inside an already-aborted PostgreSQL transaction. Inject the ID function for deterministic collision tests.

Rename never changes the slug. Deletion never removes the record or its uniqueness reservation. Lookup uses exact stored lowercase slug; uppercase/malformed/unknown slugs yield 404 rather than redirects. UUIDs are used exclusively on management/private/admin surfaces (D3).

### 3.3 Atomic persistence and public visibility

Separate insert from update. Updates must not upsert or alter owner, ID, slug, or createdAt. Every mutation runs a repository transaction, loads the blog with `SELECT ... FOR UPDATE`, evaluates visibility and permissions, validates a candidate change, then updates mutable fields and commits. Competing edit/publish/unpublish/delete calls serialize on the blog row. A failed authorization, validation, update, or callback rolls back. Return success only after commit; repeated no-op operations preserve updatedAt/deletedAt.

Use `pgxtx.With/From` from the existing account adapter pattern, not the task repository's private transaction context. Keep transaction mechanics in infrastructure; application calls the port. A simple read uses no row lock. Auth checks current actor eligibility at request authentication; an account change concurrent with an already-authenticated request may overlap that request. Public visibility is determined from the database snapshot at read time; a later read after account-change commit must reflect it.

Public list, count, and detail share this exact predicate: blog `status = 'published'`, blog `deleted_at IS NULL`, joined owner `status = 'active'`, owner `deleted_at IS NULL`. Obtain current author name in the same join. No per-blog SQL account lookup, cached publication eligibility, account-write cascade, or background sync. Reactivation restores eligible published blogs automatically without changing their publication state (D5).

## 4. Component Breakdown

| Layer / component | Status | Placement and justification |
|---|---|---|
| `domain/aggregate/blog.go`, `domain/enum/blog_status.go` | New | Blog lifecycle, immutable ownership/slug, validation and timestamps, independent of Task |
| `domain/service/blog_content.go`, `blog_slug.go`, `blog_access.go` | New | Pure content normalization, slug policy, and permissions/visibility facts; business rules stay inward |
| `domain/repository/blog.go` | New | Blog persistence/read port and named query/projection structs; no application DTO imports |
| `application/dto/blog.go`, `application/usecase/blog.go` | New | Actor/input/view types and orchestration; existing TaskUseCase unchanged |
| `infrastructure/repository/blogrepository/` | New | Plain `Postgres{Pool}` and `Memory{Accounts}` adapters with public fields; no infrastructure constructors |
| `delivery/http/blog.go` | New | Decode, actor translation, query/UUID parsing and view serialization |
| `delivery/http/router.go` | Modified | Add Blog handler and explicit route boundaries; retain existing route behavior |
| `infrastructure/runtime/runtime.go` | Modified | Compose Blogs in Postgres and INFRA_OPTIONAL memory modes using the same account source as authentication |
| `pkg/errs/codes.go`, `translations/errors.en.toml`, `errors.id.toml` | Modified | Add blog codes/messages; retain shared envelope/status mapping |
| `.mockery.yml`, `test/mocks/` | Modified / generated | Register BlogRepository and generate mocks, never handwrite |
| `docs/api/*`, new migration pair, new tests | Modified / New | Contract, storage and verification deliverables |
| Auth, roles/account status, clock, envelope, migration runner | Reused as-is | Mechanisms already present and appropriate |
| Account/task use cases, AMQP publisher, deployment config | Reused as-is | Blogs adds composition; no new account writes, events or environment settings |

### 4.1 Domain model and HTML

Aggregate holds ID, ownerID, slug, title, content, status, createdAt, updatedAt, deletedAt, with private state/accessors like existing aggregates. `NewBlog` always creates draft. `RestoreBlog` rehydrates persisted state. Validate candidate title/content before assigning any changed fields. Inject current time from the use case; normalize UTC to milliseconds like existing aggregates. No-op publish/unpublish/repeat-admin-delete does not change timestamps. Effective edit, transition, or first deletion updates updatedAt. Deleted content is immutable.

Pure content processing belongs to `domain/service/blog_content.go`. Use the already-resolved `golang.org/x/net/html` parser (currently an indirect module; promote to direct when imported). It is a parser, not a sanitizer: implement and test an explicit rebuild policy over a parsed body-context fragment, then serialize only rebuilt nodes. Local module documentation/source for v0.60.0 was inspected; no new package version is selected here.

Processing contract:

1. Reject invalid UTF-8 input. Parse the fragment as HTML; malformed markup follows parser repair behavior. Parser failure is 400 `INVALID_BLOG_CONTENT`.
2. Keep only HTML-namespace `p`, `br`, `h1`–`h5`, `a` elements and text. Remove comments/doctypes. Drop whole `script`, `style`, `iframe`, `object`, `embed`, `template`, `noscript`, `svg`, and `math` subtrees; discard foreign-namespace subtrees. Unwrap other unsupported elements while retaining sanitized descendants. Insert text separators at removed block boundaries (including div, section, article, ul/ol/li, table/tr/td, blockquote, pre, h6) so adjacent words do not merge.
3. Discard every attribute except a single unnamespaced `href` on `a`. Decode through the parser once; trim the URL and allow only an absolute HTTP/HTTPS URL with a nonempty host, no control characters, and no userinfo. Relative/protocol-relative/other schemes or ambiguous duplicate href attributes lose href; keep safe anchor text. Strip style, class, target, event handlers and every other attribute. Escape all serialized text and attribute values; never concatenate raw source markup.
4. Render sanitized HTML for storage/response. From the sanitized node tree, extract decoded text, adding separators before/after p/h1–h5 and at br (and retained separators from removed blocks). Collapse Unicode whitespace runs to one ASCII space and trim. Count Unicode code points in this text. Do not decode entities a second time (`&amp;lt;` is visible `&lt;`, not `<`).
5. Trim title and require 10–200 code points. Draft content may be empty and must be at most 1000 normalized text code points; published content must be 100–1000 inclusive. Publish revalidates stored content. Editing published content retains published status only if the entire candidate remains valid. Unsupported HTML is sanitized, not rejected solely for containing unsupported tags. Stored/returned content is always sanitized HTML.

Route all aggregate creation/edit/publish paths through the same policy; delivery must not be the only enforcement point. This narrow sanitizer needs adversarial and reparse tests; parser use alone does not establish safety. Title is plain text and consumers must render it as text, not HTML (D6).

### 4.2 Port and application contracts

`BlogRepository` operations: `Create`, `GetByID` (includes deleted for policy evaluation), `GetForUpdate` (transaction-only), `Update`, `RunInTransaction`, `GetPublicBySlug`, `ListPublic`, and `ListAdmin`. Use named public-field structs under `domain/repository` for page/filter/projection results, including a read projection with current authorName. Do not return application DTOs from the port. `GetPublicBySlug` returns not-found for any visibility failure, not a blog that application must hide afterwards.

`BlogUseCase` methods: Create, Edit, Publish, Unpublish, Delete, GetPrivate, GetAdmin, ListAdmin, GetPublic, ListPublic. Dependencies: BlogRepository, AccountRepository for actor validation/owner visibility facts, Clock, and injectable UUID function. Public SQL reads enforce eligibility in the adapter; domain policy expresses the same rule for command disclosure and memory parity. Validate actor identity/role/current account in protected use cases using AccountRepository so direct calls cannot bypass auth. Anonymous methods take no actor. Use validating `NewX` only for domain/use-case input; persistence/query structs are plain structs.

Memory uses mutex-protected detached aggregate copies, global slug uniqueness, and a real staged transaction: hold the mutation lock, work against a private cloned map via transaction context, publish it only on callback success. Transaction-aware methods must not relock the mutex. Discard staged changes on error/panic and unlock reliably. Public reads obtain current owner facts from the same AccountRepository instance runtime uses, filter before sorting/counting/paging, and never hold a blog lock while making account calls. Capture blog copies under lock then owner facts; optional memory mode does not promise a cross-store database snapshot under simultaneous account changes. Later reads must reflect completed changes. Production PostgreSQL is authoritative for transaction/isolation guarantees.

### 4.3 Authorization and disclosure

Apply in this order: authenticate protected actor; enforce route-wide role gates; load resource; determine whether resource may be seen; check operation permission; validate content/state; persist. Resource visibility is not inferred merely from UUID knowledge.

| Blog / actor | Read | Edit / publish | Unpublish / delete |
|---|---|---|---|
| Live, owner | Private yes; public only if eligible | Yes | Yes |
| Live, admin non-owner | Private/admin yes; public only if eligible | 403 | Yes |
| Live, other user | Public only if eligible; private 404 | 403 if publicly eligible, otherwise 404 | Same denial |
| Deleted, owner/other non-admin | 404 on every resource surface | 404 | 404, including repeated owner delete |
| Deleted, admin | Admin list/detail yes; private detail 404 | 409 BLOG_DELETED | Unpublish 409; delete 200 unchanged |
| Missing | 404 | 404 | 404 |

All protected unauthenticated/inactive/deleted callers get 401 first. Public endpoints always return 404 for hidden detail even when a header belongs to the owner/admin. An admin can only read deleted content on the admin surface; mutation errors must not include deleted content. An admin's forbidden action on a live draft returns 403 because admin is entitled to read it. Owner-private GET does not serve as an alternate public UUID lookup for other users. Admins cannot author new blogs; if historical role changes ever produce an admin owner, owner-only permissions still mean owner identity, not a blanket prohibition on admin-owned records (current API has no role-change feature).

## 5. Flow Sequence

### 5.1 Create

1. Protected user gate and use-case actor check require active non-deleted role user.
2. Validate title, sanitize content and validate draft limit; obtain server time and UUID.
3. Generate fixed slug and construct draft owned by actor; insert with bounded collision retry (§3.2).
4. Return managed detail only after persistence, HTTP 201. No completion publisher call.

### 5.2 Edit / publication / moderation

1. Authenticate, parse UUID and supported input, then enter repository transaction.
2. Lock/load current blog; apply §4.3 disclosure/permissions. Resolve owner eligibility only when needed for non-owner public visibility; unexpected lookup failures are internal errors, not permission grants.
3. Edit a validated candidate, publish after publication validation, unpublish to draft, or set deletedAt once. Check authorization before idempotency.
4. Persist effective changes only; commit, then return 200. Admin no-op delete returns empty success; owner repeat delete is hidden/404. Admin unpublish may be followed by owner republish without approval.

### 5.3 Reads and account changes

1. Public GET has no auth middleware. Validate query or exact slug, then run the eligible blog/account join.
2. List counts/pages eligible rows in one snapshot; detail returns only sanitized stored content and current author name.
3. Private/admin reads authenticate and apply their surface policy. Admin filter changes only deletion selection, not publication/owner constraints on public routes.
4. Account deactivate/delete requires no blog write. Subsequent public reads omit its posts; reactivation reveals non-deleted published posts again.

## 6. Request / Response

All responses retain `status` and `data`. Error `data` is `{}` with `error.message`, `error.code`, `error.errors` always present. Field errors retain existing `{message, code}` shape. Dates use existing RFC3339 millisecond UTC formatting; nullable deletedAt is present as null for live records.

Create body: required string `title`; optional string `content` defaults to empty. PATCH: optional string `title` and/or `content`, at least one present. Explicit null, wrong types, empty patch, unknown fields, and client owner/id/slug/status/timestamp fields are 400. Presence must distinguish omitted from null; pointer-to-string alone does not do this. Empty content clears a draft but fails a published edit. Edit does not accept a status change.

Publish/unpublish/delete accept no body or `{}`; reject null, arrays, nonempty objects and malformed/trailing JSON as 400. Use existing JSON helper as a pattern, but explicitly verify one complete JSON value and EOF for blog inputs; existing `decodeJSON` returns nil for EOF and uses `More()`, which is not a sufficient presence/trailing-value contract by itself. Keep task input behavior unchanged.

Public list item fields are exactly `id`, `slug`, `title`, `authorName`, `createdAt`; no full content, account credentials, or deletion state. Public detail adds `content` to these fields. Managed detail has `id`, `ownerId`, `slug`, `title`, `content`, `status`, `createdAt`, `updatedAt`, `deletedAt`. Admin detail uses managed detail plus `authorName`. Admin list items contain public summary fields plus `ownerId`, `status`, `updatedAt`, `deletedAt`, without content.

Example public list envelope:

```json
{"status":"success","data":{"items":[],"page":1,"pageSize":20,"total":0}}
```

Example hidden detail (message is localized):

```json
{"status":"error","data":{},"error":{"message":"Blog not found","code":"BLOG_NOT_FOUND","errors":[{"message":"Blog not found","code":"BLOG_NOT_FOUND"}]}}
```

## 7. Error Handling

| HTTP | Code | Trigger |
|---|---|---|
| 400 | `INVALID_BLOG_TITLE` | Trimmed title outside 10–200 code points |
| 400 | `INVALID_BLOG_CONTENT` | Bad UTF-8/parser error or content outside applicable bounds |
| 400 | `INVALID_BLOG_PAGINATION` / `INVALID_BLOG_FILTER` | Query validation in §2.2 |
| 400 | Existing `MALFORMED_REQUEST`, `VALIDATION_FAILED`, `INVALID_IDENTIFIER` | JSON shape/required input or malformed private/admin UUID |
| 401 | Existing `UNAUTHENTICATED` | Missing/invalid/revoked identity, inactive/deleted caller |
| 403 | Existing `FORBIDDEN` | Admin create, non-admin admin-route access, forbidden operation on visible blog |
| 404 | `BLOG_NOT_FOUND` | Missing/hidden blog or unknown/noncanonical public slug |
| 409 | `BLOG_DELETED` | Admin attempts edit/publish/unpublish on deleted blog |
| 409 | `BLOG_IDENTIFIER_CONFLICT` | Five generated identifier attempts exhausted |
| 500 | Existing `INTERNAL_ERROR` | Unexpected lookup, SQL, transaction or storage failure |

Add English and Indonesian messages for each new code; preserve wrapping via `errs.Error`. Do not expose raw SQL, source HTML, private content, or ownership facts in denials. Ordinary repeated publish/unpublish is successful, not 409. Infrastructure errors do not become 404 or silently empty pages. No blog-specific 503 event failure exists because Blogs has no publisher.

## 8. Constraints Checklist

- [x] Design uses four root layers, current auth, named public-field structs, raw parameterized SQL and plain repository adapters.
- [x] BR-TASKDEMO-050–054: user-only create, server ownership, owner edits, authenticated/current actor.
- [x] BR-TASKDEMO-060–067: draft/published only, owner publication, owner/admin withdrawal/deletion, authorized idempotency.
- [x] BR-TASKDEMO-070–075: public/private/admin separation, deleted exceptions, read-time owner eligibility.
- [x] BR-TASKDEMO-080–084: Unicode bounds, sanitized HTML, publication validation, stable unique title-derived slug.
- [x] OQ-01–14, requested decision log, API documentation task and stable commit Task IDs are incorporated.
- [x] Migration → domain → repository → use case → handler/route → tests is the task dependency order; API docs is an explicit step before final tests.
- [x] Existing task/account API and completion events are preserved. These checks describe design coverage only.

## 9. Risks & Open Questions

No blocking business questions remain. Slug base/suffix, endpoint layout, pagination validation, admin filter and sanitizer behavior above are engineering choices within approved OQs, recorded here and in the decision log rather than presented as new business approvals.

- Sanitizer correctness is the largest implementation risk. Adversarial reparse tests must prove closed tag/attribute policy, single entity decoding and no mutation after rejection.
- Non-Latin title fallback is less descriptive; stable Unicode slugs/transliteration would require an intentional contract revision.
- Offset pages can shift under concurrent writes and large offsets cost more; reassess with measured usage before adopting cursor pagination.
- Memory optional mode cannot supply PostgreSQL cross-store isolation. Test transactional blog behavior there and SQL isolation against a real database.
- Existing migration tests target older versions and migrate downward. Run blog migration scenarios in a disposable, isolated database and serialize migration tests; never point them at shared data.
- Row locks serialize writes per blog, not account authentication with all in-flight actions. Public reads honor their statement/transaction snapshot and avoid cross-request caching; stronger account-write synchronization would be a separate requirement.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-10-29 | Initial additive Blogs design | Approved capability and delta; no implementation changes |
