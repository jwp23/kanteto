# Software Bill of Materials (SBOM)

Purpose: This file lists all approved technologies, libraries, and dependencies for Kanteto, including their specific versions.

> The AI developer must adhere to this list and the specified versions. Do not add new dependencies without review per `security.md` Section 3.

---

## 0. Technology Stack Overview

| Category | Component Name | Version | Rationale / Usage |
| :--- | :--- | :--- | :--- |
| **Language** | `Go` | `1.26` | Primary language. Single binary, cross-compilation, strong concurrency. |
| **CLI Framework** | `github.com/spf13/cobra` | `^1.8` | Command routing (`kt add`, `kt done`, `kt daemon`, etc.). |
| **TUI Framework** | `github.com/charmbracelet/bubbletea` | `^1.2` | Interactive terminal UI with Elm architecture. |
| **TUI Styling** | `github.com/charmbracelet/lipgloss` | `^1.0` | Terminal styling and urgency gradient colors. |
| **TUI Components** | `github.com/charmbracelet/bubbles` | `^0.20` | Text input, spinners, and other TUI widgets. |
| **Database** | `dolt` (external CLI) | `v1.81.10+` | Dolt database via CLI (`dolt sql`). User-installed, not a Go module. |
| **Migration** | `modernc.org/sqlite` | `^1.46` | Pure-Go SQLite — used only by `kt migrate` for one-time import from legacy SQLite DB. |
| **ID Generation** | `github.com/oklog/ulid/v2` | `^2.2` | Sortable, unique task IDs. |
| **Config Parsing** | `github.com/BurntSushi/toml` | `^1.4` | TOML config file loading. |
| **Date Parsing** | `github.com/tj/go-naturaldate` | `^1.3` | Natural language date input ("tomorrow", "march 11"). |
| **Testing** | `testing` (stdlib) | built-in | Standard Go testing. No external test frameworks. |

---

## 1. Version Management & Updates

- **Update Strategy:** Dependencies are updated manually. Run `go get -u <module>` for minor/patch updates. Major version bumps require running the full test suite and reviewing changelogs for breaking changes.
- **Security Scanning:** Run `govulncheck ./...` periodically to check for known vulnerabilities. Run `go mod verify` to ensure module integrity against `go.sum`.
- **Pinning:** `go.sum` provides cryptographic verification of all module versions. Do not delete or regenerate `go.sum` without cause.

---

## 2. Documentation & Resources

- **Core Framework Documentation:**
  - **Cobra CLI:** https://cobra.dev/
  - **Bubble Tea:** https://github.com/charmbracelet/bubbletea
  - **Lip Gloss:** https://github.com/charmbracelet/lipgloss
  - **Bubbles:** https://github.com/charmbracelet/bubbles

- **Database:**
  - **Dolt:** https://docs.dolthub.com/
  - **Dolt SQL reference:** https://docs.dolthub.com/sql-reference
  - **modernc.org/sqlite (migration only):** https://pkg.go.dev/modernc.org/sqlite

- **Development Tools:**
  - **Go Documentation:** https://go.dev/doc/
  - **Go Module Reference:** https://go.dev/ref/mod
  - **govulncheck:** https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
