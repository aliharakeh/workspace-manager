package services

import (
	"bytes"
	"strings"
	"testing"
)

func TestTermBufferKeepsRecentOutputAndCountsAll(t *testing.T) {
	var b termBuffer
	if off := b.add([]byte("one\n")); off != 0 {
		t.Fatalf("first offset = %d", off)
	}
	if off := b.add([]byte("two\n")); off != 4 {
		t.Fatalf("second offset = %d", off)
	}
	if got := string(b.snapshot()); got != "one\ntwo\n" || b.total != 8 || b.lastByte != '\n' {
		t.Fatalf("snapshot=%q total=%d last=%q", got, b.total, b.lastByte)
	}

	// Past the cap the oldest output goes, but the total keeps counting and the
	// snapshot restarts on a line boundary.
	var big termBuffer
	line := append(bytes.Repeat([]byte("x"), 99), '\n')
	for i := 0; i < (3*maxTermBuffer)/len(line); i++ {
		big.add(line)
	}
	snap := big.snapshot()
	if len(snap) > maxTermBuffer || len(snap) < maxTermBuffer-len(line) {
		t.Fatalf("snapshot size = %d", len(snap))
	}
	if !bytes.HasPrefix(snap, line) || !bytes.HasSuffix(snap, line) {
		t.Fatal("snapshot must hold whole lines")
	}
	if big.total < int64(3*maxTermBuffer)-int64(len(line)) {
		t.Fatalf("total = %d", big.total)
	}
}

func TestLineTap(t *testing.T) {
	var tap lineTap
	// A line split over chunks, colors, and a cursor move instead of a newline.
	if got := tap.feed([]byte("\x1b[32m  Local:")); got != nil {
		t.Fatalf("unfinished line returned %q", got)
	}
	got := tap.feed([]byte("   http://localhost:5173/\x1b[0m\r\nnext\x1b[2;1Hthird\r\n\r\n"))
	want := []string{"  Local:   http://localhost:5173/", "next", "third"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
	tap.feed([]byte("no newline yet"))
	if rest := tap.flush(); len(rest) != 1 || rest[0] != "no newline yet" {
		t.Fatalf("flush = %q", rest)
	}
}
