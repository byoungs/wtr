package tui

import (
	"fmt"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// diffScrollY tracks vertical scroll position for the diff view.
// Package-level var is fine for a single-instance TUI.
var diffScrollY int

func (a App) updateDiffView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if diffScrollY > 0 {
				diffScrollY--
			}
		case key.Matches(msg, keys.Down):
			diffScrollY++
		case key.Matches(msg, keys.NextFile):
			if a.selectedFile < len(a.files)-1 {
				a.selectedFile++
				diffScrollY = 0
			}
		case key.Matches(msg, keys.PrevFile):
			if a.selectedFile > 0 {
				a.selectedFile--
				diffScrollY = 0
			}
		case key.Matches(msg, keys.Space):
			k := a.reviewKey(a.selectedFile)
			a.reviewed[k] = !a.reviewed[k]
		case key.Matches(msg, keys.Toggle):
			a.sideBySide = !a.sideBySide
			diffScrollY = 0
		case key.Matches(msg, keys.Back):
			a.screen = screenFileList
			diffScrollY = 0
		}
	}
	return a, nil
}

func (a App) viewDiffView() string {
	if a.selectedFile >= len(a.files) {
		return "No file selected"
	}

	f := a.files[a.selectedFile]
	var b strings.Builder

	// Title bar
	name := f.NewName
	if name == "" || name == "/dev/null" {
		name = f.OldName
	}
	reviewed := ""
	if a.reviewed[a.reviewKey(a.selectedFile)] {
		reviewed = " ✓ reviewed"
	}
	viewMode := "side-by-side"
	if !a.sideBySide {
		viewMode = "unified"
	}
	title := styleTitle.Width(a.width).Render(
		fmt.Sprintf("%s  [%d/%d]%s  (%s)",
			name, a.selectedFile+1, len(a.files), reviewed, viewMode))
	b.WriteString(title + "\n")

	if f.Binary {
		b.WriteString("\n  Binary file\n")
	} else if a.sideBySide {
		b.WriteString(a.renderSideBySide(f))
	} else {
		b.WriteString(a.renderUnified(f))
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render(
		"  j/k: scroll  f/b: next/prev file  v: toggle view  space: reviewed  esc: back"))

	return b.String()
}

type sideBySidePair struct {
	oldNum     int
	oldContent string
	oldType    git.LineType
	newNum     int
	newContent string
	newType    git.LineType
	isHeader   bool
	headerText string
}

func (a App) renderSideBySide(f git.FileDiff) string {
	halfWidth := (a.width - 3) / 2 // -3 for " │ " divider
	if halfWidth < 20 {
		halfWidth = 20
	}

	var pairs []sideBySidePair

	for _, hunk := range f.Hunks {
		// Hunk header
		pairs = append(pairs, sideBySidePair{
			isHeader:   true,
			headerText: hunk.Header,
		})

		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		i := 0
		for i < len(hunk.Lines) {
			line := hunk.Lines[i]
			switch line.Type {
			case git.LineContext:
				pairs = append(pairs, sideBySidePair{
					oldNum: oldNum, oldContent: line.Content, oldType: git.LineContext,
					newNum: newNum, newContent: line.Content, newType: git.LineContext,
				})
				oldNum++
				newNum++
				i++

			case git.LineRemoved:
				// Collect consecutive removed lines
				var removed []git.DiffLine
				for i < len(hunk.Lines) && hunk.Lines[i].Type == git.LineRemoved {
					removed = append(removed, hunk.Lines[i])
					i++
				}
				// Collect consecutive added lines that follow
				var added []git.DiffLine
				for i < len(hunk.Lines) && hunk.Lines[i].Type == git.LineAdded {
					added = append(added, hunk.Lines[i])
					i++
				}
				// Pair them
				maxLen := len(removed)
				if len(added) > maxLen {
					maxLen = len(added)
				}
				for j := 0; j < maxLen; j++ {
					p := sideBySidePair{}
					if j < len(removed) {
						p.oldNum = oldNum
						p.oldContent = removed[j].Content
						p.oldType = git.LineRemoved
						oldNum++
					}
					if j < len(added) {
						p.newNum = newNum
						p.newContent = added[j].Content
						p.newType = git.LineAdded
						newNum++
					}
					pairs = append(pairs, p)
				}

			case git.LineAdded:
				pairs = append(pairs, sideBySidePair{
					newNum: newNum, newContent: line.Content, newType: git.LineAdded,
				})
				newNum++
				i++
			}
		}
	}

	// Apply scroll and viewport
	viewHeight := a.height - 5 // title + help + padding
	if viewHeight < 1 {
		viewHeight = 1
	}
	maxScroll := len(pairs) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if diffScrollY > maxScroll {
		diffScrollY = maxScroll
	}
	end := diffScrollY + viewHeight
	if end > len(pairs) {
		end = len(pairs)
	}
	visible := pairs[diffScrollY:end]

	lineNumWidth := 4
	contentWidth := halfWidth - lineNumWidth - 2

	var b strings.Builder
	divider := lipgloss.NewStyle().Foreground(colorSubtle).Render("│")

	for _, p := range visible {
		if p.isHeader {
			header := styleHunkHeader.Render(p.headerText)
			b.WriteString(header + "\n")
			continue
		}

		left := formatHalf(p.oldNum, p.oldContent, p.oldType, lineNumWidth, contentWidth)
		right := formatHalf(p.newNum, p.newContent, p.newType, lineNumWidth, contentWidth)
		b.WriteString(left + " " + divider + " " + right + "\n")
	}

	return b.String()
}

func formatHalf(num int, content string, lineType git.LineType, lineNumWidth, contentWidth int) string {
	// Line number
	numStr := strings.Repeat(" ", lineNumWidth)
	if num > 0 {
		numStr = fmt.Sprintf("%*d", lineNumWidth, num)
	}
	numStyled := lipgloss.NewStyle().Foreground(colorSubtle).Render(numStr)

	// Truncate or pad content
	if len(content) > contentWidth {
		content = content[:contentWidth-1] + "…"
	}
	padded := content + strings.Repeat(" ", max(0, contentWidth-len(content)))

	// Style based on line type
	var styled string
	switch lineType {
	case git.LineAdded:
		styled = styleAdded.Render(padded)
	case git.LineRemoved:
		styled = styleRemoved.Render(padded)
	default:
		styled = styleContext.Render(padded)
	}

	return numStyled + " " + styled
}

func (a App) renderUnified(f git.FileDiff) string {
	viewHeight := a.height - 5
	if viewHeight < 1 {
		viewHeight = 1
	}

	var allLines []string

	for _, hunk := range f.Hunks {
		allLines = append(allLines, styleHunkHeader.Render(hunk.Header))
		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		for _, line := range hunk.Lines {
			var prefix, numStr string
			switch line.Type {
			case git.LineContext:
				prefix = " "
				numStr = fmt.Sprintf("%4d %4d", oldNum, newNum)
				oldNum++
				newNum++
			case git.LineRemoved:
				prefix = "-"
				numStr = fmt.Sprintf("%4d     ", oldNum)
				oldNum++
			case git.LineAdded:
				prefix = "+"
				numStr = fmt.Sprintf("     %4d", newNum)
				newNum++
			}

			numStyled := lipgloss.NewStyle().Foreground(colorSubtle).Render(numStr)
			var contentStyled string
			switch line.Type {
			case git.LineAdded:
				contentStyled = styleAdded.Render(prefix + line.Content)
			case git.LineRemoved:
				contentStyled = styleRemoved.Render(prefix + line.Content)
			default:
				contentStyled = styleContext.Render(prefix + line.Content)
			}
			allLines = append(allLines, numStyled+" "+contentStyled)
		}
	}

	// Apply scroll
	maxScroll := len(allLines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if diffScrollY > maxScroll {
		diffScrollY = maxScroll
	}
	end := diffScrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}
	visible := allLines[diffScrollY:end]

	return strings.Join(visible, "\n")
}
