# Sessions demo - explicit session IDs, resume, and forking.
# Run from the repository root: vhs scripts/demo-tapes/sessions.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/sessions.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo sessions"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "/custom"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "What is 2+2?"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "/status"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/fork"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "What is 3+3?"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "/history"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "exit"
Sleep 500ms
Enter
Sleep 2s
