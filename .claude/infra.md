# Infrastructure Blueprint

Purpose: This file describes the project's technical foundation, including the programming language, coding standards, and how to build and run the application.

---

## What We're Building

- **Programming Language:** Go 1.26 (latest stable)
- **Main Framework/Tool:** Cobra (CLI), Bubble Tea (TUI), Lip Gloss (styling)
- **A Quick Summary:** A single-binary CLI + TUI tool for tracking micro-tasks with recurring schedules and audible reminders. See `prd.md` for full feature details.

---

## How to Run it on Your Computer

- **Installation Command:** `go mod download`
- **Build Command:** `go build -o kt .`
- **Run:** `./kt` (TUI) or `./kt --help` (CLI)
- **Run Tests:** `go test ./... -v`

---

## Project Architecture & Conventions

- **Framework:** Go with Cobra (CLI routing), Bubble Tea (TUI framework), Lip Gloss (terminal styling)
- **Directory Structure:**
  - **`cmd/`:** Cobra command definitions. One file per command (`add.go`, `done.go`, `list.go`, `daemon.go`). `root.go` is the entry point that launches the TUI when called with no subcommand.
  - **`internal/task/`:** Domain model (`model.go`), business logic (`service.go`), recurrence computation (`recurrence.go`), and ULID generation (`id.go`).
  - **`internal/store/`:** SQLite repository. All database queries and schema migrations live here.
  - **`internal/tui/`:** Bubble Tea models and views. `app.go` is the main model; `day.go`, `week.go`, `month.go` are view renderers; `styles.go` defines the urgency gradient and Lip Gloss styles.
  - **`internal/daemon/`:** Background reminder process. Wakes every 30 seconds, checks for due reminders, plays sound.
  - **`internal/nlp/`:** Natural language date parsing. Handles "march 11", "tomorrow", "weekdays at 4pm", etc.
  - **`internal/config/`:** XDG-compliant configuration loading from `~/.config/kanteto/config.toml`.
- **Key Architectural Decision:** CLI, TUI, and daemon all share a single `task.Service` layer backed by the same SQLite database. No duplicated logic between interfaces.

---

## Code Generation Style Guide

When writing or modifying code, adhere to the following standards:

- **Variable Naming:** Go conventions — `camelCase` for unexported, `PascalCase` for exported.
- **File Naming:** All lowercase with underscores where needed (e.g., `service_test.go`).
- **Comments:** Only where logic is not self-evident. No redundant godoc on obvious functions.
- **Linting:** Run `go vet ./...` and `staticcheck ./...` before committing.
- **Constants:** `PascalCase` for exported, `camelCase` for unexported (Go convention).
- **Error Handling:** Return errors up the stack. Do not silently swallow errors. Use `fmt.Errorf` with `%w` for wrapping.
- **Testing:** TDD — write failing tests first, then implement. Use table-driven tests where appropriate.

---

## Where it Lives

- **Hosting Provider:** Local computer only. Kanteto is a CLI tool, not a networked service.
- **External Services:** None. Kanteto has no network dependencies.
- **Distribution:** Single binary built with `go build`. No CGO required (pure-Go SQLite via `modernc.org/sqlite`). Cross-compilation via `GOOS`/`GOARCH` environment variables.

---

## Where Your Data is Stored

- **Data Storage Method:** SQLite database at `~/.local/share/kanteto/kanteto.db` (XDG-compliant, respects `XDG_DATA_HOME`).
- **Important Notes:**
  - SQLite WAL mode is enabled for concurrent access between daemon and CLI/TUI.
  - The database is auto-created on first run with schema migrations.
  - The daemon PID file lives at `~/.local/share/kanteto/daemon.pid`.
  - User configuration lives at `~/.config/kanteto/config.toml` (entirely optional).
- **Schema Details:**
  - `tasks` table: `id` (TEXT PK, ULID), `title` (TEXT), `due_at` (DATETIME nullable), `completed` (INTEGER), `completed_at` (DATETIME nullable), `created_at` (DATETIME), `remind_at` (DATETIME nullable), `reminded` (INTEGER), `recurrence_pattern` (TEXT nullable), `recurrence_time` (TEXT nullable), `recurrence_next_due` (DATETIME nullable).
  - `schema_version` table: tracks migration version for future schema updates.
  - Indexes on `due_at`, `remind_at`, and `completed` for efficient date-range queries.
