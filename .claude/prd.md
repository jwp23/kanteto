# Product Requirements Document

Purpose: This file defines what we are building and for whom, focusing on the project's features, goals, and user experience.

---

## 1. The Big Picture

- **Project Name:** Kanteto
- **One-Sentence Summary:** A TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time. The only CLI subcommand is `kt migrate` for one-time SQLite→Dolt migration.
- **Who is this for?** Anyone who makes small commitments throughout the day — "I'll send that by 4pm," "remind me to follow up Friday" — and needs a fast, keyboard-driven way to track them from the terminal.
- **What this app will NOT do:**
  - It will not replace full project management tools (Jira, Linear, Asana).
  - It will not send background notifications or OS-level reminders. The only alert is the in-TUI audible bell (Story 17), which fires only while the TUI is open.
  - It will not have a web or mobile interface.

---

## 2. The Features

- **Story 1:** As a user, I want to add a task with a natural language deadline so that I can quickly capture commitments without thinking about date formats.
  - Press `a` in the TUI to open the inline add prompt.
  - Type `Call dentist by march 11` or `Buy groceries` (no deadline).

- **Story 2:** As a user, I want to create recurring tasks so that I can track things I do on a regular schedule.
  - Inline add supports recurrence patterns: `Send weekly update every weekdays at 4pm`.

- **Story 3:** ~~Removed.~~ The audible reminder daemon was removed in v0.3.0.

- **Story 4:** As a user, I want to view my tasks by day, week, or month so that I can plan my time across different horizons.
  - Day view: OVERDUE / TODAY / UPCOMING sections.
  - Week view: 7-column grid with tasks under each day.
  - Month view: Calendar grid with task counts per day.

- **Story 5:** As a user, I want to navigate forward and backward in time so that I can see what's coming up or review past tasks.
  - `h/l` or arrow keys in TUI.

- **Story 6:** As a user, I want to mark tasks as done, snooze them, or delete them so that I can manage my list quickly.
  - `space` to complete, `s` to snooze, `x` to delete in TUI.
  - Completing a recurring task advances it to the next occurrence.

- **Story 7:** As a user, I want tasks to visually shift from white to red as their deadline approaches so that I can see urgency at a glance.
  - Continuous gradient: white (>2h) -> yellow (2h) -> amber (1h) -> orange (30m) -> red (overdue).

- **Story 8:** As a user, I want an interactive TUI that launches with `kt` so that I can browse, add, and manage tasks without remembering commands.
  - Keyboard-driven: `j/k` to move, `d/w/m` to switch views, `a` to add, `space` to complete.

### NLP & Task Editing

- **Story 9:** As a user, I want to type deadlines naturally (e.g., "today 4pm", "friday 12pm") without needing the word "at" so that task entry matches how I actually talk.
    * Feature name: `nlp_bare_time`

- **Story 10:** As a user, I want to add tasks like "review doc Friday 2pm" (date at the end, no "by" or "at" keyword) so that deadlines are detected regardless of phrasing.
    * Feature name: `nlp_trailing_date`

- **Story 11:** As a user, I want to press `e` in the TUI to edit a task's deadline inline so that I can reschedule without leaving the interface.
    * Feature name: `tui_edit_time`

- **Story 12:** As a user, I want to re-parse existing undated tasks so that tasks created before the NLP fix get their deadlines detected retroactively.
    * Feature name: `reparse_migration`

- **Story 13:** As a user, I want to move a cursor across individual days in week view so that I can inspect tasks due on a specific day without leaving the TUI.
    * Feature name: `week_view_day_cursor`
    * `j/k` and `left/right` arrow keys move the column cursor across the 7 days (Sunday through Saturday).
    * `h/l` advance or retreat by one full week, keeping the cursor on the same day-of-week position.
    * Pressing `Enter` on a highlighted column drills into day view for that date — consistent with the existing month-view drill-down behavior described in Story 8.
    * The active column is highlighted with brackets and inverted styling.

### Tags & Profiles

- **Story 14:** As a user, I want to tag tasks so that I can categorize and filter them.
    * Feature name: `task_tags`
    * Tags are managed via the TUI.
    * Tags display in dim brackets in TUI day view.

- **Story 15:** As a user, I want to switch between profiles (e.g., "work" vs "personal") so that I can scope my task views.
    * Feature name: `task_profiles`
    * Profile switching is managed via config (`active_profile` in `config.toml`).
    * TUI shows active profile in header for non-default profiles.

### Sync

- **Story 16:** As a user, I want to sync my tasks across machines using Dolt so that I have the same task list everywhere.
    * Feature name: `dolt_sync`
    * Kanteto stores data in a Dolt repository at `~/.local/share/kanteto/`.
    * Sync is performed via the `dolt` CLI directly in the data directory (no `kt sync` command).
    * `kt migrate` — one-time migration from SQLite to Dolt backend.

### Alerts

- **Story 17:** As a user, I want to hear an audible alert when a task's deadline passes while the TUI is open so that I notice time-sensitive commitments without constantly watching the screen.
    * Feature name: `tui_due_alert`
    * The alert fires only while the TUI is running — no background daemon, no OS notification.
    * On each 1-minute tick, the TUI checks for tasks whose `due_at` just crossed "now". Any newly-due task triggers a terminal bell (`\a`).
    * Each task rings the bell at most once per session (tracked in-memory, resets on exit).
    * Completed or deleted tasks don't trigger the bell.
    * No configuration required. Users who want silence disable the bell in their terminal emulator — standard terminal behavior.

---

## 3. The Look and Feel

- **Overall Style:** Clean, minimal, fast. The TUI should feel like lazygit or htop — responsive and keyboard-driven.
- **Main Colors:** Default terminal colors with urgency gradient (white -> yellow -> amber -> red) for approaching deadlines. Overdue tasks are bold red. Completed tasks are dimmed with a checkmark.
- **Key Screens:**
  - **Day View (default):** Header bar with date, current view indicator, and active profile (if non-default). Three sections: OVERDUE (red), TODAY, UPCOMING. Keybinding footer. Tasks display tags in dim brackets.
  - **Week View:** 7-column grid (Sunday-Saturday). Current day column highlighted by default. `j/k` or `left/right` moves a cursor column across the 7 days; `h/l` shifts the entire week forward or backward. Press `Enter` on any column to drill into day view for that date.
  - **Month View:** Calendar grid with day numbers and task counts. Current day highlighted. Press Enter on a day to drill into day view.
  - **Help Overlay:** `?` shows all keybindings in a centered overlay.
  - **Inline Add Prompt:** `a` opens a text input at the bottom for quick task entry with NLP parsing.
