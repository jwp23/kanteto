# Infrastructure Blueprint

Purpose: This file describes the project's technical foundation, including the programming language, coding standards, and how to build and run the application.

---

## What We're Building

- **Programming Language:** Go 1.26 (latest stable)
- **Main Framework/Tool:** Cobra (CLI), Bubble Tea (TUI), Lip Gloss (styling)
- **A Quick Summary:** A single-binary CLI + TUI tool for tracking micro-tasks with recurring schedules. See `prd.md` for full feature details.

---

## How to Run it on Your Computer

- **Installation Command:** `go mod download`
- **Build Command:** `go build -o kt ./cmd/kt`
- **Run:** `./kt` (TUI) or `./kt --help` (CLI)
- **Run Tests:** `go test ./... -v`

---

## Project Architecture & Conventions

- **Framework:** Go with Cobra (CLI routing), Bubble Tea (TUI framework), Lip Gloss (terminal styling)
- **Directory Structure:**
  - **`cmd/`:** Cobra command definitions. One file per command (`add.go`, `done.go`, `list.go`, `daemon.go`). `root.go` is the entry point that launches the TUI when called with no subcommand.
  - **`internal/task/`:** Domain model (`model.go`), business logic (`service.go`), recurrence computation (`recurrence.go`), and ULID generation (`id.go`).
  - **`internal/store/`:** Dolt-backed repository. Shells out to `dolt sql` for queries; auto-initializes the Dolt repo on first use.
  - **`internal/tui/`:** Bubble Tea models and views. `app.go` is the main model; `day.go`, `week.go`, `month.go` are view renderers; `styles.go` defines the urgency gradient and Lip Gloss styles.
  - **`internal/nlp/`:** Natural language date parsing. Handles "march 11", "tomorrow", "weekdays at 4pm", etc.
  - **`internal/config/`:** XDG-compliant configuration loading from `~/.config/kanteto/config.toml`.
  - **`internal/sync/`:** Dolt sync operations (push/pull/remote management). Thin wrapper around `dolt` CLI commands.
- **Key Architectural Decision:** CLI, TUI, and daemon all share a single `task.Service` layer backed by a Dolt database (via `dolt sql` CLI). No duplicated logic between interfaces.

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
- **External Dependencies:** `dolt` (v1.81.10+) and `git` must be on PATH. User installs separately.
- **External Services:** Optional Dolt remote (GitHub, DoltHub) for sync. Not required for local use.
- **Distribution:** Single binary built with `go build`. No CGO required. Requires `dolt` CLI as an external runtime dependency. Cross-compilation via `GOOS`/`GOARCH` environment variables.

---

## Where Your Data is Stored

- **Data Storage Method:** Dolt database at `~/.local/share/kanteto/` (XDG-compliant, respects `XDG_DATA_HOME`). The store shells out to `dolt sql -q` for queries and `dolt sql -q ... -r json` for reads.
- **Important Notes:**
  - Dolt repo is auto-initialized on first run with table creation and initial commit.
  - Sync to remotes via `dolt push`/`dolt pull` (wrapped by `kt sync` commands).
  - The daemon PID file lives at `~/.local/share/kanteto/daemon.pid`.
  - User configuration lives at `~/.config/kanteto/config.toml` (entirely optional).
- **Schema Details (MySQL dialect):**
  - `tasks` table: `id` (VARCHAR(255) PK, ULID), `title` (VARCHAR(1024)), `due_at` (DATETIME nullable), `completed` (TINYINT(1)), `completed_at` (DATETIME nullable), `created_at` (DATETIME), `recurrence_pattern` (VARCHAR(255) nullable), `recurrence_time` (VARCHAR(255) nullable), `tags` (JSON), `profile` (VARCHAR(255) default 'default').
