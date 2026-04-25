package linewriter

import (
	"bytes"
	"strings"
)

type linewriter struct {
	buf bytes.Buffer
	fn  func(line string)
}

func NewLinewriter(fn func(s string)) *linewriter {
	return &linewriter{
		fn: fn,
	}
}

func (w *linewriter) Write(p []byte) (int, error) {
	w.buf.Write(p)

	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			break
		}

		w.fn(strings.TrimRight(line, "\n"))
	}

	return len(p), nil
}

func (w *linewriter) Flush() {
	if w.buf.Len() > 0 {
		w.fn(w.buf.String())
		w.buf.Reset()
	}
}
