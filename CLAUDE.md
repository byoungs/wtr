# wtr — Worktree Review TUI

Terminal UI for reviewing git worktree diffs, running tests, and landing branches.

## Build & Run

    make build     # Build binary to bin/wtr
    make test      # Run tests
    make install   # Install to ~/go/bin/wtr

## Usage

Run from any git repo root (on main branch):

    wtr             # Review worktrees in current repo
    wtr /path/to   # Review worktrees in another repo

## Screens

- **Worktree List** — shows all worktrees with commit/test/dirty status
- **File List** — committed file diffs for a worktree (vs main)
- **Diff View** — unified or side-by-side diff for a single file
- **All Diffs** — integrated unified view of all files
- **Git Status** — uncommitted/untracked files with revert capability
- **Test Output** — live/persisted output from make validate
- **Help** — keybinding reference (press h or ?)

## Modes

wtr auto-detects which mode to use:

- **Worktree mode** — when git worktrees exist (besides main). Landing screen shows worktree list.
- **Direct mode** — when only main branch exists (no worktrees). Landing screen shows branch review with unpushed commits, test status, and push action.

### Direct Mode Keys
- `→`/`enter`: review changed files (diff vs origin)
- `g`: git status (uncommitted changes)
- `t`: run make validate (background)
- `o`: view test/validate output
- `l`: push to origin (test + validate + push)
- `u`: refresh
- `h`/`?`: help
- `q`: quit

## Key Bindings

### Worktree List
- `→`/`enter`: review files
- `e`: edit worktree in VS Code
- `t`: run make validate (background, survives exit)
- `o`: view test/validate output
- `r`: rebase on main
- `s`: squash to 1 commit on main
- `l`: land (ff-only merge + test + validate + push)
- `d`: delete worktree (gentle, then type "force" if needed)
- `u`: refresh worktree state
- `h`/`?`: help
- `q`: quit

### File List
- `→`/`enter`: view diff
- `e`: edit file in VS Code
- `a`: all diffs (integrated view)
- `g`: git status (uncommitted changes)
- `x`: toggle reviewed checkmark
- `←`/`esc`: back

### Diff View
- `j/k`/arrows: scroll line by line
- `space`/`b`: page down/up
- `]`/`[`: next/prev file
- `n`/`p`: next/prev hunk
- `v`: toggle side-by-side / unified
- `x`: toggle reviewed (auto-marks when scrolled to bottom)
- `←`/`esc`: back

### Git Status
- `→`/`enter`: view diff
- `r`: revert/delete file (with confirmation)
- `e`: edit in VS Code
- `←`/`esc`: back

## Key Binding Rules

When adding or changing a keybinding:
1. Update `internal/wtr/keys.go` (struct + var)
2. Update the handler in the relevant screen file
3. Update the help bar (`styleHelp.Render(...)`) in that screen's view function
4. Update `internal/wtr/help.go` help screen
5. Update this CLAUDE.md

## Tech Stack

Go + Bubble Tea (charmbracelet/bubbletea). No database.

State sources:
- Git worktrees (live from git CLI)
- `.git/wtr-state.json` — persisted test status + tested commit hashes
- `.git/wtr/<branch>.log` — test/validate output (survives restart)
- `.git/wtr/<branch>.status` — running/passed/failed
- `.git/wtr/<branch>.pid` — background process PID

## Project Structure

    cmd/wtr/main.go               Entry point
    internal/git/branch.go         Branch info (ahead/behind origin)
    internal/git/worktree.go       Worktree discovery + commit counts
    internal/git/diff.go           Diff parsing (committed + working tree)
    internal/wtr/app.go            Root Bubble Tea model, screen routing
    internal/wtr/worktree_list.go  Worktree list screen
    internal/wtr/file_list.go      File list screen
    internal/wtr/diff_view.go      Diff view (side-by-side + unified)
    internal/wtr/direct_landing.go Direct mode landing screen
    internal/wtr/all_diffs.go      Integrated all-files diff view
    internal/wtr/git_status.go     Uncommitted changes screen
    internal/wtr/test_output.go    Live/persisted test output viewer
    internal/wtr/help.go           Help screen
    internal/wtr/styles.go         Lip Gloss styles
    internal/wtr/keys.go           Key bindings (single source of truth)
    internal/land/land.go          Land workflow (merge + test + validate + push)
    internal/runner/runner.go      Background process management
    internal/state/state.go        Persistent state (JSON)
