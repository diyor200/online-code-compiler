package v1

import (
	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) executeCode(c *gin.Context) {
	var req scheme.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	res := h.executor.Execute(c.Request.Context(), req.ToModel())
	if res.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": res.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
