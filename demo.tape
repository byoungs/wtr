Output docs/demo.gif

Set Shell zsh
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 10
Set Theme "Catppuccin Mocha"

# Launch wtr pointing at ai-scheduler
Type "wtr ~/src/ai-scheduler"
Enter
Sleep 3s

# worktree-four should be visible — navigate to it
# (it may already be selected as first item)
# Press enter to review files
Enter
Sleep 2s

# Scroll down through the file list
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 2s

# Open a diff (enter on selected file)
Enter
Sleep 2s

# Scroll through the diff
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Space
Sleep 1s
Space
Sleep 2s

# Toggle side-by-side view
Type "v"
Sleep 2s

# Scroll in side-by-side
Space
Sleep 1s
Space
Sleep 2s

# Go back to file list
Left
Sleep 1s

# View all diffs
Type "a"
Sleep 2s

# Scroll through all diffs
Space
Sleep 1s
Space
Sleep 1s
Space
Sleep 2s

# Go back
Left
Sleep 1s

# Go back to worktree list
Left
Sleep 1s

# Quit
Type "q"
Sleep 500ms
