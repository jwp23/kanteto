# Security Blueprint

Purpose: This file establishes the security rules and best practices for Kanteto.

## 0. Baseline Best Practices

- **Never Hardcode Secrets:** No API keys, passwords, or secrets in source code. Kanteto currently has no secrets, but if external integrations are added in the future, this rule applies.
- **Use a `.gitignore` file:** The `.gitignore` must include entries for any files containing secrets (`.env`, `.env.local`, `*.pem`).
- **Apply the Principle of Least Privilege:** The daemon process should run with the minimum permissions needed (read/write to the SQLite database and play audio).

---

## 1. Data Sensitivity Level

- **My Project's Data is: Internal.** Kanteto stores task titles, due dates, and schedule information on the local filesystem. It does not handle personal identifiable information (PII), authentication credentials, or sensitive data. All data is local to the user's machine.

---

## 2. Authentication & Authorization

- **Authentication Method: None.** Kanteto is a single-user local CLI tool. There are no user accounts or login.
- **Authorization Rules: N/A.** The user who runs the binary has full access to all tasks. File permissions on the SQLite database (`~/.local/share/kanteto/kanteto.db`) are the only access control, managed by the OS.

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
  - `github.com/charmbracelet/bubbles` — TUI components (text input, etc.)
  - `modernc.org/sqlite` — Pure-Go SQLite (no CGO, reduces attack surface from C code)
  - `github.com/oklog/ulid/v2` — ULID generation
  - `github.com/BurntSushi/toml` — TOML config parsing
  - `github.com/tj/go-naturaldate` — Natural language date parsing

---

## 4. Secrets Management

- **Where Secrets are Stored: N/A.** Kanteto has no secrets. It does not use API keys, database passwords, or access tokens. The optional config file (`~/.config/kanteto/config.toml`) contains only display preferences and sound file paths — no sensitive data.
- **Who Has Access: Only the local user.** Data and config files are created with standard user permissions.

---

## 5. Application-Specific Security Considerations

- **SQLite Injection:** All database queries must use parameterized queries (`?` placeholders). Never interpolate user input into SQL strings.
- **Command Injection:** The daemon plays sounds via `exec.Command("afplay", path)`. The sound file path must be validated — no shell interpretation, no user-controlled arguments passed unsanitized.
- **File Path Traversal:** When resolving XDG paths or custom sound file paths from config, validate that paths are absolute and within expected directories. Do not follow symlinks outside the data directory.
- **Daemon PID File:** The PID file at `~/.local/share/kanteto/daemon.pid` should be created with mode `0600` to prevent other users from spoofing the daemon.
