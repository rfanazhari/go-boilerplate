---
id: user-management-authentication-task-ownership
service: taskdemo
status: planned
rules: BR-TASKDEMO-010..011, BR-TASKDEMO-020..023, BR-TASKDEMO-030..031, BR-TASKDEMO-040..042
updated: 2026-10-27
implementation:
---

# User Management, Authentication, and Task Ownership

## Purpose

Decide who may use the service, which accounts an administrator may create or retire, and which work items a person may create as their own. The outcome is that tasks belong to people, and administrative account changes stay with the admin role.

## Ubiquitous language

| Term | Meaning |
|---|---|
| **Account** | A person the service recognises, with a role and a lifecycle (active, non-active, or soft-deleted). |
| **Admin** | The role allowed to add, deactivate, and soft-delete accounts. |
| **User** | The role allowed to maintain its own profile and to create owned tasks. |
| **Active** | An account state in which login is allowed and issued tokens remain valid until revoked for another reason. |
| **Non-active** | An account state set by an admin from active. Login is denied and issued tokens are revoked. An admin may return the account to active. |
| **Soft delete** | An admin action that retires an account without treating it as physically gone. Login is denied, issued tokens are revoked, and the email may be reused on a later account. |
| **Profile** | Name and password of the account; a user may change only those two fields for themselves. Role, status, and email are locked. |
| **Owner** | The account that created a task; that task belongs to them. |
| **Credential token** | An opaque bearer token issued at login, stored by the service so it can be revoked, and sent as `Authorization: Bearer`. |

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Add a new account | Admin | A new account exists for someone who could not create themselves. |
| Set status active → non-active | Admin | The target account is no longer active. |
| Soft-delete an account | Admin | The target account is retired (soft delete). |
| Log in | Account (active) | A credential token is issued for that identity. |
| Update own profile | User (authenticated) | Name and/or password for that identity are changed. |
| Create a task | User (authenticated) | A new task exists in todo and is owned by that person. An admin cannot create a task. |
| View a task | User or admin (authenticated) | A user may view only tasks they own. An admin may view any task by id. There is no list endpoint. |
| Change a task | Owner (authenticated) | Start, complete, and reschedule succeed only for the owner. An admin who is not the owner cannot change the task. |

## Rules

### Auth

### BR-TASKDEMO-010 — Acting as an account requires an authenticated identity

**Given** a request that creates, views, or changes a task, updates a profile, or performs admin account work  
**When** no authenticated identity is present  
**Then** the service rejects the request (unauthenticated)  

**Why:** Without a known person, ownership and admin power cannot be attributed, so work and account changes would be anonymous and unaccountable.

### BR-TASKDEMO-011 — Admin account work is denied to the user role

**Given** an authenticated identity whose role is user  
**When** they attempt to add an account, set a status to non-active, or soft-delete an account  
**Then** the service rejects the request  

**Why:** If any signed-in person could retire or create accounts, administrators could not be trusted as the only people who control who belongs in the service.

### User Management

### BR-TASKDEMO-020 — An account has exactly one of two roles: admin or user

**Given** an account  
**When** its role is recorded or used for authorisation  
**Then** the role is either admin or user, never both and never a third role  

**Why:** A third role or overlapping roles would make it unclear who may create people and who may only do their own work.

### BR-TASKDEMO-021 — Only an admin can add a new account

**Given** an authenticated admin  
**When** they add a new account  
**Then** a new account exists that did not exist before  

**Why:** The organisation must control who is admitted; open self-enrolment is not part of this capability.

### BR-TASKDEMO-022 — Only an admin can change an account from active to non-active

**Given** an authenticated admin and a target account that is active  
**When** the admin sets that account to non-active  
**Then** the target account is non-active  

**Why:** Taking someone out of day-to-day use is an administrative control, not a self-service status, so that access is withdrawn on purpose rather than by accident.

### BR-TASKDEMO-023 — Only an admin can soft-delete an account

**Given** an authenticated admin and a target account that has not already been soft-deleted  
**When** the admin deletes that account  
**Then** the account is soft-deleted, not removed as if it never existed  

**Why:** Retiring a person must remain reversible in records and auditable; a hard erase would hide who owned work and who was admitted.

### Profile

### BR-TASKDEMO-030 — An authenticated user can update their own profile

**Given** an authenticated identity whose role is user  
**When** they submit a profile update for themselves  
**Then** the service accepts a change to that identity's name and/or password only; role, status, and email are unchanged  

**Why:** People need to keep their own details current without waiting for an administrator for ordinary self-service changes.

### BR-TASKDEMO-031 — A user cannot update another account's profile

**Given** an authenticated identity whose role is user  
**When** they attempt to update a different account's profile  
**Then** the service rejects the request  

**Why:** Allowing one person to change another's details would let colleagues impersonate or sabotage each other.

### Task Ownership

### BR-TASKDEMO-040 — Creating a task assigns ownership to the authenticated user

**Given** an authenticated identity whose role is user  
**When** they successfully create a task  
**Then** that identity is the owner of the new task  

**Why:** Work must be attributable; without an owner, later visibility and responsibility cannot be decided per person.

### BR-TASKDEMO-041 — A user creates a task in todo

**Given** an authenticated identity whose role is user  
**When** they create a task  
**Then** the new task starts as todo and is owned by them  

**Why:** Ownership must not change the existing meaning of a new piece of work: it has been recorded, not started. Lifecycle detail remains in [task tracking](./task-tracking.md) (`BR-TASKDEMO-002`).

### BR-TASKDEMO-042 — A user may view and change only tasks they own

**Given** an authenticated identity whose role is user and a task they do not own  
**When** they attempt to view or change that task  
**Then** the service rejects the request as if that task did not exist  

**Why:** People must not see or alter colleagues' work by default. An admin may view any task by id; an admin who is not the owner still cannot change it. There is no task list.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| Unauthenticated task, profile, or admin account request | Rejected (HTTP 401) | `BR-TASKDEMO-010` |
| User role attempts to add, deactivate, or soft-delete an account | Rejected (HTTP 403) | `BR-TASKDEMO-011` |
| Admin creates a task, or a user updates another profile | Rejected (HTTP 403) | `BR-TASKDEMO-040`, `BR-TASKDEMO-031` |
| Account role is anything other than admin or user | Not a valid account role | `BR-TASKDEMO-020` |
| User updates their own profile | Name and/or password accepted; role, status, email locked | `BR-TASKDEMO-030` |
| User reads or changes someone else's task | Rejected as not found (HTTP 404) | `BR-TASKDEMO-042` |
| Admin reads another user's task by id | Allowed | Open question 6 |
| Admin starts, completes, or reschedules another user's task | Rejected as not found (HTTP 404) | Open question 11 |
| Non-active or soft-deleted account attempts login, or presents a token | Login denied; tokens revoked / request unauthenticated | Open question 2 |
| Admin deactivates or soft-deletes themselves or the last admin | Rejected | Open question 4 |
| Existing tasks with no owner at migration | `owner_id` set to the seeded admin | Open question 13 |

## Examples

| Input | Expected outcome |
|---|---|
| Admin Ada adds account for Budi with role user | A new user account for Budi exists |
| User Budi, authenticated, creates a task titled "Prepare report" | Task exists as todo, owner is Budi |
| User Citra, authenticated, requests Budi's task | Rejected (`BR-TASKDEMO-042`) |
| User Budi attempts to add an account for Dedi | Rejected (`BR-TASKDEMO-011`) |
| Admin Ada sets Budi from active to non-active | Budi is non-active; login is denied and Budi's tokens are revoked |
| Admin Ada soft-deletes Citra | Citra is soft-deleted; her email may be reused on a later account |
| User Budi updates Citra's profile | Rejected (`BR-TASKDEMO-031`) |
| Admin Ada `GET` Budi's task by id | Allowed |
| Admin Ada starts Budi's task | Rejected as not found |
| Admin Ada `POST /tasks` | Rejected (admin cannot create a task) |

## Known gaps

Not yet implemented — verify against code after implementation.

Today the service has no authentication or task ownership (`AGENTS.md` treats those as out of scope for the current task-tracking slice). Existing task rules `BR-TASKDEMO-001`–`009` do not assign an owner.

## Open questions

Product / PO should decide. Do not implement guesses.

1. Admin pertama dibuat lewat apa (seed/bootstrap)?  
   **Answer:** seed

2. User non-active atau soft-deleted: login ditolak? Token yang sudah terbit dicabut atau dibiarkan sampai expired?  
   **Answer:** login ditolak, token dicabut

3. Email boleh dipakai ulang setelah soft delete?  
   **Answer:** boleh

4. Admin boleh menonaktifkan/menghapus dirinya sendiri atau admin terakhir?  
   **Answer:** tidak boleh

5. Field profile yang boleh diubah user. Usulan (belum diputuskan): nama dan password saja; role, status, dan email terkunci.  
   **Answer:** nama dan password saja
   **Proposal on the table:** name and password only; role, status, and email locked.

6. Visibilitas task: user hanya lihat task sendiri (sudah menjadi `BR-TASKDEMO-042`). Admin boleh lihat semua atau tidak?  
   **Answer:** admin boleh lihat semua

7. Status non-active boleh kembali ke active?  
   **Answer:** boleh

Additional gaps that would otherwise become invented rules:

8. Role of an account that an admin just added (always user, or admin-chosen)?  
   **Answer:** bisa admin chosen

9. How the new account receives credentials (admin-set password, invite, other)?  
   **Answer:** admin-set password

Design decisions approved 2026-10-27 (delta discussion):

10. How is identity presented on HTTP?  
    **Answer:** login issues an opaque bearer token; the service stores it (hashed) so it can revoke; clients send `Authorization: Bearer`. `GET /health` and login stay unauthenticated.

11. May an admin create a task, or change another user's task?  
    **Answer:** no. Admin may `GET` any task by id. `POST /tasks` is user-role only. Start, complete, and reschedule remain owner-only, including when the caller is admin.

12. HTTP status when the service rejects?  
    **Answer:** 401 if there is no valid identity; 403 if the role is forbidden (user doing admin work, admin creating a task, user updating another profile); 404 if a user (or a non-owner admin) views or changes a task they do not own — same as unknown id.

13. Existing `tasks` rows with no owner?  
    **Answer:** migration sets `owner_id` to the seeded admin, then the column is required.

14. Does task JSON include owner?  
    **Answer:** yes — additive `ownerId` on success `data`. Clients do not send owner.

15. Does `TaskCompleted` include owner or actor?  
    **Answer:** no. Payload stays `taskId` and `completedAt`.

16. Is there a task list?  
    **Answer:** no. Admin “see all” means get-by-id, not a collection resource.

## Out of scope

- Self-registration or public sign-up
- Additional roles beyond admin and user
- Hard delete of accounts
- Transfer of task ownership, unowned tasks, or deleting tasks
- Changing an existing task's owner
- Password reset, email verification, MFA, SSO/OAuth, JWT, session cookies
- Listing, searching, or paging accounts or tasks (not in the invariants)
- Any change to task lifecycle rules `BR-TASKDEMO-001`–`009` except that create now requires an owner

## Related

- Existing flow reference: [`task-tracking.md`](./task-tracking.md) — task lifecycle this capability must not contradict; create still starts as todo (`BR-TASKDEMO-002`)
- Existing flow reference: `docs/api/openapi.yaml` — current unauthenticated task HTTP contract
- Existing flow reference: `AGENTS.md` — current product slice has no auth or ownership
- Sibling capabilities: [`task-tracking`](./task-tracking.md)
- Source document: capability-doc request "User Management + Authentication + Task Ownership" (2026-10-27)
