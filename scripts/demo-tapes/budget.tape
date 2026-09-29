# Budget demo - cost tracking across requests.
# Run from the repository root: vhs scripts/demo-tapes/budget.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/budget.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo budget"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "What is the capital of France?"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 4s

Type "List 3 programming languages"
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
