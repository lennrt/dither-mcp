# Compare three algorithms and generate a real local GIF loop.
Output "docs/assets/compare.gif"
Output "docs/assets/compare.mp4"
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
Type "# Compare textures with one reproducible palette."
Enter
Type "dither-mcp compare --root . --input source.png --output studies.png \"
Enter
Type "  --algorithms atkinson,floyd-steinberg,bayer-8 --palette sepia --width 320 | dither-summary"
Enter
Sleep 5s
Type "clear"
Enter
Type "# Create a 24-frame wave animation from the source image."
Enter
Type "dither-mcp animate --root . --input source.png --output wave.gif \"
Enter
Type "  --algorithm atkinson --palette gameboy --width 480 --pixel-scale 2 \"
Enter
Type "  --effect wave --frames 24 --fps 12 | dither-summary"
Enter
Sleep 7s
Type "# The local GIF is ready to inspect, export, or share."
Enter
Sleep 4s
