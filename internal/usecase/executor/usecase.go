package executor

import (
	"context"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"os"
	"path/filepath"
	"time"
)

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

func (u *UseCase) Execute(ctx context.Context, data domain.ExecuteRequest) domain.ExecutionResult {
	startTime := time.Now()

	langConfig, ok := domain.LanguageConfigs[data.Language]
	if !ok {
		return domain.ExecutionResult{
			Error: fmt.Errorf("language %s not supported", data.Language),
		}
	}

	// create temp dir for the code
	tempDir, err := os.MkdirTemp("", "code-exec-*")
	if err != nil {
		logger.Error("failed to create temp dir:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to create temp dir: %w", err)}
	}
	defer os.RemoveAll(tempDir)

	// write code to file
	codePath := filepath.Join(tempDir, langConfig.FileName)
	if err = os.WriteFile(codePath, []byte(data.Code), 0644); err != nil {
		logger.Error("failed to write code:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to write code: %w", err)}
	}

	// pull image if needed
	if err = u.pullImageIfNotExist(ctx, langConfig.Image); err != nil {
		logger.Error("failed to pull image:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to pull image: %w", err)}
	}

	// compile if needed
	if langConfig.NeedsCompile {
		compileResult := u.runContainer(ctx, langConfig.Image, langConfig.CompileCmd, tempDir, "")
		if compileResult.Error != nil {
			return compileResult
		}
		if compileResult.ExitCode != 0 {
			return domain.ExecutionResult{
				Stderr:   compileResult.Stderr,
				Error:    fmt.Errorf("compilation failed"),
				ExitCode: compileResult.ExitCode,
			}
		}
	}

	// execute code
	result := u.runContainer(ctx, langConfig.Image, langConfig.RunCmd, tempDir, data.Stdin)
	result.ExecutionTime = time.Since(startTime).Seconds()

	return result
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
func (u *UseCase) runContainer(ctx context.Context, image string, cmd []string, volumePath, stdin string) domain.ExecutionResult {
	containerConfig := &container.Config{
		Image:           image,
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
		AutoRemove:     true,
		ReadonlyRootfs: false,
		SecurityOpt:    []string{"no-new-privileges"},
		Binds:          []string{fmt.Sprintf("%s:/app", volumePath)},
		Resources: container.Resources{
			Memory:    u.config.MemoryLimit,
			CPUQuota:  u.config.CPUQuota,
			CPUPeriod: 100000,
			PidsLimit: func() *int64 { v := int64(50); return &v }(), // Limit processes,
		},
	}

	// create container
	resp, err := u.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		logger.Error("failed to create container:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to create container: %w", err)}
	}

	// Attach to container for stdin/stdout/stderr
	attachResp, err := u.client.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  stdin != "",
	})
	if err != nil {
		logger.Error("failed to attach container:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to attach container: %w", err)}
	}
	defer attachResp.Close()

	//Write stdin if provided
	if stdin != "" {
		go func() {
			attachResp.Conn.Write([]byte(stdin))
			attachResp.CloseWrite()
		}()
	}

	// Start container
	if err = u.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("failed to start container:", err)
		return domain.ExecutionResult{Error: fmt.Errorf("failed to start container: %w", err)}
	}

	// read output
	stdout, stder := u.readOutput(attachResp.Conn)

	// wait to container finish
	statusCh, errCh := u.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			logger.Error("failed to wait for container:", err)
			return domain.ExecutionResult{Error: fmt.Errorf("failed to wait for container: %w", err)}
		}
	case status := <-statusCh:
		return domain.ExecutionResult{
			Stdout:   stdout,
			Stderr:   stder,
			ExitCode: int(status.StatusCode),
		}
	case <-ctx.Done():
		// timeout force kill containers
		u.client.ContainerKill(ctx, resp.ID, "SIGKILL")
		return domain.ExecutionResult{
			Stderr: "execution timeout exceeded",
			Error:  fmt.Errorf("execution timeout exceeded"),
		}
	}

	return domain.ExecutionResult{Error: fmt.Errorf("unexpected error")}
}

func (u *UseCase) readOutput(reader io.Reader) (string, string) {
	var stdout, stderr string
	buf := make([]byte, 8192)

	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}

		if n < 8 {
			continue
		}

		// docker multiplexes stdout/stderr
		// first 8 bytes: header (stream type + size)
		streamType := buf[0]
		size := int(buf[4])<<24 | int(buf[5])<<16 | int(buf[6])<<8 | int(buf[7])

		if size > len(buf)-8 {
			size = len(buf) - 8
		}

		content := string(buf[8 : size+8])

		if streamType == 1 { // stdout
			stdout += content
		} else if streamType == 2 { // stderr
			stderr += content
		}
	}

	return stdout, stderr
}
