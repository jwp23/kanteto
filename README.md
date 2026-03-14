# Kanteto

*kanteto* (κάντε το) — "Do It" in Greek.

A CLI and TUI tool for tracking small tasks and promises that are too small
for tickets but still need to get done on time.

> **Note:** Kanteto is under active development. Features may change between
> releases. Feel free to take it for a test drive and
> [open an issue](https://github.com/jwp23/kanteto/issues) if you have ideas
> or run into problems.

## Prerequisites

- **Go** 1.25+ (for building)
- **Dolt** v1.81.10+ — install from https://docs.dolthub.com/introduction/installation
- **git** (required by Dolt for remote sync)

## Install

```sh
go install github.com/jwp23/kanteto/cmd/kt@latest
```

Or build from source:

```sh
git clone https://github.com/jwp23/kanteto.git
cd kanteto
go build -o kt ./cmd/kt
```

## Quick Start

```sh
kt add "Call dentist" --by "march 11"
kt add "Buy groceries"
kt list
kt done <id>
kt                    # launch the TUI
```

## CLI Commands

### `kt add`

Add a task with an optional deadline or recurrence.

```sh
kt add "Call dentist" --by "tomorrow at 3pm"
kt add "Send weekly update" --every "weekdays at 4pm"
```

| Flag | Description |
|------|-------------|
| `--by <date>` | Natural language deadline (`tomorrow`, `march 11`, `next friday`, `in 5 minutes`, `at 3pm`) |
| `--every <pattern>` | Recurrence (`daily at 9am`, `weekdays at 4pm`, `friday at 5pm`) |

### `kt list`

Show tasks grouped into OVERDUE, TODAY, UPCOMING, and ANYTIME sections.

```sh
kt list
kt list --next   # shift forward one day
kt list --prev   # shift back one day
```

### `kt done <id>`

Mark a task as complete. Recurring tasks advance to the next occurrence.

### `kt snooze <id>`

Postpone a task's deadline (default: 1 hour).

```sh
kt snooze <id> --for "2 hours"
```

### `kt rm <id>`

Permanently delete a task.

### `kt tag <id> <tag>` / `kt untag <id> <tag>`

Add or remove tags on a task.

```sh
kt tag abc123 work
kt untag abc123 work
kt list --tag work       # filter by tag
```

### `kt profile`

Manage task profiles to scope views (e.g., work vs personal).

```sh
kt profile use work      # switch active profile
kt profile list          # show all profiles
kt profile show          # show active profile
kt add "Task" --profile personal   # override for one command
```

### `kt sync`

Sync tasks to a Dolt remote (GitHub, DoltHub, etc.).

```sh
kt sync remote add origin https://doltremoteapi.dolthub.com/user/tasks
kt sync push             # commit and push to remote
kt sync pull             # pull from remote
kt sync remote list      # show configured remotes
```

### `kt migrate`

One-time migration from an existing SQLite database to Dolt.

```sh
kt migrate               # reads kanteto.db, writes to dolt/
```

### `kt daemon`

Start the background reminder daemon. Checks for due tasks every 30 seconds
and plays an audible alert.

```sh
kt daemon &
```

Task IDs support prefix matching — type just enough characters to be unambiguous.

## TUI

Run `kt` with no arguments to launch the interactive terminal UI.

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move cursor down / up |
| `h` / `l` | Previous / next time period |
| `t` | Jump to today |
| `Enter` | Drill down from month view into day view |

### Views

| Key | View |
|-----|------|
| `d` | Day |
| `w` | Week |
| `m` | Month |

### Actions

| Key | Action |
|-----|--------|
| `space` | Complete task |
| `a` | Add task inline (supports natural language: `Call dentist by march 11`) |
| `x` | Delete task |
| `?` | Help overlay |
| `q` | Quit |

## Configuration

Kanteto uses an optional TOML config file at `~/.config/kanteto/config.toml`:

```toml
reminder_lead_time = "15m"    # how far before due time to alert
sound_file = "/path/to/alert.wav"
active_profile = "default"   # current profile
```

Data is stored in a Dolt database at `~/.local/share/kanteto/`.
Both paths follow the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) spec.

## License

MIT
