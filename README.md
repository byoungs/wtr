# wtr — Worktree Review TUI

A terminal UI for reviewing git worktree diffs, running tests, and landing branches. Built for developers who use `git worktree` for parallel feature work and want a fast way to review, validate, and merge without leaving the terminal.

![wtr demo](docs/demo.gif)

## Install

```bash
go install github.com/byoungs/wtr/cmd/wtr@latest
```

Or build from source:

```bash
make build    # → bin/wtr
make install  # → ~/go/bin/wtr
```

## Usage

Run from any git repo root (on the main branch):

```bash
wtr             # review worktrees in current repo
wtr /path/to    # review worktrees in another repo
```

## Screens

### Worktree List

The landing screen shows all worktrees with their status at a glance — commit counts, test results, and dirty state.

| Key | Action |
|-----|--------|
| `enter` / `→` | Review changed files |
| `t` | Run `make validate` in background |
| `v` | View test/validate output |
| `r` | Rebase on main |
| `s` | Squash to single commit |
| `l` | Land (ff-only merge → test → validate → push) |
| `d` | Delete worktree |
| `o` | Open in VS Code |
| `u` | Refresh state |

### File List

Shows files changed in a worktree vs main. Mark files as reviewed with `x`.

| Key | Action |
|-----|--------|
| `enter` / `→` | View file diff |
| `x` | Toggle reviewed checkmark |
| `a` | View all diffs (integrated) |
| `g` | Git status (uncommitted changes) |
| `o` | Open file in VS Code |
| `esc` / `←` | Back |

### Diff View

Unified or side-by-side diff viewer with hunk navigation.

| Key | Action |
|-----|--------|
| `j` / `k` / arrows | Scroll line by line |
| `space` / `b` | Page down / up |
| `]` / `[` | Next / prev file |
| `n` / `p` | Next / prev hunk |
| `v` | Toggle side-by-side / unified |
| `x` | Toggle reviewed |
| `esc` / `←` | Back |

### Git Status

View uncommitted and untracked files in a worktree. Revert individual files with `r`.

### Test Output

Live-streaming output from `make validate`, persisted across sessions.

## How It Works

wtr stores state alongside your repo in `.git/wtr/`:

- **`.git/wtr-state.json`** — test status, reviewed files, tested commit hashes
- **`.git/wtr/<branch>.log`** — test output (survives restart)
- **`.git/wtr/<branch>.status`** — running / passed / failed
- **`.git/wtr/<branch>.pid`** — background process PID

Tests run in the background and survive quitting wtr. Come back later and the results are waiting.

## Landing a Branch

Press `l` to land a branch. This runs four steps in sequence:

1. `git merge --ff-only` into main
2. `make test`
3. `make validate`
4. `git push`

If any step fails, the process stops and shows you what went wrong.

## Tech Stack

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea). No database, no config files, no setup.
