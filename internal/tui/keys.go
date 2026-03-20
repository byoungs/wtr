package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Land     key.Binding
	Delete   key.Binding
	Test     key.Binding
	Toggle   key.Binding
	Space    key.Binding
	NextFile key.Binding
	PrevFile key.Binding
	NextHunk key.Binding
	PrevHunk key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Land:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "land")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete worktree")),
	Test:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "run tests")),
	Toggle:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle view")),
	Space:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "mark reviewed")),
	NextFile: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "next file")),
	PrevFile: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "prev file")),
	NextHunk: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next hunk")),
	PrevHunk: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev hunk")),
}
