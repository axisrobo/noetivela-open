package client

import (
	"bufio"
	"io"
	"strings"
)

// sseScanner reads SSE "data: <json>" lines and yields the JSON payloads.
// Non-data fields and comments are ignored.
type sseScanner struct {
	scanner *bufio.Scanner
}

func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{scanner: bufio.NewScanner(r)}
}

// Next returns the next data payload, or ok=false at stream end.
func (s *sseScanner) Next() (string, bool) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, "data:")), true
	}
	return "", false
}
