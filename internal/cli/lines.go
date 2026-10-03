package cli

import (
	"errors"
	"io"
)

// lineLimitReader bounds newline-delimited MCP messages before the protocol
// library buffers or parses them. A message that exceeds the limit terminates the session.
type lineLimitReader struct {
	source io.Reader
	count  int
	failed bool
}

const maxMessageBytes = 2 << 20

func (r *lineLimitReader) Read(p []byte) (int, error) {
	if r.failed {
		return 0, errors.New("MCP message exceeds 2 MiB")
	}
	n, err := r.source.Read(p)
	for i, c := range p[:n] {
		if c == '\n' {
			r.count = 0
		} else {
			r.count++
			if r.count > maxMessageBytes {
				r.failed = true
				return i, errors.New("MCP message exceeds 2 MiB")
			}
		}
	}
	return n, err
}
