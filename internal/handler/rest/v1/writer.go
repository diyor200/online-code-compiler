package v1

import (
	"fmt"
	"net/http"
)

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (w *sseWriter) send(event, data string) {
	fmt.Fprintf(w.w, "event: %s\ndata: %s\n\n", event, data)
	w.flusher.Flush()
}

func (w *sseWriter) Stdout(s string) {
	w.send("stdout", s)
}

func (w *sseWriter) Stderr(s string) {
	w.send("stderr", s)
}

func (w *sseWriter) Done(code int) {
	w.send("done", fmt.Sprintf("%d", code))
}

func (w *sseWriter) Error(err error) {
	w.send("error", err.Error())
}
