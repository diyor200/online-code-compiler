package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/diyor200/code-compiler/pkg/linewriter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
)

var tasks = sync.Map{}

type UseCase struct {
	client *client.Client
	config domain.ExecutorConfig
}

func New(client *client.Client, config domain.ExecutorConfig) *UseCase {
	return &UseCase{
		client: client,
		config: config,
	}
}

func (u *UseCase) CreateTask(ctx context.Context, data domain.ExecuteRequest) (string, error) {
	taskID := uuid.NewString()
	tasks.Store(taskID, data)

	return taskID, nil
}

func (u *UseCase) Execute(ctx context.Context, taskID string, writer domain.StreamWriter) {
	startTime := time.Now()

	taskData, ok := tasks.Load(taskID)
	if !ok {
		writer.Error(fmt.Errorf("task not found"))
		return
	}

	data, ok := taskData.(domain.ExecuteRequest)
	if !ok {
		logger.Error("failed to convert task to domain")
		writer.Error(errors.New("internal server error"))
		return
	}

	// remove task
	tasks.Delete(taskID)

	langConfig, ok := domain.LanguageConfigs[data.Language]
	if !ok {
		writer.Error(fmt.Errorf("language %s not supported", data.Language))
		return
	}

	// create temp dir for the code
	tempDir, err := os.MkdirTemp("", "code-exec-*")
	if err != nil {
		logger.Error("failed to create temp dir:", err)
		writer.Error(fmt.Errorf("failed to create temp dir: %w", err))
		return
	}
	defer os.RemoveAll(tempDir)

	// write code to file
	codePath := filepath.Join(tempDir, langConfig.FileName)
	if err = os.WriteFile(codePath, []byte(data.Code), 0644); err != nil {
		logger.Error("failed to write code:", err)
		writer.Error(fmt.Errorf("failed to write code: %w", err))
		return
	}

	// pull image if needed
	if err = u.pullImageIfNotExist(ctx, langConfig.Image); err != nil {
		logger.Error("failed to pull image:", err)
		writer.Error(fmt.Errorf("failed to pull image: %w", err))
		return
	}

	cmd := langConfig.RunCmd
	// compile if needed
	if langConfig.NeedsCompile {
		cmd = []string{
			"sh", "-c",
			fmt.Sprintf("%s && %s",
				joinCmd(langConfig.CompileCmd),
				joinCmd(langConfig.RunCmd),
			),
		}
	}

	// execute code
	u.runContainer(ctx, langConfig.Image, cmd, tempDir, data.Stdin, writer)
	executionTime := time.Since(startTime).Seconds()
	fmt.Println("execution time: ", executionTime)
}

func (u *UseCase) pullImageIfNotExist(ctx context.Context, image string) error {
	_, _, err := u.client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}

	// pull image
	out, err := u.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		logger.Error("failed to pull image:", err)
		return err
	}
	defer out.Close()

	io.Copy(os.Stdout, out)
	return nil
}

// runContainer container configuration with resource limits
func (u *UseCase) runContainer(ctx context.Context, image string, cmd []string,
	volumePath, stdin string, writer domain.StreamWriter) {
	containerConfig := &container.Config{
		Image:           image,
		WorkingDir:      "/app",
		Cmd:             cmd,
		Tty:             false,
		AttachStdin:     stdin != "",
		AttachStdout:    true,
		AttachStderr:    true,
		OpenStdin:       stdin != "",
		StdinOnce:       true,
		NetworkDisabled: u.config.NetworkDisabled,
	}

	hostConfig := &container.HostConfig{
		AutoRemove:     false,
		ReadonlyRootfs: false,
		SecurityOpt:    []string{"no-new-privileges"},
		Binds:          []string{fmt.Sprintf("%s:/app", volumePath)},
		Resources: container.Resources{
			Memory:    u.config.MemoryLimit,
			CPUQuota:  u.config.CPUQuota,
			CPUPeriod: 100000,
			PidsLimit: func() *int64 { v := int64(64); return &v }(), // Limit processes,
		},
	}

	// create container
	resp, err := u.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		logger.Error("failed to create container:", err)
		writer.Error(fmt.Errorf("failed to create container: %w", err))
		return
	}
	defer u.client.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	// Attach to container for stdin/stdout/stderr
	attachResp, err := u.client.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  stdin != "",
	})
	if err != nil {
		logger.Error("failed to attach container:", err)
		writer.Error(fmt.Errorf("failed to attach container: %w", err))
		return
	}
	defer attachResp.Close()

	// Write stdin if provided
	if stdin != "" {
		go func() {
			attachResp.Conn.Write([]byte(stdin))
			attachResp.CloseWrite()
		}()
	}

	// Start container
	if err = u.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("failed to start container:", err)
		writer.Error(fmt.Errorf("failed to start container: %w", err))
		return
	}

	done := make(chan struct{})

	// stream output
	go func() {
		defer close(done)
		stdout := linewriter.NewLinewriter(writer.Stdout)
		stderr := linewriter.NewLinewriter(writer.Stderr)

		_, err := stdcopy.StdCopy(stdout, stderr, attachResp.Reader)
		if err != nil && err != io.EOF {
			logger.Errorf("stream error: %v", err)
			writer.Error(fmt.Errorf("stream error: %w", err))
		}

		stdout.Flush()
		stderr.Flush()
	}()

	// wait to container finish
	statusCh, errCh := u.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	var status container.WaitResponse

	select {
	case err = <-errCh:
		if err != nil {
			logger.Error("failed to wait for container:", err)
			writer.Error(fmt.Errorf("failed to wait for container: %w", err))
		}
	case status = <-statusCh:
	case <-ctx.Done():
		killCtx := context.Background() // don't use cancelled ctx
		_ = u.client.ContainerKill(killCtx, resp.ID, "SIGKILL")
		<-done
		writer.Error(fmt.Errorf("execution timeout"))
		return
	}
	<-done

	writer.Done(int(status.StatusCode))
}

func joinCmd(cmd []string) string {
	return strings.Join(cmd, " ")
}
