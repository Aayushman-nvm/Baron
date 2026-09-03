# Feature Gap Analysis — Rule-Enforced Workflow Orchestration Engine

> Mapping the design document requirements against what is currently built in Taskwondo.

---

## Summary

| # | Requirement | Status |
|---|-------------|--------|
| 1 | Ordered steps — cannot skip | ❌ Not built |
| 2 | Rule-based state transitions | ✅ Built |
| 3 | Step ownership / assignee | ⚠️ Item-level only |
| 4 | Document requirements per step | ❌ Not built |
| 5 | Deviation / exception flagging | ❌ Not built |
| 6 | Audit trail | ✅ Built |
| 7 | SLA monitoring | ✅ Built |
| 8 | Escalation rules | ✅ Built |
| 9 | Role-based access | ✅ Built |
| 10 | Portal for external parties | ✅ Built |
| 11 | Project templates | ❌ Not built |
| 12 | Vendor/supplier recommendations | ❌ Not built |
| 13 | Workflow deviation visualization | ❌ Not built |
| 14 | Freemium / plan limits | ✅ Built |

**5 fully built · 2 partial · 6 missing**

---

## What Is Already Built

### ✅ Rule-based state transitions
The workflow engine enforces valid `from → to` status transitions. Any attempt to move to a disallowed status returns a 409 error. Full CRUD exists for defining workflows, statuses, and transitions at both system and project level. This is the backbone of the "only valid transitions allowed" requirement.

### ✅ Audit trail
Every field change on a work item is recorded as a `WorkItemEvent` — actor, timestamp, old value, new value, event type. These are surfaced in an activity timeline in the UI and exposed through the portal (filtered to public-visible events only). Complete traceability is there.

### ✅ SLA monitoring
Sophisticated per-status time limits scoped by project, work item type, workflow, and priority. Tracks accumulated time with anti-gaming (elapsed carries forward if a status is re-entered). Computes `on_track` / `warning` / `breached` status. Supports business hours mode. Background worker runs periodic breach scans.

### ✅ Escalation rules
Named escalation lists with multiple levels. Each level has a threshold percentage of SLA consumed and targets users/teams to notify. On-call team integration is included. Mapped per work item type. Worker fires notifications when thresholds are crossed.

### ✅ Role-based access
Two tiers: global roles (`admin`, `user`) and project roles (`owner`, `admin`, `member`, `viewer`, `customer`). Customer role is specifically locked to the portal. Middleware enforces access at every route.

### ✅ Portal for external parties
Dedicated portal API namespace (`/api/v1/portal/...`) with: public queue listing, ticket creation, viewing and updating own tickets, comments, event history, and attachments. Frontend portal pages exist. Customers only see their own tickets — information leakage is explicitly prevented.

### ✅ Freemium / plan limits
`max_projects_per_user` global setting (default: 5) and `max_namespaces_per_user` (default: 1) with per-user overrides. Admins are exempt. Project creation is blocked when limits are reached. Frontend shows usage counts.

---

## What Is Partially Built

### ⚠️ Step ownership / assignee
Work items have an `assignee_id` field — one assignee per item. This satisfies "each task has an owner" at the item level. What the design doc implies but what is missing: **per-step ownership** — the idea that when a work item enters status X, it should automatically be owned by a specific person or role. That concept doesn't exist in the model.

---

## What Is Missing

### ❌ Ordered steps — cannot skip
Statuses have a display `position` field and transitions constrain movement, but **sequential enforcement does not exist**. You can define a transition that jumps from step 1 to step 5 and the engine will allow it. There is no "this step must be completed before the next one can start" concept. This is the core difference between a workflow *tracker* and a workflow *enforcer*.

**Complexity to add:** High. Requires schema changes (`is_sequential` on workflow), transition validation logic that checks position ordering, and UI changes to show blocked steps.

### ❌ Document / attachment requirements per step
No mechanism exists to say "status Y cannot be entered unless attachment of type X has been uploaded." Attachments can be uploaded freely at any time but there are no gates.

**Complexity to add:** Medium-high. Needs a new `workflow_status_requirements` table, service-layer checks on every status transition, and a UI for configuring and displaying requirement status.

### ❌ Deviation / exception flagging
When a workflow step is skipped or an action taken out of order, nothing records that as a deviation. There is no `workflow_deviation` event type, no flagging logic, and no UI for reviewing exceptions. This is the "even if payment is made under pressure, the system records it as an exception" requirement from the design doc — completely unbuilt.

**Complexity to add:** Medium. Mainly a new event type + service logic to detect and record deviations on transition. UI would show flagged items distinctly.

### ❌ Project templates
No template feature exists anywhere in the codebase — no database table, no API endpoint, no frontend page. When creating a project, there is no option to base it on a previous project's structure (workflows, queues, teams, SLA config).

**Complexity to add:** Medium. Schema for storing template snapshots of a project's configuration, an API to create-from-template, and a UI selection step during project creation. Does not touch core workflow logic.

### ❌ Vendor / supplier recommendations
Nothing exists here. No vendor profiles, no performance ratings, no recommendation engine, no category-based matching. This is entirely absent.

**Complexity to add:** High. Entirely new domain — new tables (vendors, ratings, categories, project-vendor links), new service logic, and a new UI section. Risk of scope creep if mixed with the core workflow system.

### ❌ Workflow deviation visualization (git-branch-style)
The activity timeline shows a linear history of events. There is no branch-style diagram showing where a workflow diverged from its intended path, what was skipped, and how it affected downstream steps.

**Complexity to add:** High. Requires the deviation flagging feature above as a prerequisite, plus a non-trivial frontend graph/tree rendering component. Likely the last thing to implement.

---

## Recommended Implementation Order

These are ordered by impact vs. risk. The top items build on existing infrastructure with minimal chance of breaking what's already working.

### 🟢 Low risk — add without touching core

| Feature | Why it's safe |
|---------|---------------|
| **Project templates** | New tables and endpoints only, no changes to existing workflow logic |
| **Deviation event type** | Add a new `WorkItemEvent` type — additive change, no existing code modified |

### 🟡 Medium risk — extend existing systems

| Feature | What it touches |
|---------|----------------|
| **Document requirements per step** | Adds a new table and a check inside the transition validation path — surgical but touches the transition logic |
| **Per-step ownership** | Schema addition to `workflow_status`, small service-layer change on transition |

### 🔴 High risk — significant new work

| Feature | Why it's hard |
|---------|--------------|
| **Sequential step enforcement** | Core workflow engine change — affects every status update across all projects |
| **Vendor/supplier recommendations** | Entirely new domain, risk of scope creep |
| **Workflow deviation visualization** | Depends on deviation flagging + complex frontend graph rendering |

---

## Notes on the Design Document's "New Additions"

1. **Recommend project template when category matches** — covered by the project templates gap above, plus a category field on projects (which also doesn't currently exist).
2. **Vendor recommendations with delivery time and ratings** — fully absent, high effort.
3. **Git-branch-style workflow deviation display** — fully absent, highest effort, depends on deviation flagging being built first.
4. **Freemium plan limits** — already built. Nothing to do here.
