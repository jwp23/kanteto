# Kanteto Design Document

**Date:** 2026-03-03
**Status:** Approved

## Summary

Kanteto is a CLI + TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done. It persists data across sessions, reminds you with audible alerts when things are due, supports recurring schedules, and provides day/week/month views with time navigation.

**Target users:** Non-technical users who want a fast, keyboard-driven way to manage micro-commitments from the terminal.

## Tech Stack

- **Language:** Go 1.26
- **TUI:** Bubble Tea (charmbracelet/bubbletea) + Lip Gloss (styling)
- **CLI:** Cobra (spf13/cobra)
- **Storage:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **NLP dates:** olebedev/when or tj/go-naturaldate + custom recurring parser
- **Sound:** afplay (macOS) / paplay (Linux) for reminder audio

## Data Model

```
Task {
  id:           string (ULID)
  title:        string
  due_at:       timestamp | null        # one-off deadline
  completed:    bool
  completed_at: timestamp | null
  created_at:   timestamp
  recurrence:   RecurrenceRule | null
  remind_at:    timestamp | null
  reminded:     bool
}

RecurrenceRule {
  pattern:      string                  # "weekdays", "weekly:fri", "daily", "monthly:15"
  time:         string                  # "16:00", "17:00"
  next_due:     timestamp               # precomputed next occurrence
}
```

**Key decisions:**
- ULID for sortable, unique IDs
- Recurring tasks advance to next occurrence on completion (not deleted)
- Remind and due are separate — reminder can fire before the deadline
- Completed recurring tasks reset `reminded` and compute `next_due`

## CLI Commands

```
kt add "Send weekly update" --every "weekdays at 4pm"
kt add "Call dentist" --by "march 11"
kt add "Review PRs" --every "friday at 5pm"
kt add "Buy groceries"

kt done <id>
kt snooze <id> --for "1 hour"
kt rm <id>
kt edit <id> --title "new title"

kt list                          # today's tasks
kt day / kt week / kt month     # views
kt day -n / kt day -p           # next/prev navigation
kt day --date "march 15"        # specific date

kt                               # launches TUI
kt daemon start/stop/status      # manage background reminders
```

## TUI Layout

```
┌─────────────────────────────────────────────────────┐
│  kanteto              Wed, Mar 4 2026         day > │
├─────────────────────────────────────────────────────┤
│                                                     │
│  > OVERDUE                                          │
│    o Call dentist                     Mar 1 (3d ago) │
│                                                     │
│  > TODAY                                            │
│    o Send weekly update                       4:00p │
│    o Review PRs                               5:00p │
│    o Buy groceries                                  │
│                                                     │
│  > UPCOMING                                         │
│    o Team retro                          Thu 10:00a │
│    o Submit expense report                   Mar 11 │
│                                                     │
├─────────────────────────────────────────────────────┤
│  a:add  space:done  x:delete  s:snooze  ?:help      │
└─────────────────────────────────────────────────────┘
```

**Keybindings:**

| Key | Action |
|-----|--------|
| j/k, up/down | Move between tasks |
| h/l, left/right | Navigate back/forward in time |
| d w m | Switch day/week/month view |
| a | Add task |
| space | Mark done/undone |
| x | Delete |
| s | Snooze |
| Enter | Expand/edit details |
| q | Quit |
| ? | Help overlay |

**Urgency gradient:** Tasks shift color as they approach their deadline:
- More than 2 hours away: default (white/normal)
- 2 hours: yellow
- 1 hour: amber/orange
- 30 min: deep orange
- 15 min: orange-red
- Overdue: red

Continuous RGB interpolation via Lip Gloss, not discrete steps.

## Daemon & Reminders

The daemon is a lightweight background process:

1. Wakes every 30 seconds
2. Queries SQLite for tasks where `remind_at <= now AND reminded = false`
3. Plays sound via `afplay` (macOS) / `paplay` (Linux)
4. Marks `reminded = true`; for recurring tasks, computes `next_due` and resets
5. Sleeps

**PID management:** Writes to `~/.local/share/kanteto/daemon.pid`. Checks PID on start to prevent duplicates.

**Catch-up on TUI open:** Overdue tasks display in red at the top. No extra sound — the visual is sufficient.

**Sound:** Bundled default notification sound (embedded in binary). User can override in config.

## Configuration

All paths follow XDG conventions:

```
~/.config/kanteto/config.toml       # user config
~/.local/share/kanteto/kanteto.db   # SQLite database
~/.local/share/kanteto/daemon.pid   # daemon PID
```

Respects `XDG_CONFIG_HOME` and `XDG_DATA_HOME` if set.

```toml
# ~/.config/kanteto/config.toml (entirely optional)

[reminder]
sound = "/path/to/custom-sound.wav"
lead_time = "15m"

[display]
time_format = "12h"      # "12h" or "24h"
week_start = "sunday"    # "sunday" or "monday"

[urgency]
gradient_start = "2h"    # when color shifting begins
```

All settings have sensible defaults. Zero config required to get started.

## Architecture

```
┌──────────────────────────────────────────────┐
│                  kanteto binary               │
├──────────┬──────────┬──────────┬─────────────┤
│  CLI     │  TUI     │  Daemon  │  NLP Parser │
│ (Cobra)  │(BubbleTea)│ (goroutine)│ (date parse)│
├──────────┴──────────┴──────────┴─────────────┤
│              Task Service Layer               │
│  (add, complete, snooze, list, query by date) │
├──────────────────────────────────────────────┤
│              SQLite Repository                │
│  (CRUD, date range queries, recurrence mgmt) │
├──────────────────────────────────────────────┤
│           ~/.local/share/kanteto/             │
│                kanteto.db                     │
└──────────────────────────────────────────────┘
```

**Package structure:**

```
kanteto/
├── cmd/
│   ├── root.go           # kt (launches TUI)
│   ├── add.go
│   ├── done.go
│   ├── list.go           # list/day/week/month
│   └── daemon.go         # daemon start/stop/status
├── internal/
│   ├── task/
│   │   ├── model.go
│   │   ├── service.go
│   │   └── service_test.go
│   ├── store/
│   │   ├── sqlite.go
│   │   └── sqlite_test.go
│   ├── tui/
│   │   ├── app.go
│   │   ├── day.go
│   │   ├── week.go
│   │   ├── month.go
│   │   └── styles.go
│   ├── daemon/
│   │   └── daemon.go
│   ├── nlp/
│   │   ├── parse.go
│   │   └── parse_test.go
│   └── config/
│       └── config.go
├── go.mod
├── go.sum
└── main.go
```

**Key principles:**
- Single binary — CLI, TUI, and daemon in one
- Service layer — all interfaces share the same `task.Service`
- SQLite WAL mode — concurrent reads/writes between daemon and CLI/TUI
- Pure Go SQLite (modernc.org/sqlite) — no CGO, clean cross-compilation
