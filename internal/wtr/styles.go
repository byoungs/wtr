package wtr

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors (Catppuccin Mocha palette)
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorRed       = lipgloss.Color("#f38ba8")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorBlue      = lipgloss.Color("#89b4fa")
	colorSubtle    = lipgloss.Color("#6c7086")
	colorText      = lipgloss.Color("#cdd6f4")
	colorHighlight = lipgloss.Color("#313244")

	// Diff line styles
	styleAdded      = lipgloss.NewStyle().Foreground(colorGreen)
	styleRemoved    = lipgloss.NewStyle().Foreground(colorRed)
	styleContext    = lipgloss.NewStyle().Foreground(colorText)
	styleHunkHeader = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)

	// List styles
	styleSelected = lipgloss.NewStyle().Background(colorHighlight).Bold(true)
	styleNormal   = lipgloss.NewStyle().Foreground(colorText)
	styleReviewed = lipgloss.NewStyle().Foreground(colorGreen)

	// Status indicators
	stylePass    = lipgloss.NewStyle().Foreground(colorGreen)
	styleFail    = lipgloss.NewStyle().Foreground(colorRed)
	styleRunning = lipgloss.NewStyle().Foreground(colorYellow)
	stylePending = lipgloss.NewStyle().Foreground(colorSubtle)

	// Layout
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorBlue).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorSubtle)
	styleHelp = lipgloss.NewStyle().Foreground(colorSubtle)
)

// padToBottom adds blank lines so the help bar sits at the bottom of the terminal.
// linesUsed is the number of content lines already written (including title).
func padToBottom(b *strings.Builder, height, linesUsed int) {
	// Reserve 1 line for help bar
	for i := linesUsed; i < height-1; i++ {
		b.WriteString("\n")
	}
}
