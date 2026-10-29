# API changelog

## 1.2.0 - 2026-10-30

- Add unauthenticated public blog reads: `GET /blogs`, `GET /blogs/{slug}` (`security: []`).
- Add authenticated management: `POST /blog-records` (user role, `201`), `GET /blog-records` (admin role), and id-based get/update/publish/unpublish/delete on `/blog-records/{id}`.
- Create is `201`; other blog successes are `200`. Invalid `/blog-records/{id}` is `400` `INVALID_BLOG_IDENTIFIER`.

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
