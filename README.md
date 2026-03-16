# Kanteto

*kanteto* (κάντε το) — "Do It" in Greek.

A TUI tool for tracking small tasks and promises that are too small
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

Kanteto stores its data in a Dolt repository at `~/.local/share/kanteto/dolt/`.
To sync tasks across machines, use the `dolt` CLI directly in the data directory.

Dolt supports several remote backends — you are not limited to DoltHub:

| Remote type | URL format |
|-------------|------------|
| Git (GitHub, GitLab, etc.) | `git@github.com:user/repo.git` or `https://github.com/user/repo.git` |
| DoltHub | `https://doltremoteapi.dolthub.com/user/repo` |
| Filesystem | `/path/to/remote/dir` |
| AWS (S3 + DynamoDB) | `aws://[table:bucket]/db` |
| GCS | `gs://bucket/path` |

### Initial remote setup (GitHub example)

1. Create a **new repository** on GitHub (e.g. `kanteto-tasks`).

2. GitHub repos need at least one commit before Dolt can push. Initialize it:

```sh
cd /tmp
git clone git@github.com:<user>/<repo>.git
cd <repo>
git commit --allow-empty -m "init"
git push origin main
cd - && rm -rf /tmp/<repo>
```

3. Add the remote and push from the Dolt data directory:

```sh
cd ~/.local/share/kanteto/dolt
dolt remote add origin git@github.com:<user>/<repo>.git
dolt push origin main
```

> **Note:** Dolt stores data in custom git refs that won't appear as normal
> files in GitHub's UI. The repo will look mostly empty in the browser —
> this is expected. Use `dolt clone` or `dolt pull` to retrieve the data on
> another machine.

3. On your **second machine**, after installing Kanteto and Dolt:

```sh
mkdir -p ~/.local/share/kanteto
dolt clone git@github.com:<user>/<repo>.git ~/.local/share/kanteto/dolt
```

### Ongoing sync

```sh
cd ~/.local/share/kanteto/dolt
dolt push origin main        # push local changes to remote
dolt pull origin              # pull remote changes
```

### Useful commands

```sh
cd ~/.local/share/kanteto/dolt
dolt remote -v               # list configured remotes
dolt status                  # check for uncommitted changes
dolt log                     # view commit history
```

## Configuration

Kanteto uses an optional TOML config file at `~/.config/kanteto/config.toml`:

```toml
active_profile = "default"   # current profile
```

Data is stored in a Dolt database at `~/.local/share/kanteto/`.
Both paths follow the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) spec.

## License

MIT
