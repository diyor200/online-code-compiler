package v1

import (
	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) executeCode(c *gin.Context) {
	var req scheme.CodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.executor.Execute(c.Request.Context(), req.ToModel())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
