package bootstrap

import (
	"context"
	"fmt"
	"github.com/diyor200/code-compiler/internal/config"
	v1 "github.com/diyor200/code-compiler/internal/handler/rest/v1"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"log"
)

func Run() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer dockerClient.Close()

	// check docker running or not
	pingRes, err := dockerClient.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	log.Printf("[DOCKER]\n\tVersion: %s\n\tAPI Version: %s\n\tOSType: %s",
		pingRes.BuilderVersion, pingRes.APIVersion, pingRes.OSType)

	usecases := NewUseCaseBuilder(dockerClient)

	server := gin.Default()
	handler := v1.NewHandler(server, cfg, usecases.executor)

	if err = handler.Engine.Run(fmt.Sprintf("%s:%s", cfg.HttpHost, cfg.HttpPort)); err != nil {
		panic(err)
	}
}
