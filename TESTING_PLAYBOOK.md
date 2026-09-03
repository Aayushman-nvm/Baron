# Manual Testing Playbook

Step-by-step reproduction guide for every feature listed as "already built" in the gap analysis.
Run these in order — each section depends on the previous one having succeeded.

**Prerequisites:** API running on `:8080`, frontend on `:5173`, Neon DB connected.

---

## Setup: Create test accounts

You need three accounts to test all role scenarios.

### Admin account (already exists from .env seed)
- Email: `admin@baron`
- Password: `baron`

### Regular member account
1. Go to `http://localhost:5173/register`
2. Register with email `member@baron.test`, password `Test1234`

### Customer account
1. Register with email `customer@baron.test`, password `Test1234`

---

## Feature 1 — Role-based access

**What to verify:** Different roles see different things and can't access what they shouldn't.

### 1a. Admin role
1. Log in as `admin@baron`
2. Confirm the left sidebar shows a **System Settings** section (gear icon at bottom)
3. Go to System Settings → Directory → confirm you can see all users
4. Go to System Settings → Workflows → confirm you can create/edit global workflows

### 1b. Project member role
1. Log in as `member@baron.test`
2. Create a new project: click **New Project**, name it `VENDOR`, key `VENDOR`
3. Confirm you land on the project overview
4. Go to project Settings → Members → invite `customer@baron.test` with role **Customer**

### 1c. Customer role
1. Log out, log in as `customer@baron.test`
2. Accept the invite (check the invite link from the member account, or go to `/invites/<code>`)
3. Confirm: clicking the VENDOR project takes you to a **Support** page, NOT the full project internals
4. Confirm: the customer sees no Items, Milestones, Queues, or Settings tabs — only Support
5. Confirm: the customer can see "My Tickets" in the support section

**Bug to watch for:** Customer seeing a "project not found" page instead of the support page (known intermittent issue noted in your bug report — note if you can reproduce it).

---

## Feature 2 — Rule-based state transitions

**What to verify:** Invalid status jumps are rejected. Only defined transitions are allowed.

1. Log in as `member@baron.test`, go to project VENDOR
2. Go to **Settings → Workflows** tab
3. You should see a default workflow assigned. Click to view it.
4. Note the statuses and their defined transitions (e.g. `To Do → In Progress → Done`)
5. Go to **Items**, create a new task: title `Transition Test`, type `task`
6. Open the item. Status should be the workflow's initial status (e.g. `To Do`)
7. Try to change status: click the status field — confirm only **valid next statuses** appear (not all statuses)
8. Move it to `In Progress`, save
9. Now try to move it back — confirm whether backward transitions are allowed or blocked based on your workflow definition

**What to note:** If you see ALL statuses available regardless of transitions, the transition enforcement may have a UI bug (the backend would still reject it, but the UI shouldn't show invalid options).

---

## Feature 3 — Audit trail

**What to verify:** Every change is recorded with who did it and when.

1. Open the `Transition Test` item from Feature 2
2. Change the assignee to yourself
3. Change the priority from `medium` to `high`
4. Add a comment: "Testing audit trail"
5. Click the **Activity** tab on the item detail page
6. Confirm you see entries for:
   - Status change (To Do → In Progress)
   - Assignee change
   - Priority change
   - Comment added
7. Each entry should show the actor name and timestamp

**What to note:** If any of those changes don't appear in the activity log, that's a bug.

---

## Feature 4 — SLA monitoring

**What to verify:** SLA targets can be configured, and items show SLA status.

### 4a. Configure an SLA target
1. Log in as `member@baron.test`, go to project VENDOR
2. Go to **Settings → SLA** (in the project settings sidebar)
3. Create an SLA target:
   - Type: `task`
   - Status: `In Progress`
   - Priority: `high`
   - Target: `1 hour` (3600 seconds)
   - Calendar: `24x7`
4. Save

### 4b. Verify SLA indicator on item
1. Create a new task with priority `high`, assign it, move it to `In Progress`
2. Open the item — look for an SLA indicator (progress bar or status badge near the top)
3. Confirm it shows `on_track` with the remaining time

### 4c. Verify SLA status in list
1. Go back to the Items list
2. Look for an SLA column or indicator on high-priority in-progress items
3. Confirm the SLA status is visible

**What to note:** If no SLA indicator appears even after configuring a target and moving an item to the relevant status, that's a bug.

---

## Feature 5 — Escalation rules

**What to verify:** Escalation lists can be created and mapped to item types.

1. Go to project VENDOR → Settings → **Escalation** (in the project settings sidebar)
2. Create an escalation list named `Vendor Escalation`
3. Add a level:
   - Threshold: `50%` (fires at 50% of SLA time consumed)
   - Add yourself as a user to notify
4. Save the list
5. Go to **Escalation Mappings** (same page or a tab within it)
6. Map `task` type → `Vendor Escalation` list
7. Save

**What to note:** The actual email notification won't fire because the worker isn't running (no NATS). But you can verify the configuration saves and persists by refreshing the page and confirming the mapping is still there.

---

## Feature 6 — Portal (external party access)

**What to verify:** Customer can create and track tickets through the portal.

### 6a. Set up a public queue (required for portal ticket creation)
1. Log in as `member@baron.test`, go to project VENDOR
2. Go to **Queues** in the left sidebar
3. Create a queue: name `Support Queue`, type `support`, check **Public** (make it visible to customers)
4. Optionally add a category: `Billing`, `Technical`

### 6b. Customer creates a ticket
1. Log in as `customer@baron.test`
2. Navigate to the VENDOR project → Support section
3. Confirm "My Tickets" page loads with a **New Ticket** button
4. Click New Ticket, fill in:
   - Title: `Cannot access my account`
   - Description: `I get an error when logging in`
   - Priority: `high`
5. Submit — confirm the ticket appears in the list with a display ID (e.g. `VENDOR-1`)

### 6c. Customer views ticket history
1. Click the ticket → confirm you see the detail page with status, comments tab, and activity tab
2. Add a comment: "This is still happening"
3. Confirm the comment appears

### 6d. Member responds
1. Log in as `member@baron.test`, go to Items
2. Find the ticket `VENDOR-1` — confirm it appears (it should, since the member can see all items)
3. Change its status to `In Progress`
4. Add an internal comment (if visibility options exist)

### 6e. Customer sees the update
1. Log back in as `customer@baron.test`
2. Open the ticket — confirm the status shows `In Progress`
3. Check the Activity tab — confirm the status change is visible

---

## Feature 7 — Freemium / plan limits

**What to verify:** Project creation is blocked once the limit is reached.

1. Log in as a non-admin account (e.g. `member@baron.test`)
2. Go to System Settings (as admin) → confirm `max_projects_per_user` is set to `5` (the default)
3. As `member@baron.test`, create projects until you hit the limit
4. On the 6th creation attempt, confirm you get an error message saying the limit has been reached
5. Log in as admin — confirm admin can still create projects beyond the limit (admins are exempt)

**Shortcut:** You can lower the limit to `1` in System Settings → General (as admin), then try to create a second project as a regular user.

---

## Quick Sanity Checks

These don't need detailed steps — just verify they load without errors:

| Check | Where |
|-------|-------|
| Project overview loads | `/{namespace}/projects/VENDOR` |
| Items list loads with filters | Items tab → try filtering by status, priority, type |
| Milestones page loads | Milestones tab → create one, verify it shows |
| Workflow settings page loads | Settings → Workflows |
| System settings accessible to admin only | Log in as `member@baron.test` → try navigating to `/system` → should redirect or 403 |
| Dark mode toggle works | Top-right profile menu → appearance |
| Cmd+K command palette works | Press `Ctrl+K` → type a project name → confirm it navigates |

---

## Bug Report Template

When you find a bug, record it like this:

```
### Bug: [short name]

**Feature:** (which feature above)
**Steps to reproduce:**
1.
2.
3.
**Expected:** 
**Actual:** 
**Severity:** Critical / High / Medium / Low
**Notes:** (screenshots, error messages, console output)
```

---

## Known Issues (from previous testing)

These are already documented — don't spend time on them, just confirm if they're still present:

1. Overview tab not highlighted when navigating to a project *(fixed in this session)*
2. Long email IDs overflow profile popup *(fixed in this session)*
3. Customer role intermittently shows "project not found" instead of support page
4. @mention in work item descriptions returns no results
5. Feedback type badge gets cropped in the items list on large screens *(fixed in this session)*
