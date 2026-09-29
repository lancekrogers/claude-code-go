# Subagents demo - switching between specialized agents.
# Run from the repository root: vhs scripts/demo-tapes/subagents.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/subagents.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo subagents"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "agents"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "@security Summarize security best practices"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "@docs What makes good documentation?"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "exit"
Sleep 500ms
Enter
Sleep 2s
