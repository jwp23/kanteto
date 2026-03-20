# Changelog

Purpose: This file is a running log that tracks all notable changes, new features, and workflow updates for the project over time.
It also serves as a record of **completed beads issues** and significant workflow milestones.

> The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),  
> and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## Version Numbering Rules

We follow **Semantic Versioning (SemVer)** for all projects:

- **MAJOR (X.0.0):** Incompatible or breaking workflow or API changes.
- **MINOR (0.X.0):** New features, plan types, or template enhancements added in a backwards-compatible way.
- **PATCH (0.0.X):** Bug fixes, template corrections, or workflow refinements that don’t break existing functionality.

> For student or prototype projects:
>
> - Use **0.x.x** versions while iterating (pre-1.0).
> - Bump to **1.0.0** only when the core features are stable and production-ready.

---

## Issue Completion Logging

Significant beads issues should be recorded in the changelog when completed. Use this format:

---

### Issue Completion Entry Example

**Issue:** `AES-42`
**Type:** `feature`
**Status:** `closed`
**Summary:** Implemented secure login and registration flow with Firebase Auth.
**Commit Reference:** `feat: add login flow (Closes: AES-42)`
**Date:** 2025-10-24

---

This ensures transparency and traceability for all AI-executed workflows.

---

## [Unreleased]

## [0.6.1] - 2026-03-20

### Fixed

- Tasks not visible after `go install` of v0.6.0: embedded Dolt DSN used `database=kanteto` instead of `database=dolt`, creating an empty database instead of reading the existing one.

## [0.6.0] - 2026-03-20

### Changed

- **Breaking:** Replaced Dolt CLI shell-outs with embedded Dolt SDK (`github.com/dolthub/driver`). The `dolt` CLI is no longer a runtime dependency. All SQL operations run in-process via `database/sql`.
- Replaced debounced auto-sync with per-mutation commit + async push. Every mutation (add, complete, delete, edit) is committed synchronously in-process; push to remote fires asynchronously in the background.
- `store.New()` now accepts `*sql.DB` instead of a directory path.
- `sync.New()` now accepts `*sql.DB` instead of a directory path.
- Sync operations use `CALL DOLT_*` SQL procedures instead of CLI commands.
- Simplified TUI model: removed `autoSyncGen`, `autoSyncBusy`, `autoSyncDirty`, `autoSyncPushed` fields; replaced with single `pushBusy` flag.

### Added

- `Sync.Snapshot(msg)` method: fast in-process add + commit (no push).
- `Sync.PushRemote()` method: push-only (no add/commit) for async background use.
- `Syncer` interface expanded with `Snapshot` and `PushRemote` methods.
- `openDoltDB()` helper in `cmd/root.go` handles embedded database creation on first run.
- `toLocal()` helper in store to convert UTC timestamps from embedded driver to local time.
- Build tag `gms_pure_go` removes ICU/cgo dependency for clean `go install`.

### Removed

- `dolt` CLI runtime dependency — no longer needed on PATH.
- JSON parsing layer in store (`queryJSON`, `rowToTask`, `strVal`, `timeVal`, `intBool`, `tagsVal`).
- Debounce auto-sync machinery (`autoSyncTickMsg`, `autoSyncResultMsg`, `autoSyncFlushQuitMsg`, `scheduleAutoSync`, `doAutoSync`).
- `Store.initRepo()` and `Store.runDolt()` CLI helpers.
- `Sync.Dir()` method.

### Fixed

- "Cannot update manifest: database is read only" errors caused by concurrent CLI access during auto-sync.

## [0.5.0] - 2026-03-19

### Added

- Auto-sync with debounced background push: TUI automatically commits and pushes to Dolt remote after mutations, with debounce to batch rapid changes. Includes 243 new sync tests.

## [0.4.1] - 2026-03-16

### Fixed

- `kt migrate` no longer fails with "Dolt repo already exists" when `PersistentPreRunE` pre-initializes the Dolt directory. Migrate command now overrides `PersistentPreRunE` with a no-op.
- `kt migrate` handles older SQLite schemas missing `tags`, `profile`, or `recurrence_*` columns — introspects schema via `PRAGMA table_info` and substitutes defaults.
- `store.New()` now calls `ensureSchema()` unconditionally, so a user-initiated `dolt init` still gets the `tasks` table created.

### Changed

- `kt migrate` replaces `.dolt` directory guard with data-level check: errors if tasks already exist in Dolt unless `--force` flag is passed.
- README sync instructions corrected: data directory is `~/.local/share/kanteto/dolt/` (not `~/.local/share/kanteto/`). Added GitHub repo initialization step and second-machine clone instructions.

### Added

- `--force` flag on `kt migrate` to allow re-running migration when tasks already exist.
- `TestMigrate_ExistingEmptyRepo` and `TestMigrate_OldSchema` test cases.

## [0.4.0] - 2026-03-14

### Added

- TUI due-date alert: audible notification when a task's deadline passes while TUI is open. Tries `paplay`/`afplay`/`aplay` with configurable `sound_file`, falls back to terminal bell. One alert per tick, `alertedIDs` prevents re-firing (`kanteto-e0t`, `kanteto-5vl`, `kanteto-o6w`).
- `AlertPlayer` interface + `NewSoundPlayer()` for testable sound playback (`kanteto-o6w`).
- `SoundFile` config field in `config.toml` (`kanteto-5vl`).
- TUI tag management keybindings: `t` to add tag, `T` to remove tag on selected task (`kanteto-dut`).
- TUI reparse migration: `R` triggers confirmation flow to retroactively detect deadlines in undated tasks via `nlp.ExtractDeadline`. Preview count shown before executing (`kanteto-8rd`).
- `Service.Reparse()` method with `ReparseResult{Updated, Total}` — scans undated tasks, strips title, sets DueAt (`kanteto-8rd`).
- TUI sync push/pull: `p` to pull, `P` to push via async `tea.Cmd`. Includes `Syncer` interface, nil-syncer and no-remote guards, 3s auto-clear status (`kanteto-3ol`).
- 17 new tests across `tag_test.go`, `reparse_test.go`, `sync_test.go`, and `service_test.go`.

### Changed

- "Jump to today" keybinding reassigned from `t` to `.` (dot) — `t` now used for tag mode (`kanteto-dut`).
- `tui.New()` signature now accepts `AlertPlayer` parameter; `cmd/root.go` constructs `NewSoundPlayer(cfg.SoundFile)` (`kanteto-5vl`, `kanteto-o6w`).
- Tick handler now always calls `refreshData()` and runs due-alert detection, not just on midnight crossing (`kanteto-5vl`).
- Updated footer keybinding hints and help overlay for all new actions.

### Issues Closed

- `kanteto-e0t` (task): TDD tests for due-alert detection logic
- `kanteto-5vl` (feature): Track alerted task IDs and detect newly-due tasks on tick
- `kanteto-o6w` (feature): Wire sound output for due alerts
- `kanteto-dut` (feature): Tag management TUI keybindings
- `kanteto-8rd` (feature): Reparse migration TUI + service method
- `kanteto-3ol` (feature): Dolt sync push/pull TUI keybindings

## [0.3.0] - 2026-03-14

### Fixed

- `root.go` data directory now uses `dolt/` subdirectory, matching `migrate.go` — TUI sees migrated data (`kanteto-dxm`).
- Broken `make smoke` target — removed references to deleted CLI commands, now tests `--version` and `--help` only (`kanteto-i36`).
- DB query inside `View()` in `week.go` — moved `ListByDateRange` call to `refreshData()` pre-fetch, matching the `monthTasks` pattern (`kanteto-61q`).
- `ListAll` error silently discarded in `refreshData()` — now surfaced to TUI footer (`kanteto-bg7`).
- Missing `skipIfNoDolt` guard in `tui` and `task` test helpers — tests now skip gracefully without dolt (`kanteto-7w7`).

### Removed

- `remind_at` and `reminded` columns from Dolt schema.
- `RemindAt`, `Reminded` fields from `task.Task` model.
- `DefaultLeadTime` constant and `ReminderLeadTime` config field.
- `SetLeadTime` method, remind_at calculations in Add/AddRecurring/Complete/Snooze/SetDueAt.
- `ListDueReminders`, `MarkReminded` from Repository interface, Store, and ProfileStore.
- `GetDueReminders`, `MarkReminded` from Service.
- 3 tests for removed reminder code; net -172 lines (`kanteto-ouy`).

### Issues Closed

- `kanteto-dxm` (bug): Fix data directory path in root.go
- `kanteto-ouy` (task): Remove vestigial ReminderLeadTime and remind_at system
- `kanteto-wdm` (task): Bump version string to 0.3.0
- `kanteto-63h` (chore): Update go.mod Go version to 1.26.1
- `kanteto-rq0` (chore): Remove dead `config.Save()`
- `kanteto-i36` (bug): Fix broken make smoke target
- `kanteto-7w7` (bug): Add skipIfNoDolt guard to task and TUI test helpers
- `kanteto-bg7` (bug): Surface ListAll error in refreshData()
- `kanteto-61q` (bug): Move DB query out of View() in week.go
- `kanteto-ryg` (task): Add SQL special character round-trip test
- `kanteto-1z0` (chore): Remove unused RecurrenceNextDue field
- `kanteto-73s` (task): Update security.md and tests.md for v0.3.0

### Changed

- **BREAKING:** Replaced SQLite with Dolt CLI as sole datastore. Kanteto now requires `dolt` (v1.81.10+) and `git` on PATH.
- Store implementation (`internal/store/`) shells out to `dolt sql -q` with `-r json` for queries. Schema uses MySQL dialect (VARCHAR, TINYINT(1), JSON type).
- TUI `refreshData()` refactored from 4 queries to 1 `ListAll` call with in-memory bucketing — critical for Dolt CLI latency.
- `rawStore` type changed from `*store.Store` to `task.Repository` interface.
- `ListProfiles()` added to `task.Repository` interface and `ProfileStore`.

### Added

- `internal/sync/` package: thin wrapper around `dolt add/commit/push/pull/remote` commands.
- SQL special character round-trip test for `quote()` escaping (`kanteto-ryg`).

### Removed (cleanup session)

- Dead `config.Save()` function — no callers remain after CLI removal (`kanteto-rq0`).
- `RecurrenceNextDue` field from model, store, migration, and schema docs — unused caching optimization (`kanteto-1z0`).

### Changed (cleanup session)

- `go.mod` Go version bumped from 1.25.0 to 1.26.1 (`kanteto-63h`).
- `security.md` and `tests.md` updated for v0.3.0 Dolt-only architecture (`kanteto-73s`).
- `kt migrate` — one-time migration from SQLite (`kanteto.db`) to Dolt. Reads all tasks including completed, writes to `dolt/` subdirectory, creates initial commit.
- 78 new tests across doltstore, sync, and migration (225 total, up from 147).
- README: prerequisites section, migrate command docs, Dolt sync instructions.
- Updated infra.md and sbom.md for Dolt architecture.

### Removed

- CLI subcommands: `add`, `done`, `rm`, `edit`, `list`, `snooze`, `reparse`, `tag`, `profile`, `sync`, `daemon` (all task management now via TUI; only `kt migrate` remains).
- `internal/daemon/` package (orphaned after `cmd/daemon.go` removal).
- `rawStore` variable and profile/config wiring from `cmd/root.go`.

### Issues Closed

- `kanteto-1wy` (DoltStore implementation)
- `kanteto-1wd` (TUI single-fetch refactor)
- `kanteto-yzq` (Sync operations and CLI)
- `kanteto-sgo` (SQLite to Dolt migration)
- `kanteto-6k1` (Documentation updates)
- `kanteto-ona` (Phase 0 Dolt driver spike — retroactively closed)

## [0.2.6] - 2026-03-13

### Fixed

- TUI date not updating when left open overnight: added 1-minute tick that detects midnight crossover and auto-advances viewDate if the user was viewing "today" (`kanteto-9fe`).

### Issues Closed

- `kanteto-9fe`

## [0.2.5] - 2026-03-10

### Added

- Week view day cursor navigation (Story 13): `j/k` and arrow keys move a cursor across the 7 day columns, `Enter` drills into day view for the selected day, `h/l` still shifts by week (`kanteto-477`).
- Bracket + inverted-style highlight on selected week column header.
- Week-view-specific footer keybindings and updated help overlay.
- 8 new week cursor tests mirroring month_test.go pattern (`kanteto-2w9`).

### Issues Closed

- `kanteto-477`, `kanteto-9h5`, `kanteto-cuw`, `kanteto-61x`, `kanteto-2w9`

## [0.2.4] - 2026-03-06

### Fixed

- Version string corrected from `0.2.2` → `0.2.3` in `cmd/root.go` (`kanteto-4hl`).
- Week view N+1 query pattern: replaced 7 per-day `ListByDateRange` calls with single query + map bucketing (`kanteto-pms`).

### Added

- NLP bare weekday parsing: `"friday"`, `"this mon"`, `"this friday"` etc. now resolve to next occurrence (`kanteto-pms`).
- 6 store-level tests: ListUndated, ListOverdue, ListOverdueAsOf, ListDueReminders, MarkReminded, Update (`kanteto-6w8`).
- 2 CLI recurring tests: happy path + invalid pattern (`kanteto-6w8`).
- 6 TUI render tests: day view sections/empty/cursor prefix, week view header/tasks/empty (`kanteto-6w8`).
- Documentation comment on `RecurrenceNextDue` dead column (`kanteto-4hl`).

### Coverage Improvements

- `internal/store`: 62.7% → 87.3%
- `internal/tui`: 53.5% → 84.2%
- `internal/nlp`: 89.2% → 90.9%
- `cmd`: 58.8% → 62.8%

### Issues Closed

- `kanteto-4hl`, `kanteto-6w8`, `kanteto-pms`

## [0.2.3] - 2026-03-06

### Added

- Daemon lifecycle management: `kt daemon start`, `kt daemon stop`, `kt daemon status` subcommands (`kanteto-imw`).
- `PIDPath()`, `IsRunning()`, `Stop()` functions with duplicate instance prevention and `syscall.Signal(0)` process checking.
- Context-based shutdown with `signal.NotifyContext` for SIGINT/SIGTERM handling.
- `SoundPlayer` interface for testable reminder playback (`kanteto-d7k`).
- 11 daemon unit tests covering PID lifecycle, duplicate prevention, reminder firing, context cancellation (63.3% coverage) (`kanteto-d7k`).
- 3 daemon integration tests: reminder flow, PID cleanup, concurrent SQLite WAL access (`kanteto-4hh`).
- 14 TUI tests across 3 new files: day view sections/cursor/empty state, keybindings (j/k/space/x/view switching/time nav/help), add/snooze input modes (53.5% coverage) (`kanteto-s01`).
- `Makefile` with build, test, vet, cover, smoke, clean, all targets (`kanteto-6iw`).

### Issues Closed

- `kanteto-imw`, `kanteto-d7k`, `kanteto-s01`, `kanteto-6iw`, `kanteto-4hh`

## [0.2.2] - 2026-03-06

### Fixed

- Daemon PID file permissions tightened from 0644 to 0600 (`kanteto-fe6`).
- TUI now captures and renders errors from Complete/Delete/Add operations in red footer text (`kanteto-ici`).
- Month view eliminated N+1 queries — single `ListByDateRange` call with map lookup (`kanteto-y28`).
- `infra.md` build command corrected to `go build -o kt ./cmd/kt` (`kanteto-fnj`).
- Consolidated duplicate `defaultLeadTime` constant — exported from `config` package (`kanteto-zyq`).

### Added

- `kt edit [id] --title --by --every` command for editing tasks (`kanteto-n20`).
- TUI snooze prompt: press `s` to snooze selected task with duration input (`kanteto-6o6`).
- `--version` flag on root command, prints `kt version 0.2.2` (`kanteto-ak1`).
- `Example` fields on all CLI commands for `--help` output (`kanteto-ak1`).
- Friendlier error messages in `resolveID` with guidance on next steps (`kanteto-ak1`).
- 11 new CLI command tests: add, done, snooze, rm, edit (`kanteto-oja`).
- 6 new integration tests: full lifecycle, recurring advance, snooze, date range, overdue, edit workflow (`kanteto-286`).

### Issues Closed

- `kanteto-fe6`, `kanteto-fnj`, `kanteto-zyq`, `kanteto-ici`, `kanteto-y28`, `kanteto-n20`, `kanteto-6o6`, `kanteto-ak1`, `kanteto-oja`, `kanteto-286`, `kanteto-va6`

## [Unreleased]

### Changed

- Rewrote `README.md` with full CLI/TUI usage documentation, install instructions, keybinding tables, and configuration reference.
- Moved binary entry point from root `main.go` to `cmd/kt/main.go` so `go install .../cmd/kt@latest` produces a `kt` binary.
- Updated `.gitignore` to use `/kt` (root-only) instead of `kt` which was blocking `cmd/kt/` directory.

### Housekeeping

- Closed 22 stale beads issues already implemented in codebase (identified by Quartermaster backlog review): 5 epics, 6 foundation/data, 3 business logic, 3 CLI, 4 TUI, 1 daemon.
- Created 2 split-off issues for unfinished slices: `kanteto-n20` (kt edit command), `kanteto-6o6` (TUI snooze prompt).
- Backlog reduced from 33 to 12 open issues.

## [0.2.1] - 2026-03-03

### Added

- `kt list --next` and `kt list --prev` CLI flags for navigating forward/backward by one day (Story 5 completion).
- `ListOverdueAsOf(time.Time)` method on Repository, Store, and Service for date-relative overdue queries.
- Month view cursor tracking with `j/k` (by week), `←/→` (by day) navigation in TUI.
- Press `Enter` in month view to drill down into day view for the selected date (Story 8 completion).
- Selected day highlighted with `[day]` and inverted style in month grid.
- Updated TUI footer and help overlay with month-view-specific keybindings.
- 18 new tests across `cmd/list_test.go` and `internal/tui/month_test.go`.

### Issues Closed

- `kanteto-hqe`: Add --next/--prev flags to kt list CLI command
- `kanteto-qyg`: Implement month view drill-down on Enter

## [0.2.0] - 2026-03-03

### Added

- Go project scaffolding: `go mod init`, Cobra root command, directory structure (`cmd/`, `internal/task/`, `internal/store/`, `internal/tui/`, `internal/daemon/`, `internal/nlp/`, `internal/config/`).
- Task domain model with ULID-based ID generation (`internal/task/model.go`, `internal/task/id.go`).
- XDG-compliant config package loading from `~/.config/kanteto/config.toml` (`internal/config/`).
- NLP date parser supporting natural language dates (`march 11`, `tomorrow at 3pm`, `next friday`, `in 5 minutes`, `at 3pm`) and durations (`1 hour`, `30m`, `2 days`) (`internal/nlp/`).
- SQLite store with pure-Go `modernc.org/sqlite`, WAL mode, auto-migration, CRUD, date-range queries, and reminder queries (`internal/store/`).
- Task service layer with Repository interface: Add, Complete, Delete, Snooze, ListAll, ListToday, ListOverdue, ListUndated, reminder management (`internal/task/service.go`).
- Recurring task engine: daily/weekly/weekdays/<day> patterns, auto-advance on completion (`internal/task/recurrence.go`).
- CLI commands: `kt add` (with `--by` and `--every` flags), `kt done`, `kt rm`, `kt snooze`, `kt list`, `kt daemon`.
- Bubble Tea TUI with day/week/month views, j/k navigation, space to complete, a to add inline with NLP parsing, x to delete, h/l time navigation, d/w/m view switching, ? help overlay.
- Urgency color gradient: white (>2h) → yellow (2h) → amber (1h) → orange (30m) → red (overdue).
- ANYTIME section for undated tasks in both CLI list and TUI day view.
- Reminder daemon checking every 30s with audible alerts via paplay/afplay/aplay.
- `ExtractDeadline` for inline TUI input: "test kt in 5 minutes" → title + deadline.

### Issues Closed

- `kanteto-14p` (epic): Foundation — project scaffolding and config
- `kanteto-14p.1`: Initialize Go module and project structure
- `kanteto-14p.2`: Implement task domain model with ULID
- `kanteto-14p.3`: Implement XDG config package
- `kanteto-14p.4`: Implement Cobra CLI skeleton with root command
- `kanteto-9xs`: Implement NLP date parsing
- `kanteto-nl4`: Implement SQLite store with migrations
- `kanteto-90e`: Implement task service layer
- `kanteto-5kz`: Implement kt add command with NLP
- `kanteto-1xd`: Implement kt done and kt rm commands
- `kanteto-r8s`: Implement kt snooze command
- `kanteto-pf3`: Implement kt list CLI command
- `kanteto-iwx`: Implement recurring tasks
- `kanteto-2o2`: Implement reminder daemon
- `kanteto-f8b`: Implement Bubble Tea TUI with day view
- `kanteto-dvf`: Implement week and month TUI views
- `kanteto-0nt`: Implement urgency color gradient
- `kanteto-wwp`: Implement TUI inline add and help overlay
- `kanteto-dki`: Implement time navigation in TUI

---

### Added

- Initialized Dolt-backed beads database with `bd init --force` and restored epic `kanteto-14p` from JSONL backup.
- Installed git hooks via `bd hooks install` (`core.hooksPath = .beads/hooks/`): pre-commit, post-merge, pre-push, post-checkout, prepare-commit-msg.
- Installed Claude Code hooks (SessionStart, PreCompact) via `bd setup claude --project`.

### Changed

- Migrated from `.claude/implementation/` and `features.json` to beads (`bd`) for issue tracking.
- Updated `workflow.md` to use beads CLI commands for planning, execution, and status management.
- Clarified changelog role in tracking **issue completions** and **workflow milestones**.

### Added

- Introduced beads (`bd`) for centralized issue tracking with priorities, dependencies, and labels.
- Added branching strategy and PR workflow documentation to `workflow.md`.
- Enhanced multi-agent coordination with `--actor` and `--assignee` flags.

### Deprecated

- Removed `.claude/implementation/` directory structure — now handled by beads.

---

## [0.1.1] - 2025-09-15

### Added

- Introduced initial autonomous workflow logic:
  - Beads (`bd`) CLI for issue tracking
  - Issue types: bug, feature, task, epic, chore
  - Status management: open, in_progress, blocked, deferred, closed
- Updated `workflow.md` and `claude.md` to define issue-based planning and execution.

### Changed

- Revised `tests.md` to support automatic test execution after each feature step.
- Added changelog integration rules for issue completions.

---

## [0.1.0] - 2025-08-31

### Added

- Created initial set of Markdown context files (`claude.md`, `prd.md`, `infra.md`, `workflow.md`, `security.md`, `sbom.md`, `tests.md`).
- Added `changelog.md` to track project history.
- Added `first_prompt.md` as interactive setup guide for template population.
- Defined examples for both local Python applications and Next.js + Supabase applications to guide new students.

### Notes

- This is the first structured version of the project templates.
- Future releases will focus on workflow automation, changelog integration, and feature-based plan versioning.
