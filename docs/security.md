# Security and operational model

The trusted operator chooses an existing workspace root and an MCP client. The client can request operations on files beneath that root. Treat that scope as a capability. Use a dedicated images directory. Avoid your home directory and trees that contain secrets. The tool does not decide whether a photograph is sensitive.

## Filesystem and publication

- The service accepts only relative local paths. It rejects absolute paths, parent traversal, URLs, backslashes, NUL, and paths that contain colons.
- Go `os.Root` anchors access to open directory handles. Symlinks cannot escape the root. The service allows relative symlinks to files inside it.
- Inputs must be regular files. POSIX opens use `O_NONBLOCK` and then recheck the opened descriptor. A FIFO substitution therefore cannot block the service.
- The service limits reads before and during loading. It validates image headers before image allocation. It scans GIF block metadata before allocating all frames.
- Publication uses a random owner-only temporary file in the destination's opened parent directory. The service writes the complete file, syncs it, and closes it. A hard link publishes it only if the destination does not exist. The service removes the temporary name on all ordinary success and error paths.
- The service never replaces existing files, directories, or symlinks. There is no MCP delete tool or overwrite flag. A failed call can create an output parent directory. The final output file will not be partially visible.

Atomic publication requires a local filesystem with hard-link support. Unsupported filesystems fail without attempting an unsafe overwrite. A successful link is the publication point. Cancellation that arrives after the final check may race with that completed publication. The service does not fsync directory entries and does not claim power-loss durability.

An abruptly killed process can leave a private `.dither-*.tmp` file. An operator may remove it after confirming that no writer is active. When atomic local publication matters, do not use a live output directory on a network or cloud-sync filesystem.

`os.Root` does not block mount-point traversal, bind mounts, or hostile changes by a privileged local actor. It provides a path boundary, not an OS sandbox. The service cannot protect its process from the account that owns and debugs it. Avoid roots with special mounts. Run the service under an ordinary unprivileged account.

## Resource limits

The [MCP guide](mcp.md#limits-and-errors) lists enforced limits. The service serializes calls at admission to cap aggregate image allocations. It bounds input bytes, image pixels, frame count, aggregate motion pixels, and encoded output. Each call has a two-minute context deadline. Cancellation terminates subprocesses. Image loops check the context.

Standard-library image decode/encode calls cannot be preempted mid-call. The deadline is therefore cooperative, not a hard real-time CPU or RSS ceiling. Use OS memory and CPU limits for hostile multi-tenant workloads. The service caps MCP framing at 2 MiB before JSON parsing. It caps argument objects at 1 MiB. An oversized message ends the connection.

Batch results commit independently. A later failure or cancellation does not reverse earlier outputs. Inspect every returned item. A comparison or print archive produces one final artifact. The service publishes it only after all components complete.

## MCP Apps studio

The [studio](mcp-apps.md) is an optional view in a supporting MCP host. The Go binary serves one embedded HTML resource, `ui://dither/studio.html`, with MIME type `text/html;profile=mcp-app`. Its CSS and official MCP Apps SDK are bundled. It loads no external assets, uses no CDN, and requires no runtime Node.js process or additional server listener.

The host controls the app's sandbox and mediates tool calls through the MCP Apps bridge. App requests still pass through the service's rooted paths, strict arguments, admission queue, cancellation, and resource limits. The resource requests `data:` resources for its inline PNG previews. It requests no camera, microphone, external resource origins, or network connections.

`dither_studio` is read-only. It creates an in-memory PNG and returns the resolved recipe. It accepts no destination and publishes no artifact. The default preview fits within 512 × 512 pixels without enlargement. Every preview stays within 1,024 pixels per axis and 2 MiB PNG.

Only **Save image** in the app requests a file write. The app calls `dither_render` with a new relative output path and the accepted recipe, explicit export dimensions, and source/mask fingerprints. Unsaved control changes and failed previews do not replace that recipe. The normal atomic publication rules protect existing destinations. The host continues to control authorization for tool calls.

## Input metadata

The still-image loader parses recognized EXIF and color metadata within the input byte budget. It limits EXIF payloads and expanded ICC profiles to 4 MiB each. ICC profiles permit at most 256 tags and 65,536 samples per tone curve. EXIF IFD0 permits at most 4,096 entries. Malformed recognized container metadata and invalid or unsupported selected profiles cause errors before artifact publication. Image masks use the same validation and normalization path.

The loader applies EXIF orientation and converts supported color information to sRGB before the engine processes pixels. Profile conversion uses Go code and local data. It does not invoke a native color-management executable or fetch a profile. The loader rejects linked BMP profiles and never follows their external file references. See [formats](formats.md#orientation-and-input-color) for the accepted profile subset and PNG precedence.

## Optional media subprocesses

Video processing requires `--allow-video`. Enabled hosts must provide trusted `ffmpeg` and `ffprobe` executables through PATH. The server never accepts an executable or a raw command argument from a tool request. It never invokes a shell.

The service stages one bounded input in a private temporary directory. It admits only recognized binary MP4/MOV, WebM/Matroska, and AVI containers. It applies these controls:

- Force the matching demuxer.
- Restrict protocols to file/pipe.
- Disable MOV external data references and absolute data-reference paths.
- Pass fixed frame, duration, thread, and pixel controls.
- Set a 256 MiB maximum for each FFmpeg/ffprobe allocation.

The allocation limit is not a total memory limit. Video output omits all audio. Before publication, the service checks output byte size and confirms that the frame count is complete.

FFmpeg remains a native-code parser, not an isolated sandbox. It can use temporary disk space and codec-dependent resources. Malicious codec bugs remain possible. Keep FFmpeg updated. Use process or container isolation when processing hostile media.

The service removes temporary staging after each ordinary completion or error. The tool has no camera, microphone, network-stream, or arbitrary-filtergraph input.

## Privacy

The Go application does not fetch URLs, upload images, collect analytics, or call an LLM provider. A normal still operation needs no network. Dependency downloads and vulnerability checks are development activities.

The MCP client receives metadata. `dither_preview` and `dither_studio` explicitly send PNG previews to that client. Studio results also include the resolved recipe. The client may forward these results to a hosted model under its own policy. Local processing does not guarantee anything about the client's transmission or retention.

Inspection also reports orientation, stored dimensions, color-source identifiers, conversion status, and the selected ICC profile’s digest when applicable. It does not return the full embedded profile. Exports omit source EXIF and ICC data.

Metadata includes relative paths and processing parameters. Keep logs and demo recordings free of private source names. The included artwork comes from original procedural scenes. The project includes no private images or credentials.

## Assurance and limits

Quint models specify finite abstractions of publication and quantization. Seeded simulation and mutation checks exercise those models. They do not exhaustively verify the Go implementation, filesystems, codecs, or FFmpeg.

Go tests cover concrete security and image behavior. This includes publication collisions between two services, symlink escapes, GIF disposal, and format limits. The project has not undergone an independent penetration test or external security audit. See [testing](testing.md) for reproducible commands and evidence from this local build.

## Guarded studio exports

A studio preview returns SHA-256 fingerprints of the source and mask bytes it decoded. A save verifies the same captured buffers it will render. A mismatch creates no artifact and requires a fresh preview. Source-size export uses upright dimensions after crop; custom export requires both dimensions. The service enforces its usual dimension and pixel limits. The [embedding example](embedding.md) adds a separate loopback host with a narrow tool allowlist; it is optional and is not part of the Go server runtime.
