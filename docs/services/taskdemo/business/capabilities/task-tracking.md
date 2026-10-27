---
id: task-tracking
service: taskdemo
status: implemented
rules:
  - BR-TASKDEMO-001
  - BR-TASKDEMO-002
  - BR-TASKDEMO-003
  - BR-TASKDEMO-004
  - BR-TASKDEMO-005
  - BR-TASKDEMO-006
  - BR-TASKDEMO-007
  - BR-TASKDEMO-008
  - BR-TASKDEMO-009
updated: 2026-10-27
implementation:
  - domain/aggregate
  - application/usecase
---

# Task tracking

## Purpose

Record work, track progress, and identify overdue tasks without changing completed history.

## Ubiquitous language

A task is a unit of work with a title, optional due date, and status todo, in_progress, or done.

## Actors and triggers

A general user creates, starts, completes, reschedules, or views a task through the HTTP API.

## Rules

### BR-TASKDEMO-001 — A task must have a title of 1 to 100 characters after trimming

**Given** a create request  
**When** the title is empty, whitespace-only, or longer than 100 Unicode code points after trimming  
**Then** the service rejects the request  
**Why** empty titles cannot identify work

Local ID: BR-01.

### BR-TASKDEMO-002 — A newly created task starts as todo

**Given** a valid title  
**When** a task is created  
**Then** its status is todo  
**Why** work has not started yet

Local ID: BR-02.

### BR-TASKDEMO-003 — A due date cannot be in the past when set

**Given** a create or reschedule request  
**When** dueDate is strictly before current system time  
**Then** the request is rejected  
**Why** a past deadline cannot be planned

Local ID: BR-03.

### BR-TASKDEMO-004 — A task can be started only from todo

**Given** a task  
**When** start is requested from a status other than todo  
**Then** the transition is rejected  
**Why** progress must follow the agreed lifecycle

Local ID: BR-04.

### BR-TASKDEMO-005 — A task can be completed only from in progress

**Given** a task that is still todo  
**When** complete is requested  
**Then** the transition is rejected and no completion event is published  
**Why** work cannot finish before it starts

Local ID: BR-05.

### BR-TASKDEMO-006 — A done task cannot be modified

**Given** a done task  
**When** start, complete, or reschedule is requested  
**Then** the request is rejected  
**Why** completed history must stay trustworthy

Local ID: BR-06.

### BR-TASKDEMO-007 — A task is overdue only after its due date while not done

**Given** a task with a due date  
**When** current time is strictly after dueDate and status is not done  
**Then** details report overdue  
**Why** operators need a reliable overdue signal

Local ID: BR-07.

### BR-TASKDEMO-008 — The system owns created and completed timestamps

**Given** a create or complete request  
**When** the operation succeeds  
**Then** createdAt or completedAt is assigned by the service  
**Why** clients must not rewrite history

Local ID: BR-08.

### BR-TASKDEMO-009 — Completing a task publishes TaskCompleted

**Given** an in-progress task  
**When** completion is saved  
**Then** a TaskCompleted event with taskId and completedAt is published  
**Why** other modules can react without coupling

Local ID: BR-09.

## Edge cases

A due date equal to now is allowed. Null dueDate on reschedule clears the date. Missing dueDate on reschedule is invalid.

## Examples

Create "Prepare report", start it, then complete it. The task is done and a completion event is accepted.

## Known gaps

Live PostgreSQL and RabbitMQ checks are optional and skip when Docker resources are not running.

## Open questions

None.

## Related

- API contract in `docs/api/openapi.yaml`
- Domain package `domain/aggregate`
