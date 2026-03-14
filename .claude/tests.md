# Testing Strategy

Purpose: This file outlines the strategy for testing Kanteto to ensure it works correctly and prevent regressions.

---

## 1. Our Testing Philosophy

- **Overall Goal:** Every public behavior must have a test. We measure coverage by "does this behavior have a test?" not by line-coverage percentages.
- **TDD:** Write a failing test first, then implement the code to make it pass.
- **No mocked-behavior tests:** Tests must exercise real logic. Never write a test that only validates mocked return values.
- **Pristine output:** Test output must be clean. If a test intentionally triggers an error, capture and assert on it — no unchecked warnings or log noise.
- **Broken Windows:** All test failures are our responsibility. Never delete a failing test — diagnose and fix it, or raise the issue.

---

## 2. Types of Tests We Write

- **Unit Tests:** Yes. Domain model behavior, NLP date parsing, recurrence logic, config loading, urgency color assignment. Table-driven tests are the default pattern.
- **Integration Tests:** Yes. Full lifecycle workflows (add -> list -> complete -> get), recurring task advancement, TUI model updates with simulated keypresses. Build-tagged (`integration`) tests for concurrent Dolt access.
- **End-to-End (E2E) Tests:** No. Kanteto is a local CLI tool with no network dependencies or browser UI.

### Package Test Map

| Package | Test Focus |
|---|---|
| `internal/task/` | Model behavior (overdue, due-today), service CRUD, recurrence parsing/advancement, ULID generation, full lifecycle integration |
| `internal/store/` | CRUD operations, date-range queries, overdue/undated queries, field updates, SQL special character round-trips |
| `internal/nlp/` | Date parsing (month names, relative dates, weekdays, ISO, durations), deadline extraction from natural language |
| `internal/tui/` | Day/week/month view rendering, keybinding navigation, input modes (add/snooze), cursor movement and clamping, urgency color gradients |
| `internal/config/` | Default values, TOML file loading, XDG path resolution |
| `cmd/` | `migrate` command with happy-path and error-case coverage |

---

## 3. Testing Frameworks & Tools

- **Framework:** Go stdlib `testing` package
- **Run all tests:** `go test ./... -v`
- **Run integration tests:** `go test ./... -v -tags=integration`
- **Lint before commit:** `go vet ./...` and `staticcheck ./...`

### Key Patterns

- **Table-driven tests** — default for any function with multiple input/output cases (NLP parsing, model predicates, urgency colors, recurrence specs)
- **Dolt with `t.TempDir()` + `skipIfNoDolt`** — store and service tests create isolated Dolt repos in temp directories, skipping gracefully when dolt is not installed
- **`t.Helper()`** — all test setup functions are marked as helpers for clean failure output
- **`t.Setenv()`** — environment variable isolation for config and XDG path tests
- **TUI key simulation** — `sendKey()` and `sendSpecialKey()` simulate keypresses against Bubble Tea models
- **Fixed "now" times** — date-sensitive tests use deterministic reference times

---

## 4. Coverage Expectations

Every user story in `prd.md` must have corresponding test coverage. Rather than a numeric percentage target, we enforce coverage by layer:

- **Store:** Every query method must have a test (CRUD, date-range, overdue, undated)
- **Service:** Every public method must have a test (add, complete, delete, snooze, list variants)
- **CLI:** The `migrate` command must have happy-path and error-case tests
- **TUI:** Every view must have rendering tests; every keybinding must have a navigation test
- **NLP:** Every supported date format must appear in a table-driven test case
- **Config:** Default loading and file-based overrides must be tested
