package wtr

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
var diffTotalLines int      // set during render, used to detect "scrolled to bottom"
var diffHunkPositions []int // line indices of hunk headers, set during render

// minWrapWidth is the smallest terminal width at which unified diff lines are
// soft-wrapped. Below this, long lines overflow (same as before). Wrapping a
// line into 5+ visual rows on a very narrow terminal creates more confusion
// than overflow does, so the threshold opts out of wrapping when the gutter
// would leave too little content room to be useful.
const minWrapWidth = 80

// unifiedGutterWidth is the number of columns consumed by the line-number
// gutter in unified diffs: two 4-wide numbers, a space between them, and a
// trailing space before the content ("1234 5678 ").
const unifiedGutterWidth = 10

// wrapContent splits content into rune-chunks of at most width runes.
// Returns a single-element slice ([]string{content}) when wrapping is disabled
// or unnecessary. Wraps by rune count so multi-byte characters stay intact.
func wrapContent(content string, width int, wrap bool) []string {
	if !wrap || width <= 0 {
		return []string{content}
	}
	runes := []rune(content)
	if len(runes) <= width {
		return []string{content}
	}
	var chunks []string
	for i := 0; i < len(runes); i += width {
		end := i + width
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

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
		case key.Matches(msg, keys.PageDown):
			viewHeight := a.height - 3
			if viewHeight < 1 {
				viewHeight = 1
			}
			diffScrollY += viewHeight
		case key.Matches(msg, keys.PageUp):
			viewHeight := a.height - 3
			if viewHeight < 1 {
				viewHeight = 1
			}
			diffScrollY -= viewHeight
			if diffScrollY < 0 {
				diffScrollY = 0
			}
		case key.Matches(msg, keys.Right):
			// Right arrow: page down if more content, next file if at bottom
			viewHeight := a.height - 3
			if viewHeight < 1 {
				viewHeight = 1
			}
			if a.diffAtBottom(viewHeight) {
				// At bottom — mark reviewed and go to next file
				a.markCurrentReviewed()
				if a.selectedFile < len(a.files)-1 {
					a.selectedFile++
					diffScrollY = 0
					diffTotalLines = 0
				}
			} else {
				diffScrollY += viewHeight
			}
		case key.Matches(msg, keys.NextFile):
			// Mark current file as reviewed before advancing
			a.markCurrentReviewed()
			if a.selectedFile < len(a.files)-1 {
				a.selectedFile++
				diffScrollY = 0
				diffTotalLines = 0
			}
		case key.Matches(msg, keys.PrevFile):
			if a.selectedFile > 0 {
				a.selectedFile--
				diffScrollY = 0
				diffTotalLines = 0
			}
		case key.Matches(msg, keys.NextHunk):
			// Jump to next hunk header after current scroll position
			for _, pos := range diffHunkPositions {
				if pos > diffScrollY {
					diffScrollY = pos
					break
				}
			}
		case key.Matches(msg, keys.PrevHunk):
			// Jump to previous hunk header before current scroll position
			prev := 0
			for _, pos := range diffHunkPositions {
				if pos >= diffScrollY {
					break
				}
				prev = pos
			}
			diffScrollY = prev
		case key.Matches(msg, keys.MarkReviewed):
			k := a.reviewKey(a.selectedFile)
			a.reviewed[k] = !a.reviewed[k]
			if a.reviewed[k] && a.selectedWorktree < len(a.worktrees) {
				wt := a.worktrees[a.selectedWorktree]
				a.reviewedAt[wt.Branch] = wt.CommitHash
			}
			a.saveState()
		case key.Matches(msg, keys.Toggle):
			a.sideBySide = !a.sideBySide
			diffScrollY = 0
		case key.Matches(msg, keys.Back):
			if a.prevScreen == screenGitStatus {
				a.screen = screenGitStatus
			} else {
				a.screen = screenFileList
			}
			diffScrollY = 0
		}

		// Auto-mark reviewed if scrolled to bottom or content fits on one page
		a.autoMarkReviewed()
	}
	return a, nil
}

// diffAtBottom returns true if the diff is scrolled to the bottom or fits on one page.
func (a App) diffAtBottom(viewHeight int) bool {
	if diffTotalLines == 0 {
		return false
	}
	return diffTotalLines <= viewHeight || diffScrollY >= diffTotalLines-viewHeight
}

// markCurrentReviewed marks the currently selected file as reviewed and persists.
func (a *App) markCurrentReviewed() {
	if a.selectedFile >= len(a.files) {
		return
	}
	k := a.reviewKey(a.selectedFile)
	if !a.reviewed[k] {
		a.reviewed[k] = true
		// Track which commit this review was for
		if a.selectedWorktree < len(a.worktrees) {
			wt := a.worktrees[a.selectedWorktree]
			a.reviewedAt[wt.Branch] = wt.CommitHash
		}
		a.saveState()
	}
}

func (a *App) autoMarkReviewed() {
	if a.selectedFile >= len(a.files) {
		return
	}
	if diffTotalLines == 0 {
		return // not rendered yet — don't make assumptions
	}
	viewHeight := a.height - 3
	if viewHeight < 1 {
		viewHeight = 1
	}
	if a.diffAtBottom(viewHeight) {
		a.markCurrentReviewed()
	}
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

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	b.WriteString(styleHelp.Render(
		"  q:quit  ←back  space/b:page  ][:file  n/p:hunk  v:toggle  x:reviewed"))

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
	viewHeight := a.height - 3 // title + help + padding
	if viewHeight < 1 {
		viewHeight = 1
	}
	diffTotalLines = len(pairs)
	diffHunkPositions = nil
	for i, p := range pairs {
		if p.isHeader {
			diffHunkPositions = append(diffHunkPositions, i)
		}
	}
	maxScroll := diffTotalLines - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if diffScrollY > maxScroll {
		diffScrollY = maxScroll
	}
	end := diffScrollY + viewHeight
	if end > diffTotalLines {
		end = diffTotalLines
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
	viewHeight := a.height - 3
	if viewHeight < 1 {
		viewHeight = 1
	}

	wrap := a.width >= minWrapWidth
	contentWidth := a.width - unifiedGutterWidth

	var allLines []string
	var hunkPositions []int

	for _, hunk := range f.Hunks {
		hunkPositions = append(hunkPositions, len(allLines))
		allLines = append(allLines, styleHunkHeader.Render(hunk.Header))
		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		for _, line := range hunk.Lines {
			var prefix, numStr string
			// Gutter format: 9 chars (two %4d separated by a space). Combined
			// with the trailing space in renderUnifiedDiffLine, this must
			// match unifiedGutterWidth — widen both together if files with
			// >9999 lines become a concern.
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

			allLines = append(allLines, renderUnifiedDiffLine(numStr, prefix, line.Content, line.Type, contentWidth, wrap)...)
		}
	}

	// Apply scroll
	diffTotalLines = len(allLines)
	diffHunkPositions = hunkPositions
	maxScroll := diffTotalLines - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if diffScrollY > maxScroll {
		diffScrollY = maxScroll
	}
	end := diffScrollY + viewHeight
	if end > diffTotalLines {
		end = diffTotalLines
	}
	visible := allLines[diffScrollY:end]

	return strings.Join(visible, "\n")
}

// renderUnifiedDiffLine produces one or more visual lines for a single unified
// diff line. When wrap is enabled and prefix+content exceeds contentWidth, the
// output splits into multiple rows. The first row shows the line-number gutter
// and the prefix character; continuation rows show a blank gutter and carry
// the same color styling so wrapped content is still identifiable as
// added/removed/context.
func renderUnifiedDiffLine(numStr, prefix, content string, lineType git.LineType, contentWidth int, wrap bool) []string {
	numStyled := lipgloss.NewStyle().Foreground(colorSubtle).Render(numStr)
	blankGutter := lipgloss.NewStyle().Foreground(colorSubtle).Render(strings.Repeat(" ", len(numStr)))

	style := styleContext
	switch lineType {
	case git.LineAdded:
		style = styleAdded
	case git.LineRemoved:
		style = styleRemoved
	}

	chunks := wrapContent(prefix+content, contentWidth, wrap)
	lines := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		gutter := numStyled
		if i > 0 {
			gutter = blankGutter
		}
		lines = append(lines, gutter+" "+style.Render(chunk))
	}
	return lines
}
