# Shared VHS settings for every demo recording.
# Sourced by each <demo>.tape; tapes run from the repository root.

Set Shell "bash"
Set FontFamily "JetBrains Mono, JetBrainsMono Nerd Font, Menlo, monospace"
Set FontSize 16
Set Width 1280
Set Height 760
Set Padding 24
Set Margin 20
Set MarginFill "#1e1e2e"
Set WindowBar Colorful
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set PlaybackSpeed 1.0
Set Framerate 24
Set CursorBlink false

# Real Claude Code calls can take a while to answer.
Set WaitTimeout 180s
Set WaitPattern />>>\s*$/
