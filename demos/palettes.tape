# Discover catalog entries and render with a chosen palette.
Output "docs/assets/palettes.gif"
Output "docs/assets/palettes.mp4"
Require dither-mcp
Require dither-summary
Require ffmpeg
Require ttyd
Set Shell "bash"
Set Width 1200
Set Height 750
Set FontSize 18
Set FontFamily "Menlo"
Set Padding 28
Set Margin 32
Set MarginFill "#f0eee6"
Set WindowBar "Rings"
Set TypingSpeed 20ms
Set Framerate 20
Set Theme {"name":"Dither clay","background":"#faf9f5","foreground":"#141413","cursor":"#d97757","black":"#141413","red":"#a23e2b","green":"#3d3d3a","yellow":"#765329","blue":"#3d3d3a","magenta":"#6c4454","cyan":"#3d3d3a","white":"#e3dacc","brightBlack":"#5e5d58","brightRed":"#a23e2b","brightGreen":"#3d3d3a","brightYellow":"#765329","brightBlue":"#3d3d3a","brightMagenta":"#6c4454","brightCyan":"#3d3d3a","brightWhite":"#f0eee6"}
Hide
Type "set -o pipefail; ./scripts/demo-prepare.sh; cd demos/work; export PS1='dither > '; clear"
Enter
Sleep 500ms
Show
Type "# Find a palette by category and color count."
Enter
Type "dither-mcp palettes --category botanical --min-colors 4 --max-colors 6 \"
Enter
Type "  --limit 3 | dither-summary --palettes"
Enter
Sleep 5s
Type "clear"
Enter
Type "# Search the 256-palette library with descriptive words."
Enter
Type "dither-mcp palettes --query 'oat ink' --limit 3 | dither-summary --palettes"
Enter
Sleep 4s
Type "dither-mcp render --root . --input source.png --output oat-garden.png \"
Enter
Type "  --algorithm floyd-steinberg --palette oat-and-ink \"
Enter
Type "  --width 480 --pixel-scale 2 | dither-summary"
Enter
Sleep 5s
Type "# The render result includes a repeatable recipe."
Enter
Sleep 4s
