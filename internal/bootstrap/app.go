package bootstrap

import (
	"fmt"
	"github.com/diyor200/code-compiler/internal/config"
	v1 "github.com/diyor200/code-compiler/internal/handler/rest/v1"
	"github.com/gin-gonic/gin"
)

func Run() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	usecases := NewUseCaseBuilder()

	server := gin.Default()
	handler := v1.NewHandler(server, cfg, usecases.executor)

	if err = handler.Engine.Run(fmt.Sprintf("%s:%s", cfg.HttpHost, cfg.HttpPort)); err != nil {
		panic(err)
	}
}
