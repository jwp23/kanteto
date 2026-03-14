# Security Blueprint

Purpose: This file establishes the security rules and best practices for Kanteto.

## 0. Baseline Best Practices

- **Never Hardcode Secrets:** No API keys, passwords, or secrets in source code. Kanteto currently has no secrets, but if external integrations are added in the future, this rule applies.
- **Use a `.gitignore` file:** The `.gitignore` must include entries for any files containing secrets (`.env`, `.env.local`, `*.pem`).
- **Apply the Principle of Least Privilege:** The application should run with the minimum permissions needed (read/write to the Dolt data directory).

---

## 1. Data Sensitivity Level

- **My Project's Data is: Internal.** Kanteto stores task titles, due dates, and schedule information on the local filesystem. It does not handle personal identifiable information (PII), authentication credentials, or sensitive data. All data is local to the user's machine.

---

## 2. Authentication & Authorization

- **Authentication Method: None.** Kanteto is a single-user local CLI tool. There are no user accounts or login.
- **Authorization Rules: N/A.** The user who runs the binary has full access to all tasks. File permissions on the Dolt data directory (`~/.local/share/kanteto/dolt/`) are the only access control, managed by the OS.

---

## 3. Dependency & Supply Chain Security

- **How We Check Dependencies:** Manual review of Go modules. Run `go mod verify` to check module checksums against `go.sum`. Use `govulncheck` to scan for known vulnerabilities in dependencies.
- **Rule for Adding New Dependencies:** New dependencies must be reviewed for:
  - Active maintenance (recent commits, responsive maintainers)
  - Security track record
  - License compatibility (MIT, BSD, Apache 2.0 preferred)
  - Minimal transitive dependency footprint
- **Key Dependencies (approved):**
  - `github.com/spf13/cobra` — CLI framework (widely used, well-maintained)
  - `github.com/charmbracelet/bubbletea` — TUI framework (active development, Charm ecosystem)
  - `github.com/charmbracelet/lipgloss` — Terminal styling
  - `modernc.org/sqlite` — Pure-Go SQLite (migration-only; no longer used at runtime)
  - `dolt` CLI — Primary data backend (external runtime dependency)
  - `github.com/oklog/ulid/v2` — ULID generation
  - `github.com/BurntSushi/toml` — TOML config parsing

---

## 4. Secrets Management

- **Where Secrets are Stored: N/A.** Kanteto has no secrets. It does not use API keys, database passwords, or access tokens. The optional config file (`~/.config/kanteto/config.toml`) contains only display preferences — no sensitive data.
- **Who Has Access: Only the local user.** Data and config files are created with standard user permissions.

---

## 5. Application-Specific Security Considerations

- **SQL Injection:** All Dolt SQL queries use the `quote()` helper which escapes single quotes via doubling (`'` → `''`). User input is never interpolated raw into SQL strings.
- **Command Injection:** Dolt operations use `exec.Command` with explicit argument lists — no shell interpretation. User input is passed as SQL string literals, not as command arguments.
- **File Path Traversal:** When resolving XDG paths, validate that paths are absolute and within expected directories. Do not follow symlinks outside the data directory.
