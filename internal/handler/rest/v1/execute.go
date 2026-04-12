package v1

import (
	"net/http"

	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
)

func (h *Handler) executeCode(c *gin.Context) {
	var req scheme.ExecuteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Code) > 50000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Code exceeds maximum length of 50KB",
		})
		return
	}

	if len(req.Stdin) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Input exceeds maximum length of 10KB",
		})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected error, try again later"})
		return
	}

	// sse headers
	// ===== SSE HEADERS =====
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	writer := &sseWriter{
		w:       c.Writer,
		flusher: flusher,
	}

	h.executor.Execute(c.Request.Context(), req.ToModel(), writer)
}
