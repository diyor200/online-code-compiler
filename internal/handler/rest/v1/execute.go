package v1

import (
	"net/http"

	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
)

// createTask godoc
// @Summary Create Task
// @Description Create Task
// @Param request body scheme.ExecuteRequest true "execute request"
// @Produce json
// @Success 201
// @Failure 500
// @Router /api/v1/task [POST]
func (h *Handler) createTask(c *gin.Context) {
	var (
		req scheme.ExecuteRequest
		err error
	)
	if err = c.ShouldBindJSON(&req); err != nil {
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

	taskID, err := h.executor.CreateTask(c.Request.Context(), req.ToModel())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, scheme.ExecuteResponseV2{TaskID: taskID})

}

// getTaskResult godoc
// @Summary Get task result
// @Description Get task result
// @Param task_id path string true "task_id"
// @Produce json
// @Success 200
// @Failure 400
// @Router /api/v1/task/:id [GET]
func (h *Handler) getTaskResult(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "task_id is required"})
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

	h.executor.Execute(c.Request.Context(), taskID, writer)
}
