package wtr

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a App) viewHelp() string {
	var b strings.Builder

	title := styleTitle.Width(a.width).Render("Help")
	b.WriteString(title + "\n\n")

	heading := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	label := lipgloss.NewStyle().Foreground(colorText).Width(14)
	desc := lipgloss.NewStyle().Foreground(colorSubtle)

	// Status indicators
	b.WriteString(heading.Render("  Status Indicators") + "\n\n")

	b.WriteString("  " + stylePass.Render("↑N") + " " + desc.Render("N commits ahead of "+a.defaultBranch+" (green = up to date)") + "\n")
	b.WriteString("  " + styleRunning.Render("↓N") + " " + desc.Render("Behind "+a.defaultBranch+" by N commits — press r to rebase") + "\n")
	b.WriteString("  " + stylePending.Render("(no commits)") + " " + desc.Render("Branch exists but no changes vs "+a.defaultBranch) + "\n")
	b.WriteString("  " + stylePass.Render("✓") + "  " + desc.Render("Tests/validation passed") + "\n")
	b.WriteString("  " + styleFail.Render("✗") + "  " + desc.Render("Tests/validation failed") + "\n")
	b.WriteString("  " + styleRunning.Render("⟳") + "  " + desc.Render("Tests/validation running") + "\n")
	b.WriteString("  " + styleRunning.Render("△N") + " " + desc.Render("N uncommitted changes in worktree") + "\n")

	b.WriteString("\n")

	var helpKeys [][2]string

	if a.mode == "direct" {
		b.WriteString(heading.Render("  Review (Direct Mode)") + "\n\n")
		directKeys := [][2]string{
			{"→ / enter", "Review changed files"},
			{"g", "Git status (uncommitted changes)"},
			{"t", "Run make validate (background)"},
			{"o", "View test/validate output"},
			{"l", "Push to origin (test + validate + push)"},
			{"u", "Refresh"},
			{"h / ?", "This help screen"},
			{"q", "Quit"},
		}
		for _, kv := range directKeys {
			b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
		}
	} else {
		b.WriteString(heading.Render("  Worktree List") + "\n\n")

		helpKeys = [][2]string{
			{"→ / enter", "Review files in worktree"},
			{"e", "Edit worktree in VS Code"},
			{"t", "Run make test e2e (background)"},
			{"o", "View test/validate output"},
			{"r", "Rebase on " + a.defaultBranch},
			{"s", "Squash to 1 commit on " + a.defaultBranch},
			{"l", "Land (merge + test + validate + push)"},
			{"u", "Refresh worktree state"},
			{"del", "Delete worktree"},
			{"h / ?", "This help screen"},
			{"q", "Quit"},
		}
		for _, kv := range helpKeys {
			b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(heading.Render("  File List") + "\n\n")

	helpKeys = [][2]string{
		{"→ / enter", "View diff for file"},
		{"e", "Edit file in VS Code"},
		{"a", "View all diffs (integrated)"},
		{"/", "Search/filter files"},
		{"g", "Git status (uncommitted changes)"},
		{"x", "Toggle reviewed checkmark"},
		{"← / esc", "Back to worktree list"},
	}
	for _, kv := range helpKeys {
		b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(heading.Render("  Diff View") + "\n\n")

	helpKeys = [][2]string{
		{"space / b", "Page down / up"},
		{"] / [", "Next / prev file"},
		{"n / p", "Next / prev hunk"},
		{"v", "Toggle side-by-side / unified"},
		{"x", "Toggle reviewed checkmark"},
		{"← / esc", "Back"},
	}
	for _, kv := range helpKeys {
		b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(heading.Render("  Git Status") + "\n\n")

	helpKeys = [][2]string{
		{"→ / enter", "View diff for file"},
		{"del", "Revert / delete file"},
		{"e", "Edit file in VS Code"},
		{"← / esc", "Back to file list"},
	}
	for _, kv := range helpKeys {
		b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("  Press any key to return"))

	return b.String()
}
