# Streaming demo - real-time message streaming. We decline the proposed task to keep it short.
# Run from the repository root: vhs scripts/demo-tapes/streaming.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/streaming.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo streaming"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "no thanks, just demonstrate"
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
