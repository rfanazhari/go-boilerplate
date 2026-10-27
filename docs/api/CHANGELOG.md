# API changelog

## 1.1.0 - 2026-10-27

- **Breaking:** Task routes require `Authorization: Bearer <opaque token>` except where noted.
- Add `POST /login` (unauthenticated) returning opaque Bearer credentials.
- Add admin account routes: `POST /accounts`, `PATCH /accounts/{id}/status`, `DELETE /accounts/{id}`.
- Add user profile route: `PATCH /profile`.
- Document `401` (`UNAUTHENTICATED`, `INVALID_CREDENTIALS`) and `403` (`FORBIDDEN`) responses.
- `GET /health` remains unauthenticated.
- Task `data` responses include read-only `ownerId` (set from the authenticated user on create; not accepted in request bodies).
- `POST /tasks` is limited to the `user` role; admins receive `403`. Non-owner task access returns `404` (`TASK_NOT_FOUND`).

## 1.0.0 - 2026-10-27

- Initial Task Tracking HTTP contract: create, start, complete, reschedule, get, and unauthenticated `/health`.
