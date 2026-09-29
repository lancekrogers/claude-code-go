# Hidden shell setup shared by every demo tape.
# Gives the recording a clean prompt, keeps the recorder's claude.ai
# connectors (Drive, Gmail, ...) out of Claude's replies, and pre-builds
# every demo so compile output never shows up on camera.

Hide
Type `export PS1='\[\e[1;35m\]❯\[\e[0m\] ' ENABLE_CLAUDEAI_MCP_SERVERS=false`
Enter
Type "just demo build-all >/dev/null 2>&1; clear"
Enter
Wait+Line@300s /❯\s*$/
Show
