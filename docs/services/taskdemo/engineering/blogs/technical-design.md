# Technical Design: Blogs

## 1. Overview

Blogs adds an owned writing capability to taskdemo. A signed-in user (not an admin) creates a draft, edits it, and publishes it for anonymous readers. The author or an admin may unpublish or soft-delete it. Admins moderate; they do not author. Public list and detail show a writing only when it is published, not soft-deleted, and the author’s account is active and not soft-deleted.

This is Mode A. No blog types, tables, routes, or tests exist. Account login, roles, and task ownership already exist and stay in place. Business source: [blogs capability](../../business/capabilities/blogs.md) (BR-TASKDEMO-050–054, 060–067, 070–075, 080–084; OQ-01–14 approved 2026-10-29). Impact source: [blogs delta report](../../delta-report/delta-report-blogs.md). Product-facing technical choices: [decision log](./decision-log.md) D1–D5.

Vocabulary below uses the capability’s terms. **Publication status** is `draft` or `published`. **Soft-deleted** means `deleted_at` is set. That is a different fact from an account’s soft delete, and it is not a third publication status.

### Goals

- Persist blogs and expose the public, private, and admin operations in §2.
- Enforce the capability rules in the blog model and the blog use case, with the disclosure mapping in §7.
- Keep task routes, task lifecycle, account contracts, and `TaskCompleted` unchanged.

### Non-Goals

- Task list, task delete, ownership transfer, and changes to task title rules.
- Account registration, a new credential type, or changes to login and token revocation.
- Restore, hard delete, retention, moderation reasons, notifications, search, extra public filters, and an author’s private list.
- A blog message, outbox, or RabbitMQ publish. Blog “publish” changes publication status only.

## 2. API / Interface Contract Changes

Additive HTTP JSON. Existing routes, status codes, and the `TaskCompleted` payload stay as they are. Global OpenAPI security remains bearer; the two public blog operations set `security: []`, the same override `/health` and `/login` already use (`docs/api/openapi.yaml`).

Success envelope: `status: "success"` and `data`. Error envelope: `status: "error"`, `data: {}`, `error.message`, `error.code`, `error.errors`. Create returns `201`. Every other blog success returns `200`. Timestamps use the existing millisecond UTC layout from `delivery/http/time.go` (`formatTime`).

Clients never send `id`, `ownerId`, `slug`, `status`, `createdAt`, `updatedAt`, or `deletedAt`.

### Public reading (no login)

Registered on the root router, before the authenticated subrouter. A token on these routes is ignored (D2, D5).

| Method | Path | Success | Data |
| --- | --- | --- | --- |
| GET | `/blogs` | 200 | Page of public items |
| GET | `/blogs/{slug}` | 200 | Public detail |

Query on the list: `page`, `pageSize`. See D4 and §6.

Public item fields: `id`, `slug`, `title`, `authorName`, `createdAt`. No `content`.

Public detail fields: `id`, `slug`, `title`, `content`, `status` (`published`), `authorName`, `createdAt`.

`authorName` is `accounts.name` at read time.

Page shape:

```json
{
  "status": "success",
  "data": {
    "items": [],
    "page": 1,
    "pageSize": 20,
    "total": 0
  }
}
```

### Signed-in management

All of these sit behind the existing `Auth.Middleware` (`delivery/http/auth.go`). The caller must be an active, non-deleted account. Role subrouters match the current router: `RequireRole(user)` and `RequireRole(admin)`.

| Method | Path | Who reaches the handler | Success | Data |
| --- | --- | --- | --- | --- |
| POST | `/blog-records` | user role | 201 | Managed detail |
| GET | `/blog-records` | admin role | 200 | Admin page |
| GET | `/blog-records/{id}` | any authenticated role | 200 | Managed detail |
| PATCH | `/blog-records/{id}` | any authenticated role | 200 | Managed detail |
| POST | `/blog-records/{id}/publish` | any authenticated role | 200 | Managed detail |
| POST | `/blog-records/{id}/unpublish` | any authenticated role | 200 | Managed detail |
| DELETE | `/blog-records/{id}` | any authenticated role | 200 | `{}` |

`{id}` is the blog UUID. `{slug}` is never accepted on `/blog-records`. The public slug is never looked up as a management id.

Admin list query adds `deleted`:

| Value | Rows |
| --- | --- |
| omitted or `exclude` | `deleted_at` is null |
| `only` | `deleted_at` is set |
| `all` | no deletion filter |
| anything else | 400 `VALIDATION_FAILED` |

Admin list items: `id`, `ownerId`, `slug`, `title`, `status`, `authorName`, `createdAt`, `deletedAt`. No `content`. The admin list includes drafts and blogs whose author is inactive or soft-deleted. It uses the same `page` / `pageSize` rules as the public list.

Managed detail: `id`, `ownerId`, `slug`, `title`, `content`, `status`, `authorName`, `createdAt`, `updatedAt`, `deletedAt` (`null` when not deleted).

### Bodies

| Operation | Body |
| --- | --- |
| Create | `{ "title": string, "content"?: string }`. Missing `content` means empty. `content: null` is 400. |
| Update | `{ "title": string, "content": string }`. Both keys required. `content: ""` clears content when the draft rules allow it. |
| Publish, unpublish, delete | No body, or `{}`. Any field is 400 `MALFORMED_REQUEST`, same rule as task mutations in `delivery/http/task.go`. |

Unknown JSON fields are rejected by the existing `decodeJSON` (`DisallowUnknownFields`).

OpenAPI info version becomes `1.2.0`. `docs/api/CHANGELOG.md` gains an additive 1.2.0 entry. `docs/api/endpoints.md` gains a Blogs section; the stale task rows in that file are left untouched.

No protobuf and no new environment variable.

## 3. Data Model & Schema Changes

New table only. Migration pair sorts after `migrations/pgsql/20261027225508_accounts_tokens_task_owner`. Down drops `blogs` and nothing else.

```sql
CREATE TABLE blogs (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES accounts (id),
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT blogs_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT blogs_slug_check CHECK (
        slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
        AND char_length(slug) <= 89
    )
);

CREATE UNIQUE INDEX blogs_slug_uidx ON blogs (slug);

CREATE INDEX "blogs_owner_id_INDEX" ON blogs (owner_id);

CREATE INDEX "blogs_public_list_INDEX"
    ON blogs (created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND status = 'published';
```

`blogs_slug_uidx` is not partial. A soft-deleted slug remains reserved (D1, D3). There is no backfill and no seed row. `owner_id` uses `ON DELETE` restrict; accounts are not hard-deleted.

`content` stores the sanitized HTML, not the plain-text measurement. `title` stores the trimmed title. `updated_at` comes from the domain clock, not `time.Now()` inside the adapter (task `Save` uses the database clock; blog tests and idempotent “no change” need the injected clock).

Public list index matches the public filter and sort. Author eligibility still joins `accounts` at read time; the partial index does not encode account status (D5).

### Concepts and rules

| Term | Meaning in this service |
| --- | --- |
| Blog | Aggregate root. Identity is the UUID for its whole life, including after soft delete and title edits. |
| Author / owner | `owner_id`. Set from the signed-in user at create. Never taken from the body. |
| Publication status | `draft` or `published` only. |
| Soft-deleted blog | `deleted_at` set. Orthogonal to publication status. |
| Slug | Public identity. Fixed at create. Not the aggregate’s identity. |
| Public eligibility | Published, blog not soft-deleted, author `status = active`, author `deleted_at` null. All four are required on list and detail. |

`BlogTitle`, `BlogContent`, and `BlogSlug` are value objects owned by `Blog`. They are not shared with `Task`. Task titles stay a 1–100 string on `domain/aggregate/task.go`. Reusing `INVALID_TITLE` would show the task message (“1 and 100 characters”) from `translations/errors.en.toml`.

| Material rule | Owner | Enforcement paths | Proof plan |
| --- | --- | --- | --- |
| BR-050 create is user role only | HTTP `RequireRole(user)` and `BlogUseCase.Create` | Admin `POST /blog-records` stops at the router (403) even if the body is invalid. Direct use case call with an admin actor returns `FORBIDDEN` and does not save. | Use case table: admin actor. Smoke: admin token receives 403. |
| BR-051 creator is owner | `BlogUseCase.Create` | `owner_id` is the actor id. Body has no owner field. | Use case asserts saved owner. Unknown JSON `ownerId` is 400. |
| BR-052 edit is owner-only, including against an admin | Use case disclosure, then `Blog.Revise` | Non-owner, including admin, never reaches a write. | Admin edit of a visible blog is `FORBIDDEN`. Draft of someone else is `BLOG_NOT_FOUND`. |
| BR-053 management requires login | `Auth.Middleware` | Every `/blog-records` route. | Smoke: no token is 401. Public GETs stay 200 without a token. |
| BR-054 inactive or soft-deleted account | Existing `Auth.authenticate` | Protected blog routes use that middleware. It already returns 401 and revokes tokens when the account is non-active or soft-deleted (`delivery/http/auth.go`). Public routes do not use the middleware, so those accounts can still read as anonymous visitors. | Smoke with a non-active token on `POST /blog-records` is 401, and `GET /blogs` without a token still works. |
| BR-060 new blog is draft | `NewBlog` | Create is the only constructor. Status is `draft`. | Domain test. Create response `status` is `draft`. |
| BR-061 status is only draft or published | `PublicationStatus` and the SQL check | Parse accepts only those two values. Soft delete does not change status. | Domain rejects a third value. Migration check constraint. |
| BR-062 owner publishes; may republish after admin unpublish | Use case allows the owner; `Blog.Publish` checks content and sets `published` | No approval step. | Use case: owner publish after admin unpublish succeeds. |
| BR-063 non-owner cannot publish | Use case | Admin publish of a visible blog is `FORBIDDEN`. | Use case and smoke. |
| BR-064 owner unpublish | Use case allows the owner; `Blog.Unpublish` sets `draft` | Published blog leaves the public predicate. | Public get then returns not found. |
| BR-065 admin unpublish | Use case allows admin | Admin is not required to be the owner. | Use case. |
| BR-066 owner and admin soft-delete; repeat rules (OQ-11) | `Blog.SoftDelete`; use case disclosure | First delete sets `deleted_at` and leaves status unchanged. Admin repeat: success, no write. Owner repeat: `BLOG_NOT_FOUND` because the row is hidden. | Use case: second call’s save count. Domain: status unchanged. |
| BR-067 stranger cannot unpublish or delete | Use case | Visible published blog → `FORBIDDEN`. Hidden draft or deleted blog → `BLOG_NOT_FOUND`. | Matrix in §6. |
| BR-070 public read without login | Router | `GET /blogs` and `GET /blogs/{slug}` are outside `Auth.Middleware`. | Smoke without `Authorization`. |
| BR-071 drafts are absent from the public channel | Repository public predicate | Owner and admin tokens do not change this. | Public get of a draft slug is `BLOG_NOT_FOUND`. List omits it. |
| BR-072 draft private read is owner or admin | Use case `Get` | `GET /blog-records/{id}`. | Owner 200, admin 200, other user 404. |
| BR-073 deleted blogs are hidden from non-admins | Use case on management routes; public predicate on public routes | Applies to get, list, edit, publish, unpublish, and repeat delete. | Owner get after delete is 404. Public get is 404. |
| BR-074 admin can read a deleted blog, read-only | Admin list filter and `Get` | Edit, publish, and unpublish of a deleted blog return `BLOG_DELETED` (409). Delete repeat is the OQ-11 success. | Use case matrix. |
| BR-075 / OQ-08 author eligibility at read time | Public repository queries | Join `accounts`. No write on account deactivate. Reactivation makes the same published, non-deleted blogs public again. | Memory and Postgres fixtures: deactivate hides list and detail; activate shows them; blog status still `published`. |
| BR-080 title 10–200 rune code points after trim | `BlogTitle` | Create and revise. Trim unicode space first, then `utf8.RuneCountInString`. | Boundaries 9, 10, 200, 201. `Catatan Go` is accepted (10). |
| BR-081 content required to publish | `Blog.Publish` | Plain-text length 0 fails publish. Empty draft stays a draft. | Publish of empty content is `INVALID_BLOG_CONTENT` and status stays `draft`. |
| BR-082 draft length 0–1000; published length 100–1000 | `BlogContent` for the target status | Revise of a published blog uses the published bounds. Invalid revise changes no field. | 99, 100, 1000, 1001 on publish. Invalid published edit does not change `updated_at`. |
| BR-083 HTML allowlist | `HTMLSanitizer` implementation; measurement in `BlogContent` | Sanitize, then measure, then store the sanitized HTML. | Infrastructure sanitizer tests plus the plain-text table below. Use case stores the sanitizer’s return value, not the raw body. |
| BR-084 / OQ-06 / OQ-07 slug | `BlogSlug` and create retry | Built at create, unchanged by `Revise`, unique including deleted rows. Public detail queries slug only. | Rename keeps slug. Two identical titles get two slugs. Deleted slug insert fails the unique index. |

Domain assessment applies. The blog is an entity and the aggregate root: the UUID survives title edits, publication changes, and soft delete. Title, content, and slug are values with their own normalization. Account remains a separate aggregate; public eligibility is a read policy across the two, not a single consistency boundary (D5). Task status is a different language in a different model and is not reused.

### Title

`BlogTitle` trims with `strings.TrimSpace`, then requires 10–200 Unicode code points. The stored value is the trimmed string. The title is plain text. Angle brackets are characters, not HTML. Create and revise both use this constructor. `RestoreBlog` does not run it.

### Content measurement

Order for new input:

1. `HTMLSanitizer.Sanitize` (production policy in §4).
2. Domain plain-text length on that result.
3. Bounds for the operation. On success, store the sanitized HTML from step 1.

Plain text is derived by parsing the sanitized HTML (`golang.org/x/net/html` fragment). The parser’s text nodes are already entity-decoded. Emit a separator at `br` and at the boundary of `p`, `h1`, `h2`, `h3`, `h4`, `h5`. Collapse unicode whitespace and those separators to a single space, then trim. Count runes. Markup is not counted. Whitespace-only content has length 0.

| Sanitized HTML | Plain text | Length |
| --- | --- | --- |
| `<p>hello</p><p>world</p>` | `hello world` | 11 |
| `<p>hello<br>world</p>` | `hello world` | 11 |
| `<p>&amp;</p>` | `&` | 1 |
| `<p>   </p>` | empty | 0 |
| `a` repeated 100 times | same | 100 |

Draft accepts length 0 through 1000. Publish and a revise of a published blog accept 100 through 1000 inclusive. A revise that fails leaves title, content, status, slug, and `updated_at` as they were.

### Slug

`BlogSlug` is built from the trimmed title and an 8-character lowercase hex suffix supplied by the use case:

1. Unicode NFKD (`golang.org/x/text/unicode/norm`, already in `go.mod`).
2. Drop combining marks.
3. Lowercase.
4. Replace every rune outside `[a-z0-9]` with `-`.
5. Collapse repeated hyphens and trim hyphens.
6. If nothing remains, the base is `blog`.
7. Cut the base to 80 runes and trim a trailing hyphen.
8. Slug is `base + "-" + suffix`.

`Catatan Go` becomes `catatan-go-` plus the suffix. A symbol-only title becomes `blog-` plus the suffix. The suffix comes from 4 cryptographically random bytes. The use case retries up to five times when `Save` reports a unique-slug violation, then returns `INTERNAL_ERROR`. Title uniqueness is not an error. `Revise` cannot set a slug. `RestoreBlog` accepts the stored slug as-is.

### Restoration

`RestoreBlog` trusts the row: title, content, status, slug, `deleted_at`, and timestamps. It does not sanitize, remeasure, or reject a short title. That is safe for this module because every new row is written by `NewBlog` / `Revise` / `Publish`, and the table starts empty. A later tightening of the bounds must not start failing reads of existing rows without an explicit migration. Timestamps are normalized with the existing `normalizeInstant` (UTC, millisecond precision) used by `RestoreTask`.

## 4. Component Breakdown

| Layer/Component | Status | Justification |
| --- | --- | --- |
| `domain/aggregate` Blog, BlogTitle, BlogContent, BlogSlug | New | Task does not model drafts, HTML, or a separate deletion fact. |
| `domain/enum.PublicationStatus` | New | `domain/enum/status.go` is `todo` / `in_progress` / `done`. |
| `domain/repository.BlogRepository` | New | Port for save, locked get, public reads, admin list. |
| `domain/service.HTMLSanitizer` | New | Port so the aggregate can require sanitized input without importing infrastructure. |
| `domain/service.Clock`, `SystemClock` | Reused as-is | Same millisecond UTC clock as tasks. |
| `domain/aggregate.Task`, `Account` | Reused as-is | No behavior change. Account `Name`, `Status`, `IsSoftDeleted`, and `Role` are read for views and eligibility. |
| `application/usecase.BlogUseCase`, `application/dto` blog types | New | Commands, disclosure, paging. |
| `application/usecase.TaskUseCase` | Reused as-is | Blogs must not call it. |
| `delivery/http` blog handler and router | New + modified router | Translate JSON and register the two URL spaces. |
| `delivery/http.Auth`, `RequireRole`, envelope, `decodeJSON`, `formatTime` | Reused as-is | 401/403 mapping and localization already exist. |
| `delivery/http.pathID` | Reused as-is for tasks only | It returns `INVALID_IDENTIFIER`, whose message says “task identifier”. Blog ids use a new helper and `INVALID_BLOG_IDENTIFIER`. |
| `infrastructure/repository/blogrepository` | New | Postgres, memory, unavailable. |
| `infrastructure/repository/pgxtx` | Reused as-is | Blog transactions use `pgxtx.With` / `From`, as account and token adapters do. |
| `infrastructure/service` HTML sanitizer | New | bluemonday policy below. |
| `infrastructure/runtime` | Modified | Wire the blog use case, both adapters, and the handler. |
| `pkg/errs`, `translations/errors.{en,id}.toml` | Modified | New codes only. Existing messages stay. |
| `docs/api/openapi.yaml`, changelog, endpoints | Modified | Additive contract. |
| RabbitMQ, `CompletionPublisher` | Reused as-is for tasks | Not a blog dependency. |

### Detail per component

**Blog aggregate** (`domain/aggregate/blog.go`). Unexported fields. `NewBlog` requires a non-nil owner id, a `BlogTitle`, a `BlogContent`, a `BlogSlug`, and `now`. It sets `draft`, nil `deleted_at`, and equal `created_at` / `updated_at`. `Publish` and `Unpublish` are idempotent when the blog is not deleted: already published, or already draft, returns nil and does not change `updated_at`. Either method on a deleted blog returns `BLOG_DELETED` and changes nothing. `Publish` of a non-deleted blog whose plain-text length is outside 100–1000 returns `INVALID_BLOG_CONTENT` and changes nothing. `Revise` replaces title and content when the bounds for the current status pass, and does not change slug or status. `SoftDelete` sets `deleted_at` once; a second call reports already deleted and does not change `updated_at`.

Permission is not inside those methods. The use case decides whether this actor may invoke them, then applies disclosure. That split keeps “may this person do this?” next to the account lookup disclosure needs, and keeps “is this transition valid?” on the aggregate. `BlogUseCase` is the production caller.

**HTML policy** (`infrastructure/service`, `github.com/microcosm-cc/bluemonday`). Build with `NewPolicy`, not `UGCPolicy` and not `AllowStandardURLs`. UGC allows images, classes, and extra attributes. `AllowStandardURLs` turns relative URLs on and adds `rel="nofollow"`, which the allowlist does not permit ([bluemonday README](https://github.com/microcosm-cc/bluemonday/blob/main/README.md)).

- Allow elements `p`, `br`, `h1`, `h2`, `h3`, `h4`, `h5`, `a`.
- Allow attribute `href` on `a` only.
- Allow URL schemes `http` and `https` only.
- `AllowRelativeURLs(false)` and `RequireParseableURLs(true)`.
- Do not add `rel`, `target`, or other attributes.

Malformed HTML is re-serialized. It is not a separate client error. Disallowed tags are removed and their text is kept, except where bluemonday drops `script` and `style` contents. The stored and returned `content` is this output. Domain tests inject a fake sanitizer. The allowlist proof is the infrastructure test, and a use case test shows the saved content is the sanitizer output.

**Repository port**

- `Save(ctx, *Blog) error` — insert or update. Unique slug violation is a typed conflict the use case retries on insert only.
- `Get(ctx, id) (*Blog, error)` — any row, including deleted. Inside a `pgxtx` transaction, Postgres uses `SELECT … FOR UPDATE`. Missing id is `BLOG_NOT_FOUND`.
- `GetPublicBySlug(ctx, slug) (Blog, authorName, error)` — one query with the public predicate. Miss is `BLOG_NOT_FOUND` for a draft, a deleted blog, an ineligible author, and an unknown slug.
- `ListPublic(ctx, page, pageSize) (items, total, error)` — same predicate, `ORDER BY created_at DESC, id DESC`, limit/offset applied after the predicate, `total` of the full eligible set. List select omits `content`.
- `ListAdmin(ctx, page, pageSize, deletedFilter) (items, total, error)` — deletion filter only. No author-eligibility filter. Omits `content`. Same sort.
- `RunInTransaction(ctx, fn) error`.

`BlogSummary` in the repository package is the list row (no content, plus `authorName`). It is a query result, not a second aggregate. Public eligibility is the behavior of `GetPublicBySlug` and `ListPublic`, not a separate type.

**Memory adapter.** Same predicate and sort in Go, using an `Accounts repository.AccountRepository` field on the plain struct (no constructor). `RunInTransaction` holds the mutex for the whole callback so `Get` and `Save` cannot interleave. Task memory (`taskrepository.Memory.RunInTransaction`) calls the function without holding the lock; copying that would let a stale `Save` clear `deleted_at`. That departure is limited to the blog adapter.

**Postgres adapter.** Raw `$1` SQL. Transactions through `pgxtx`. Mutations in the use case run inside `RunInTransaction`, `Get` locks the row, then `Save` writes it. Public list and detail join:

```sql
accounts.status = 'active' AND accounts.deleted_at IS NULL
AND blogs.status = 'published' AND blogs.deleted_at IS NULL
```

**Use case.** One `BlogUseCase` with `Blogs`, `Accounts`, `Sanitizer`, `Clock`, `IDs`, and `Suffix`, matching the injectable `IDs` field on `TaskUseCase`. Paging validation lives here so a non-HTTP caller cannot skip it: `page` defaults to 1, `pageSize` to 20, `page >= 1`, `pageSize` in 1..100, and `(page-1)*pageSize` must not overflow. `page` above 1_000_000 is `VALIDATION_FAILED`. The handler passes query values through; it does not apply a second policy.

Disclosure runs after a successful `Get` and before a mutation:

- No row → `BLOG_NOT_FOUND`.
- Actor is admin → the blog is visible, including when deleted.
- Actor is the owner and the blog is not deleted → visible.
- Any other actor → visible only when the public predicate is true (load the owner account).
- Hidden → `BLOG_NOT_FOUND`, whatever the operation would have been.
- Visible and the operation is not allowed → `FORBIDDEN` (admin or stranger edit, admin or stranger publish, stranger unpublish, stranger delete).
- Visible, deleted, and the operation is edit, publish, or unpublish → `BLOG_DELETED`.
- Visible, deleted, and the operation is admin delete → 200 and no write.
- Allowed publish or unpublish that is already in the target state → 200 and no write.

Create does not use that matrix. Non-user role → `FORBIDDEN` and no insert. Slug retries only on the unique index.

**Handler** (`delivery/http/blog.go`). Maps the actor from `ActorFromContext` into `dto.BlogActor` (`ID`, `Role`). It does not import infrastructure. Author name on management responses comes from the use case (`Accounts.GetByID`), not from a client field. Delete writes `data: {}`.

**Runtime** (`infrastructure/runtime/runtime.go`). Follow the existing task wiring: placeholder `Unavailable`, replace with memory when `INFRA_OPTIONAL` is set, replace with Postgres when the pool is up. Inject `SystemClock` and the bluemonday sanitizer. No new config field.

### Material structure decisions and assessment

| Choice | Reason and material alternatives/tradeoffs | Authority and refinement boundary | Assessment outcome |
| --- | --- | --- | --- |
| Separate `Blog` aggregate and `PublicationStatus` | Task status and the 1–100 title are different rules. Sharing them would make a blog title change the task message and the task state machine. | Binding rules BR-060, BR-061, BR-080. Internal file split inside `domain/aggregate` is allowed. | Retained |
| `BlogTitle`, `BlogContent`, `BlogSlug` value objects | Create and revise both need one trim, one length rule, and one slug builder. A bare string leaves each caller able to skip that. Task’s string title is the wrong policy to copy (design assessment: different bounds are a reason to keep the models apart). | Constructors own new input. `RestoreBlog` is the trusted path (§6). A later change may unexport helpers; it may not validate old rows with today’s create rules without a migration decision. | Retained |
| `deleted_at` beside status; global unique slug | Required by D1 and D3. A third status or a partial unique index would violate them. | Binding. Index style may follow later Postgres review; uniqueness including deleted rows may not. | Retained |
| `/blogs` public and `/blog-records` authenticated | D2. One prefix cannot serve an anonymous slug GET and an authenticated id GET: the slug route would match every single segment, and the protected router is a catch-all registered after specific public routes (`delivery/http/router.go`). | Binding for the two spaces and for “token ignored on public GETs”. Path spelling `/blog-records` may be renamed only with an OpenAPI change before T5 is released. | Retained |
| Role middleware for create and admin list; actor matrix in the use case | The router can express user versus admin, as it does for `POST /tasks` and `POST /accounts`. It cannot express author-or-admin without loading the blog. | D2. Handlers must not grow a second copy of the matrix. | Retained |
| bluemonday behind `HTMLSanitizer` | The allowlist is domain policy (BR-083). The library is the mechanism. `UGCPolicy` allows a wider set than the capability. Domain measurement stays in `BlogContent` so length cannot drift from the sanitizer. | Policy configuration in §4 is binding. Another sanitizer is allowed only if the same allowlist tests pass. | Retained |
| Read-time join for public eligibility | D5. Filtering in Go after `LIMIT` would return short pages and would disagree with detail. | Predicate must be shared by list and detail in each adapter. | Retained |
| Blog memory locks across `RunInTransaction`; Postgres `FOR UPDATE` | A full-row `Save` of a stale aggregate can clear `deleted_at`. Task memory does not hold the lock; that fact is current behavior, not a guarantee this conflict can tolerate. | Required for blog mutations. Do not “fix” task memory in this work. | Retained |
| One `BlogUseCase` | Matches `TaskUseCase`. Public reads and commands change for the same product rules. | Files may be split by operation. Authorization may not move into the handler. | Retained |
| Offset `page` / `pageSize` | D4 names that contract. Cursors would be a different API. | Query names, defaults, max 100, sort, and `total` are binding. | Retained |

## 5. Flow Sequence

### Public detail

1. `GET /blogs/{slug}` hits the root router. Middleware does not run.
2. Handler calls `BlogUseCase.GetPublic`.
3. Repository runs the public join. A miss returns `BLOG_NOT_FOUND`.
4. Handler writes 200 public detail, including sanitized `content`.

Public list is the same predicate, then sort, count, and page. The handler does not filter the page again.

### Create

1. `POST /blog-records` is on the user-role subrouter. Missing or inactive identity is 401. Admin is 403 before the body is read.
2. Handler decodes title and optional content.
3. Use case rejects a non-user actor, sanitizes content, builds title and slug, and calls `NewBlog`.
4. `Save`. On slug unique violation, mint a new suffix and retry, at most five times.
5. Respond 201 with managed detail.

### Publish

1. Authenticated `POST /blog-records/{id}/publish` with an empty body.
2. In a transaction, `Get` locks the row.
3. Disclosure: hidden → 404; visible non-owner → 403; deleted → 409.
4. `Publish`. Content failure → 400 and rollback. Already published → commit with no `UPDATE`.
5. Respond 200 managed detail.

Unpublish, revise, and delete follow the same lock and disclosure order. Delete’s success body is `{}`.

### Admin list

1. `GET /blog-records` is on the admin subrouter.
2. Use case validates `page`, `pageSize`, and `deleted`.
3. Repository lists with the deletion filter and the shared sort. Author eligibility is not applied.
4. Respond 200 admin page.

## 6. Request / Response (or Input / Output)

Paging (public and admin):

| Input | Result |
| --- | --- |
| `page` omitted | 1 |
| `pageSize` omitted | 20 |
| `pageSize` 20, `page` 1 | first page |
| `page` past the end | 200, `items: []`, `total` unchanged |
| `page` &lt; 1, `pageSize` &lt; 1, `pageSize` &gt; 100, non-integer, `page` &gt; 1_000_000, offset overflow | 400 `VALIDATION_FAILED` |

Sort is `created_at DESC, id DESC`. `id DESC` is a stable tie-break, not a claim that UUIDs encode time.

### Creation, restoration, and transition contracts

| Path | Accepted inputs/states and normalization | Errors/order and failure timing | Side effects and compatibility | Proof plan |
| --- | --- | --- | --- | --- |
| Create | User actor. Title trimmed, 10–200 runes. Content sanitized then measured, length 0–1000. Slug minted. Status becomes `draft`. | Malformed JSON, then title, then content. Role 403 happens before body validation on HTTP. Slug collision retries inside the use case; the client does not see it. | Insert one row. No event. `owner_id` is the actor. | Domain bounds. Use case: admin forbidden, owner set, sanitizer output stored. |
| Restore | Any row the checks allow, including a deleted published blog and any stored HTML. No re-sanitize and no length rejection. | Adapter returns internal error only if status is outside `draft`/`published` despite the check constraint. | Load does not write. | Round-trip a deleted published row. A deliberately short title still loads. |
| Revise | Non-deleted blog, after disclosure allows the owner. New title and new sanitized content. Published blogs must still meet 100–1000. Slug unchanged. | Invalid content or title: 400, no write. Hidden: 404. Admin: 403. Deleted admin: 409. | One update of title, content, `updated_at` on success. Public detail shows the new content immediately when the public predicate still holds. | Invalid published edit asserts the repository save was not called. |
| Publish / unpublish | Owner (publish) or owner or admin (unpublish). Not deleted. Publish rechecks stored content. Idempotent when already in the target state. | Same disclosure. Content failure does not change status. | No write on the idempotent path. No RabbitMQ message. | Second publish does not call `Save`. Admin unpublish then owner publish succeeds. |
| Soft delete | Owner or admin, after disclosure. Sets `deleted_at` only. | Owner repeat 404. Admin repeat 200 without `Save`. | Row remains. Slug remains reserved. Status remains. | Unique slug still rejects a new row with that slug. Public get is 404. Admin get is 200. |
| Public read | No actor. Predicate in §3. | Every miss is `BLOG_NOT_FOUND`. | Read-only. Token ignored. | Same fixture absent from list and detail. Inactive author hidden, then visible after reactivation, status still `published`. |

### Disclosure matrix (management routes)

| Actor | Blog | Get | Edit / publish | Unpublish / delete |
| --- | --- | --- | --- | --- |
| Owner | not deleted | 200 | edit 200; publish 200 | 200 |
| Owner | deleted | 404 | 404 | 404 |
| Admin | not deleted | 200 | 403 | unpublish 200; delete 200 |
| Admin | deleted | 200 | 409 | unpublish 409; delete 200 no write |
| Other user | publicly eligible | 403 | 403 | 403 |
| Other user | draft, deleted, or author ineligible | 404 | 404 | 404 |
| Anyone | no row | 404 | 404 | 404 |

Public routes never use this matrix. A publicly eligible blog is 200 by slug. Everything else is 404 by slug.

## 7. Error Handling

New codes. Kinds use the existing `writeError` switch. Messages are new keys in both translation files. Do not edit the text of `INVALID_TITLE`, `INVALID_IDENTIFIER`, or `TASK_NOT_FOUND`.

| Code | Kind | HTTP | When |
| --- | --- | --- | --- |
| `INVALID_BLOG_TITLE` | invalid input | 400 | Title missing or outside 10–200 after trim. |
| `INVALID_BLOG_CONTENT` | invalid input | 400 | Content length outside the bounds for this operation. |
| `INVALID_BLOG_IDENTIFIER` | invalid input | 400 | `/blog-records/{id}` is not a UUID. |
| `VALIDATION_FAILED` | invalid input | 400 | Paging or `deleted` filter. Existing message reused. |
| `MALFORMED_REQUEST` | invalid input | 400 | Bad JSON, unknown fields, or a body on publish/unpublish/delete. |
| `BLOG_NOT_FOUND` | not found | 404 | No row, unknown slug, or a blog this actor must not see. |
| `FORBIDDEN` | forbidden | 403 | Role gate, or the operation is forbidden on a blog the actor may see. Existing message reused. |
| `BLOG_DELETED` | conflict | 409 | Admin edit, publish, or unpublish of a soft-deleted blog. |
| `UNAUTHENTICATED` | unauthenticated | 401 | Existing middleware. |
| `INTERNAL_ERROR` | internal | 500 | Slug retries exhausted, or an unexpected database error. |

English message text:

| Code | Message |
| --- | --- |
| `INVALID_BLOG_TITLE` | title must be between 10 and 200 characters after trimming whitespace |
| `INVALID_BLOG_CONTENT` | content length is outside the allowed range for this operation |
| `INVALID_BLOG_IDENTIFIER` | blog identifier is not a valid uuid |
| `BLOG_NOT_FOUND` | blog was not found |
| `BLOG_DELETED` | a deleted blog cannot be changed |

Indonesian message text:

| Code | Message |
| --- | --- |
| `INVALID_BLOG_TITLE` | judul harus 10 sampai 200 karakter setelah spasi di tepi dihapus |
| `INVALID_BLOG_CONTENT` | panjang isi berada di luar batas untuk operasi ini |
| `INVALID_BLOG_IDENTIFIER` | pengenal blog bukan UUID yang valid |
| `BLOG_NOT_FOUND` | blog tidak ditemukan |
| `BLOG_DELETED` | blog yang sudah dihapus tidak dapat diubah |

On a management request the order is: 401, then role 403, then 400 for the id or the body, then 404 if hidden or missing, then 403 or 409, then 400 for title or content. A validation failure does not write. There is no 503 for blogs.

## 8. Constraints Checklist

- [ ] Task routes, task rules, account routes, and `TaskCompleted` stay unchanged.
- [ ] Dependencies stay inward: domain does not import application, delivery, infrastructure, or `cmd`.
- [ ] Public blog GETs are outside auth middleware and set OpenAPI `security: []`.
- [ ] `/blog-records` always requires an active, non-deleted account.
- [ ] Create is user-role only. Admin create is 403.
- [ ] Slug is unique including soft-deleted rows, fixed on rename, and is the only public detail key.
- [ ] Management lookup uses the UUID.
- [ ] Public list and detail share the eligibility predicate, including author state at read time.
- [ ] Page default 20, maximum 100, `created_at DESC, id DESC`.
- [ ] Deleted blogs are hidden from non-admins and are read-only for admins, with idempotent admin delete.
- [ ] HTML is sanitized before length checks and storage; the response returns that sanitized HTML.
- [ ] Envelope, `gofmt`, raw `$1` SQL, plain repository structs, `_test` packages, and Mockery are preserved.
- [ ] No new environment variable and no blog publisher.

## 9. Risks & Open Questions

No open business question. OQ-01–14 are decided. Nothing in this design is deferred in a way that blocks T1–T6.

| Risk | Mitigation |
| --- | --- |
| Public list and detail drift apart | One predicate per adapter. Tests use the same fixture for both methods. |
| Memory and Postgres disagree | Memory tests always run. Postgres integration tests run when `DATABASE_URL` is reachable and skip otherwise, matching `test/infrastructure/migration_test.go`. |
| Stale save clears `deleted_at` | Row lock in Postgres; mutex across the memory transaction. |
| Sanitizer allows more than BR-083 | `NewPolicy` plus the allowlist test. Do not start from `UGCPolicy`. |
| `GET /blogs/{slug}` captures a management path | Management prefix is `/blog-records`, registered only on the authenticated subrouter. Smoke asserts `GET /blog-records/{id}` without a token is 401. |
| Deep offset pages get slow | Accepted by D4. No cache in this release. A future cache has to apply the same predicate on list and detail. |
| Down migration deletes writings | Acceptable before any blog exists. After rows exist, roll the application back and leave the table, unless product accepts losing those rows. |

Out of scope, not blockers: search, author draft list, restore, hard delete, retention, moderation notes, notifications.

## 10. Revision History

| Version | Date | Change | Reason |
| --- | --- | --- | --- |
| v1 | 2026-10-30 | Initial design | Mode A from the blogs capability and delta report. |
