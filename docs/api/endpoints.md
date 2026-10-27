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

Identifiers are server-generated UUIDs. Statuses: `todo`, `in_progress`, `done`.

Errors: 400 invalid input, 404 missing task/route, 405 method, 409 illegal transition/final task, 503 event publish failure, 500 unexpected.
