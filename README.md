# Kanteto

*kanteto* (κάντε το) — "Do It" in Greek.

A TUI tool for tracking small tasks and promises that are too small
for tickets but still need to get done on time.

> **Note:** Kanteto is under active development. Features may change between
> releases. Feel free to take it for a test drive and
> [open an issue](https://github.com/jwp23/kanteto/issues) if you have ideas
> or run into problems.

## Prerequisites

- **Go** 1.25+

## Install

```sh
go install -tags gms_pure_go github.com/jwp23/kanteto/cmd/kt@latest
```

Or build from source:

```sh
git clone https://github.com/jwp23/kanteto.git
cd kanteto
go build -tags gms_pure_go -o kt ./cmd/kt
```

The `gms_pure_go` tag uses Go's stdlib regex instead of ICU, which removes
the cgo/ICU dependency and allows a clean `go install` on any platform.

## Quick Start

```sh
kt                    # launch the TUI
kt migrate            # one-time SQLite→Dolt migration
```

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
| `s` | Snooze task |
| `e` | Edit task deadline |
| `x` | Delete task |
| `?` | Help overlay |
| `q` | Quit |

Task IDs support prefix matching — type just enough characters to be unambiguous.

## `kt migrate`

One-time migration from an existing SQLite database to Dolt.

```sh
kt migrate               # reads kanteto.db, writes to dolt/
kt migrate --force       # re-run even if tasks already exist in Dolt
```

## Syncing Across Machines

Kanteto uses an embedded Dolt engine — no separate Dolt server or CLI is
required at runtime. Task data lives at `~/.local/share/kanteto/`.

Every mutation (add, complete, delete, edit) is automatically committed
in-process. If a remote is configured, changes are pushed in the background.

Use `P` (push) and `p` (pull) in the TUI for manual sync.

### Remote setup

Remote configuration requires the `dolt` CLI as a one-time setup step.
Install it from https://docs.dolthub.com/introduction/installation, then:

```sh
cd ~/.local/share/kanteto/kanteto   # Dolt data directory
dolt remote add origin <url>
dolt push origin main
```

Dolt supports several remote backends:

| Remote type | URL format |
|-------------|------------|
| Git (GitHub, GitLab, etc.) | `git@github.com:user/repo.git` or `https://github.com/user/repo.git` |
| DoltHub | `https://doltremoteapi.dolthub.com/user/repo` |
| Filesystem | `/path/to/remote/dir` |
| AWS (S3 + DynamoDB) | `aws://[table:bucket]/db` |
| GCS | `gs://bucket/path` |

> **Note:** When using a Git remote (GitHub, GitLab), the repo needs at
> least one commit before Dolt can push. Create the repo, push an empty
> commit (`git commit --allow-empty -m "init" && git push`), then add it
> as a Dolt remote.

### Second machine

On another machine, after building `kt`, clone the remote into the data
directory:

```sh
mkdir -p ~/.local/share/kanteto
dolt clone <url> ~/.local/share/kanteto/kanteto
```

After this one-time setup, `kt` handles all sync automatically.

## Configuration

Kanteto uses an optional TOML config file at `~/.config/kanteto/config.toml`:

```toml
active_profile = "default"   # current profile
```

Data is stored in an embedded Dolt database at `~/.local/share/kanteto/kanteto/`.
Both paths follow the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) spec.

## License

MIT
