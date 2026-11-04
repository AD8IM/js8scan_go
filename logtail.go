package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"
)

type LogLine struct {
	Source string // "ALL" or "DIRECTED"
	Text   string
}

// TailFile follows a file like `tail -F`: starts at the current end,
// and reopens it if it is truncated or replaced.
func TailFile(ctx context.Context, path, source string, out chan<- LogLine) {
	var f *os.File
	var r *bufio.Reader
	var partial string
	var lastSize int64

	open := func(seekEnd bool) bool {
		nf, err := os.Open(path)
		if err != nil {
			return false
		}
		if seekEnd {
			lastSize, _ = nf.Seek(0, io.SeekEnd)
		} else {
			lastSize = 0
		}
		f, r = nf, bufio.NewReader(nf)
		return true
	}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	first := true
	for {
		select {
		case <-ctx.Done():
			if f != nil {
				f.Close()
			}
			return
		case <-t.C:
		}
		if f == nil {
			if !open(first) {
				continue
			}
			first = false
		}
		if st, err := os.Stat(path); err != nil || st.Size() < lastSize {
			f.Close()
			f = nil
			first = false
			continue
		}
		for {
			s, err := r.ReadString('\n')
			lastSize += int64(len(s))
			if err != nil { // no complete line yet
				partial += s
				break
			}
			line := partial + s
			partial = ""
			if len(line) > 0 {
				select {
				case out <- LogLine{source, trimNL(line)}:
				default:
				}
			}
		}
	}
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
