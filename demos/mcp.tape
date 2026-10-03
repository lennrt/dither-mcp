# The scripted client starts and calls the actual stdio MCP server.
Output "docs/assets/mcp.gif"
Output "docs/assets/mcp.mp4"
Require go
Require ffmpeg
Require ttyd
Set Shell "bash"
Set Width 1200
Set Height 750
Set FontSize 19
Set FontFamily "Menlo"
Set Padding 28
Set Margin 32
Set MarginFill "#f0eee6"
Set WindowBar "Rings"
Set TypingSpeed 30ms
Set Framerate 20
Set Theme {"name":"Dither clay","background":"#faf9f5","foreground":"#141413","cursor":"#d97757","black":"#141413","red":"#a23e2b","green":"#3d3d3a","yellow":"#765329","blue":"#3d3d3a","magenta":"#6c4454","cyan":"#3d3d3a","white":"#e3dacc","brightBlack":"#5e5d58","brightRed":"#a23e2b","brightGreen":"#3d3d3a","brightYellow":"#765329","brightBlue":"#3d3d3a","brightMagenta":"#6c4454","brightCyan":"#3d3d3a","brightWhite":"#f0eee6"}
Hide
Type "set -o pipefail; export PS1='dither > '; clear"
Enter
Sleep 500ms
Show
Type "# The client discovers tools, renders images, and saves a recipe."
Enter
Sleep 700ms
Type "go run ./scripts/mcp-demo"
Enter
Sleep 12s
Type "# The Go client uses a local stdio connection."
Enter
Sleep 4s
