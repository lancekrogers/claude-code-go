# Permissions demo - tool access control presets.
# Run from the repository root: vhs scripts/demo-tapes/permissions.tape
#
# Steps that call Claude are time-lapsed: the prompt is shown being sent,
# frames are hidden while Claude answers, and recording resumes on the reply.

Output docs/gif/permissions.gif
Source scripts/demo-tapes/_settings.tape
Source scripts/demo-tapes/_setup.tape

Type "just demo permissions"
Sleep 500ms
Enter
Sleep 1.5s
Hide
Wait
Show
Sleep 3s

Type "/presets"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/preset readonly"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/tools"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "/status"
Sleep 500ms
Enter
Wait
Sleep 2.5s

Type "exit"
Sleep 500ms
Enter
Sleep 2s
