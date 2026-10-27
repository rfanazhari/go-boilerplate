# Decision Log: User Management, Authentication, and Task Ownership

Technical decisions in this feature that exist because of business decisions or constraints. Each entry records why, what it means for the product, and when it should be revisited.

## D1: Opaque Bearer token, not JWT access + refresh

- **Business driver:** Login must be deniable and already-issued credentials must be withdrawn the moment an admin deactivates or soft-deletes someone (capability Q2 / Q10). JWT was listed as out of scope for this capability.
- **Technical decision:** Login issues a random opaque token. The service stores only a hash of that token. Clients send `Authorization: Bearer`. There is no refresh token in this slice. Health and login stay unauthenticated.
- **What this means for the product:** When Ada deactivates Budi, Budi’s next API call fails as unauthenticated even if he still has the string he logged in with. The organisation can take someone out of the service immediately, not “when the JWT expires.”
- **Alternatives considered:**
  - **JWT access + refresh:** Access tokens are self-contained and scale without a lookup, but they stay valid until expiry unless a denylist is added. A denylist is the same operational cost as storing opaque tokens, without fitting the “revoke now” rule as cleanly. Refresh tokens add a second credential type the capability does not describe.
  - **Server sessions / cookies:** Fine for browsers; this API is Bearer-oriented and the capability already chose `Authorization: Bearer`.
- **Trade-offs accepted:** Every authenticated request hits the token store. Stolen tokens remain usable until revoke or the user is deactivated. No silent refresh; the client logs in again after revoke.
- **Revisit when:** A stateless multi-instance deployment cannot share the token table, or product asks for short-lived access tokens plus refresh, or a real IdP/SSO is introduced.
- **Status:** active

## D2: Password hashing with bcrypt, not Argon2

- **Business driver:** Admins set passwords for new accounts; users may change their own password. Those secrets must not be stored reversible. There is no extra compliance mandate (e.g. a named algorithm in a regulation) in the capability.
- **Technical decision:** Hash and verify with bcrypt (`golang.org/x/crypto/bcrypt`), cost factor recorded in code (default cost 12 unless runtime proves too slow). Domain tests inject a fake hasher.
- **What this means for the product:** A database leak should not yield usable passwords. Login stays a normal email + password check.
- **Alternatives considered:**
  - **Argon2id:** Stronger against GPU guessing and often preferred in new greenfield systems. It is not in the current module dependencies; parameters (memory, iterations) need ops tuning on every environment, including `INFRA_OPTIONAL` tests.
  - **scrypt / PBKDF2:** No project precedent; bcrypt is the usual Go choice for this shape of app.
- **Trade-offs accepted:** bcrypt caps password length (72 bytes) — enforce a documented max on input. Weaker than Argon2 against large offline attacks if cost is left too low.
- **Revisit when:** A security review or regulation requires Argon2id, or password max-length becomes a support issue.
- **Status:** active

## D3: Role checks in HTTP middleware, not inside use cases

- **Business driver:** Admin work is denied to the user role (`BR-TASKDEMO-011`). Creating a task is user-only; admin create is forbidden (Q11). Product wants a clear split: “who may call this kind of operation” vs “which task this person owns.”
- **Technical decision:** gorilla/mux subrouters: public (health, login), authenticated, `RequireRole("admin")` for account routes, `RequireRole("user")` for `POST /tasks` and `/profile`. Use cases receive the caller’s identity and enforce **ownership** (same 404 as unknown id) and last-admin/self rules. Use cases do not contain a general RBAC matrix.
- **What this means for the product:** A user who calls an admin URL gets **403** (forbidden role). A user who calls a colleague’s task URL gets **404** (as if the work did not exist). Those two outcomes stay distinct, as Q12 required.
- **Alternatives considered:**
  - **All checks in the use case:** One place for rules, but every handler would be reachable without a role gate, and HTTP 403 vs 404 would mix with domain errors more easily. Product asked for middleware RBAC.
  - **All checks in middleware including ownership:** Middleware would have to load the task before the use case, duplicating `Get` and leaking repository into delivery.
- **Trade-offs accepted:** Role policy is URL-shaped; a new admin URL must be registered on the admin group or it is open to any authenticated role. Ownership cannot be expressed as a route group.
- **Revisit when:** Roles grow beyond admin/user, or the same use case is exposed over a second transport (e.g. queue worker) that has no HTTP middleware.
- **Status:** active

## D4: Soft delete plus unique email only among live accounts

- **Business driver:** Soft delete must not pretend the person never existed (`BR-TASKDEMO-023`), and the same email may be used again on a later account (Q3).
- **Technical decision:** `deleted_at` on `accounts`. Unique index on `LOWER(email)` **where `deleted_at IS NULL`**. Lookup-for-login ignores soft-deleted rows. Historical rows remain for ownership FKs.
- **What this means for the product:** Ada can retire Citra and later add a new person using Citra’s email. Old tasks still point at Citra’s retired account id; they are not re-parented (transfer is out of scope).
- **Alternatives considered:**
  - **Global unique email:** Blocks reuse; contradicts Q3.
  - **Hard delete:** Hides who owned work; contradicts the capability.
  - **Rename email on delete** (e.g. suffix `+deleted`): Reuse works but pollutes records and is a hidden rule.
- **Trade-offs accepted:** Two rows can share an email over time; admin tools must not assume email is a permanent key. GET-by-id still finds the retired row for ownership display.
- **Revisit when:** Product wants email reuse **and** a unique login identifier that never repeats, or GDPR-style erasure of retired rows.
- **Status:** active

## D5: Issued tokens are revoked when the account is deactivated or soft-deleted

- **Business driver:** Non-active and soft-deleted people must not keep working with a token they already received (Q2). Login is also denied.
- **Technical decision:** Deactivate and soft-delete run in one transaction with `DeleteByAccountID` on the token store. Auth middleware treats a missing hash, or a live token whose account is no longer active, as unauthenticated (401).
- **What this means for the product:** Access stops on the next request, not at some expiry. Reactivation does **not** restore old tokens; the person logs in again.
- **Alternatives considered:**
  - **Leave tokens until they expire:** Simpler store, but contradicts “revoke now.” Opaque tokens in this design have no expiry anyway.
  - **JWT with short TTL and no denylist:** Window of continued access after deactivate; rejected under D1.
- **Trade-offs accepted:** A request in flight during deactivate may still complete. No logout endpoint in this slice (revoke is an admin lifecycle effect).
- **Revisit when:** Product adds self-logout, device lists, or timed sessions.
- **Status:** active

## D6a: Bootstrap admin row in migration for owner backfill

- **Business driver:** D6 requires every task row to have `owner_id` NOT NULL immediately after migrate, while D2 forbids plaintext passwords in SQL and the seed design hashes passwords at process start.
- **Technical decision:** The accounts migration inserts one bootstrap admin with fixed UUID `e0000000-0000-4000-8000-000000000001` and a precomputed bcrypt hash of the default `SEED_ADMIN_PASSWORD` (`changeme`). `tasks.owner_id` is backfilled to that id in the same migration. `seed.SeedAdmin` at runtime updates that row from env; if no bootstrap row and no live admin, it inserts from env.
- **What this means for the product:** Migrate+backfill can run before the app; the first login uses env-configured credentials after the app has started at least once (or default dev credentials if env matches defaults).
- **Alternatives considered:**
  - **Nullable `owner_id` until seed:** Violates NOT NULL at end of up migration.
  - **Seed inside `cmd/migrate`:** Extra coupling; entrypoint already runs migrate then app.
- **Trade-offs accepted:** Bootstrap id is stable forever; historical tasks always reference it until transfer exists.
- **Revisit when:** Product requires migrate-only deploy with no runtime seed step.
- **Status:** active

## D6: Legacy tasks without an owner are assigned to the seeded admin

- **Business driver:** Every task must have an owner after this change (`BR-TASKDEMO-040`), and existing rows have none (Q13). There is no “unowned” state in the product.
- **Technical decision:** After the first admin is seeded, migration (or a single transactional migrate+backfill) sets `tasks.owner_id` to that admin, then enforces `NOT NULL` and a foreign key. New creates always set the authenticated user. Admin still cannot **create** tasks going forward; they may only **own** historical rows and **GET** any id.
- **What this means for the product:** Old work does not disappear. It shows up as owned by the bootstrap admin. Users cannot mutate those rows (not owners). An admin cannot start/complete/reschedule them either (Q11), only view by id. Cleaning or reassigning that backlog is a later product decision (transfer is out of scope).
- **Alternatives considered:**
  - **Delete legacy tasks:** Data loss; not requested.
  - **Nullable owner:** Leaves an unowned hole the capability forbids.
  - **Synthetic “system” user:** Extra role/actor the ubiquitous language does not have.
- **Trade-offs accepted:** Historical rows sit with an admin owner who cannot mutate them. Reports that assume “admin never owns tasks” will be wrong for pre-migration data.
- **Revisit when:** Product adds ownership transfer or a one-time reassignment of the backlog.
- **Status:** active
