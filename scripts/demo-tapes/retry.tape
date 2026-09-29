# Retry demo - retry policies and error handling.
# Run from the repository root: vhs scripts/demo-tapes/retry.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/retry.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo retry"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "/status"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/aggressive"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/status"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/test-retry"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "/stats"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "exit"
Sleep 500ms
Enter
Sleep 2s
