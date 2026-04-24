package wtr

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Back         key.Binding
	Quit         key.Binding
	Land         key.Binding
	Delete       key.Binding
	Test         key.Binding
	ViewOutput   key.Binding
	Toggle       key.Binding
	MarkReviewed key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	NextFile     key.Binding
	PrevFile     key.Binding
	NextHunk     key.Binding
	PrevHunk     key.Binding
	AllDiffs     key.Binding
	Squash       key.Binding
	Rebase       key.Binding
	Help         key.Binding
	Refresh      key.Binding
	Open         key.Binding
	GitStatus    key.Binding
	Revert       key.Binding
	Right        key.Binding
	Search       key.Binding
	DevServer    key.Binding
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:  key.NewBinding(key.WithKeys("esc", "left"), key.WithHelp("esc/←", "back")),

	Quit:         key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Land:         key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "land")),
	Delete:       key.NewBinding(key.WithKeys("delete", "backspace"), key.WithHelp("del", "delete worktree")),
	Test:         key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "test e2e")),
	ViewOutput:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "view output")),
	Toggle:       key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle view")),
	MarkReviewed: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "mark reviewed")),
	PageDown:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "page down")),
	PageUp:       key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "page up")),
	NextFile:     key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next file")),
	PrevFile:     key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev file")),
	NextHunk:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next hunk")),
	PrevHunk:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev hunk")),
	AllDiffs:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all diffs")),
	Squash:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "squash")),
	Rebase:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rebase")),
	Help:         key.NewBinding(key.WithKeys("h", "?"), key.WithHelp("h/?", "help")),
	Refresh:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
	Open:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit in VS Code")),
	GitStatus:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "git status")),
	Revert:       key.NewBinding(key.WithKeys("delete", "backspace"), key.WithHelp("del", "revert/delete")),
	Right:        key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "drill in")),
	Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search files")),
	DevServer:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "make dev")),
}
