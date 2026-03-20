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

## Key Bindings

### Worktree List
- `j/k` or arrows: navigate
- `enter`: review files
- `t`: run tests in background
- `l`: land (ff-only merge + make test + make validate + git push)
- `d`: delete worktree (press twice to confirm, esc to cancel)
- `q`: quit

### File List
- `j/k` or arrows: navigate
- `enter`: view diff
- `space`: toggle reviewed checkmark
- `esc`: back to worktree list

### Diff View
- `j/k` or arrows: scroll
- `f/b`: next/prev file
- `v`: toggle side-by-side / unified
- `space`: toggle reviewed checkmark
- `esc`: back to file list

## Tech Stack

Go + Bubble Tea (charmbracelet/bubbletea). No database. State is git worktrees.

## Project Structure

    cmd/wtr/main.go           Entry point
    internal/git/worktree.go  Worktree discovery
    internal/git/diff.go      Diff parsing
    internal/tui/app.go       Root Bubble Tea model
    internal/tui/worktree_list.go  Worktree list screen
    internal/tui/file_list.go      File list screen
    internal/tui/diff_view.go      Diff view (side-by-side + unified)
    internal/tui/styles.go         Lip Gloss styles
    internal/tui/keys.go           Key bindings
    internal/land/land.go          Land workflow
