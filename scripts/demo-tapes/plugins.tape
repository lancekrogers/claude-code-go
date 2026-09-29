# Plugins demo - logging, metrics, and audit plugins.
# Run from the repository root: vhs scripts/demo-tapes/plugins.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/plugins.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo plugins"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "List files in current directory"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "metrics"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "audit"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "exit"
Sleep 500ms
Enter
Sleep 2s
