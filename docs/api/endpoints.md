# Endpoints

All responses use `{ "status", "data", "error?" }`. Errors set `data` to `{}`.

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| GET | /health | 200 | Unauthenticated. HTTP readiness only. |
| POST | /tasks | 201 | Body: `{ "title", "dueDate?" }`. |
| POST | /tasks/{id}/start | 200 | Empty body or `{}`. |
| POST | /tasks/{id}/complete | 200 | Publishes TaskCompleted after save. |
| PATCH | /tasks/{id}/due-date | 200 | `{ "dueDate": RFC3339 \| null }`. |
| GET | /tasks/{id} | 200 | Details including overdue. |

## Blogs

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| GET | /blogs | 200 | Unauthenticated. Query: `page`, `pageSize`. Items omit `content`. |
| GET | /blogs/{slug} | 200 | Unauthenticated public detail. Token ignored. |
| POST | /blog-records | 201 | User role. Body: `{ "title", "content?" }`. Draft owned by caller. |
| GET | /blog-records | 200 | Admin role. Query: `page`, `pageSize`, `deleted`. |
| GET | /blog-records/{id} | 200 | Authenticated managed detail. |
| PATCH | /blog-records/{id} | 200 | `{ "title", "content" }`. |
| POST | /blog-records/{id}/publish | 200 | Empty body or `{}`. |
| POST | /blog-records/{id}/unpublish | 200 | Empty body or `{}`. |
| DELETE | /blog-records/{id} | 200 | Empty body or `{}`. `data` is `{}`. |

Public list/detail stay outside `Auth.Middleware`. Management always requires an active account. `{id}` is a UUID (`INVALID_BLOG_IDENTIFIER` when not).

Identifiers are server-generated UUIDs. Statuses: `todo`, `in_progress`, `done`.

Errors: 400 invalid input, 404 missing task/route, 405 method, 409 illegal transition/final task, 503 event publish failure, 500 unexpected.
