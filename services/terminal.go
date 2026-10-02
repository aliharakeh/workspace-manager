package services

import (
	"bytes"
	"regexp"
	"strings"
)

const (
	// maxTermBuffer is how much recent output of a command is kept for a UI that
	// opens late. The buffer may grow to twice this before it is trimmed.
	maxTermBuffer = 1 << 20
	// maxTermLine bounds the text kept while waiting for a newline.
	maxTermLine = 64 * 1024
)

// termBuffer keeps the most recent output of one terminal and counts all of it.
type termBuffer struct {
	data     []byte
	total    int64
	lastByte byte
}

// add appends p and returns the offset p starts at.
func (b *termBuffer) add(p []byte) int64 {
	start := b.total
	if len(p) == 0 {
		return start
	}
	b.data = append(b.data, p...)
	b.total += int64(len(p))
	b.lastByte = p[len(p)-1]
	if len(b.data) > 2*maxTermBuffer {
		kept := copy(b.data, b.data[len(b.data)-maxTermBuffer:])
		b.data = b.data[:kept]
	}
	return start
}

// snapshot returns a copy of the buffered output, starting at a line boundary
// when older output was dropped so it never begins inside an escape sequence.
func (b *termBuffer) snapshot() []byte {
	data := b.data
	if len(data) > maxTermBuffer {
		data = data[len(data)-maxTermBuffer:]
	}
	if int64(len(data)) < b.total {
		if i := bytes.IndexByte(data, '\n'); i >= 0 && i < 16*1024 {
			data = data[i+1:]
		}
	}
	return append([]byte(nil), data...)
}

var (
	// cursorMoveRe matches the cursor movements a terminal uses instead of a
	// line feed (H, f, E, B, d); line-based matching treats them as line breaks.
	cursorMoveRe = regexp.MustCompile(`\x1b\[[0-9;]*[HfEBd]`)
	ansiRe       = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[=>78]`)
)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// lineTap turns raw terminal output into plain-text lines (for ready-URL
// detection) without touching the output itself.
type lineTap struct {
	pending string
}

// feed adds a chunk and returns the lines it completed.
func (t *lineTap) feed(chunk []byte) []string {
	t.pending += string(chunk)
	cut := strings.LastIndexAny(t.pending, "\r\n")
	if cut < 0 {
		if len(t.pending) > maxTermLine {
			t.pending = t.pending[len(t.pending)-maxTermLine:]
		}
		return nil
	}
	complete, rest := t.pending[:cut+1], t.pending[cut+1:]
	t.pending = rest
	return plainLines(complete)
}

// flush returns the unfinished last line.
func (t *lineTap) flush() []string {
	rest := t.pending
	t.pending = ""
	return plainLines(rest)
}

func plainLines(raw string) []string {
	text := stripANSI(cursorMoveRe.ReplaceAllString(raw, "\n"))
	var lines []string
	for _, line := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' }) {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// dimLine renders text as a dimmed terminal line, used for the app's own
// messages inside a command's output.
func dimLine(text string) []byte {
	return []byte("\x1b[2m" + strings.TrimRight(text, "\r\n") + "\x1b[0m\r\n")
}

// redLine renders text as a red terminal line.
func redLine(text string) []byte {
	return []byte("\x1b[31m" + strings.TrimRight(text, "\r\n") + "\x1b[0m\r\n")
}
