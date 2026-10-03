# Record the CLI from the repository root.
Output "docs/assets/render.gif"
Output "docs/assets/render.mp4"
Require dither-mcp
Require dither-summary
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
Set TypingSpeed 22ms
Set Framerate 20
Set Theme {"name":"Dither clay","background":"#faf9f5","foreground":"#141413","cursor":"#d97757","black":"#141413","red":"#a23e2b","green":"#3d3d3a","yellow":"#765329","blue":"#3d3d3a","magenta":"#6c4454","cyan":"#3d3d3a","white":"#e3dacc","brightBlack":"#5e5d58","brightRed":"#a23e2b","brightGreen":"#3d3d3a","brightYellow":"#765329","brightBlue":"#3d3d3a","brightMagenta":"#6c4454","brightCyan":"#3d3d3a","brightWhite":"#f0eee6"}
Hide
Type "set -o pipefail; ./scripts/demo-prepare.sh; cd demos/work; export PS1='dither > '; clear"
Enter
Sleep 500ms
Show
Type "# Render one local image with four colors."
Enter
Sleep 700ms
Type "dither-mcp inspect --root . --input source.png | dither-summary --inspect"
Enter
Sleep 3s
Type "dither-mcp render --root . --input source.png --output garden.png \"
Enter
Type "  --algorithm atkinson --palette gameboy --width 960 --pixel-scale 2 | dither-summary"
Enter
Sleep 6s
Type "# The result includes the recipe and a SHA-256 digest."
Enter
Sleep 4s
