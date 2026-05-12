package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/diyor200/code-compiler/internal/config"
	v1 "github.com/diyor200/code-compiler/internal/handler/rest/v1"
	"github.com/diyor200/code-compiler/pkg/metrics"
	"github.com/diyor200/code-compiler/pkg/rate_limiter"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func Run() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	//  register metrics
	metrics.RegisterMetrics()

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
	// set cors middleware

	// set rate limiter. 10 requests per minute
	rateLimiter := rate_limiter.NewRateLimiter(rate.Limit(10.0/60.0), 5)
	rateLimiter.CleanUp()

	handler := v1.NewHandler(server, cfg, usecases.executor, rateLimiter)

	if err = handler.Engine.Run(fmt.Sprintf("%s:%s", cfg.HttpHost, cfg.HttpPort)); err != nil {
		panic(err)
	}
}
