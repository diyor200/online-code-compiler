package v1

import (
	"context"

	"github.com/diyor200/code-compiler/internal/config"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	engine   *gin.Engine
	conf     config.Config
	executor Executor
}

type Executor interface {
	Execute(ctx context.Context)
}
