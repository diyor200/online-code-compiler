package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/diyor200/code-compiler/pkg/linewriter"
	"github.com/diyor200/code-compiler/pkg/metrics"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
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

	if err := u.pullImageIfNotExist(ctx, langConfig.Image); err != nil {
		writer.Error(fmt.Errorf("failed to pull image: %w", err))
		return
	}

	// inc active tasks
	metrics.ActiveExecutions.Inc()
	defer metrics.ActiveExecutions.Dec()

	switch {
	case !langConfig.NeedsCompile:
		u.runInterpreted(ctx, langConfig, data, writer, data.Language)
	case langConfig.StdinCompile:
		u.runCompiledStdin(ctx, langConfig, data, writer) // C, C++ — next step
	default:
		u.runCompiledTmpfs(ctx, langConfig, data, writer) // Go, Java — after that
	}

	executionTime := time.Since(startTime).Seconds()
	metrics.ExecutionDuration.WithLabelValues(data.Language).Observe(executionTime)

	logger.Infof("execution time: %.2fs", executionTime)
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

func (u *UseCase) runInterpreted(ctx context.Context, config domain.LanguageConfig,
	req domain.ExecuteRequest, writer domain.StreamWriter, language string) {
	// copy slice to avoid data race on shared global config
	cmd := make([]string, len(config.RunCmd))
	copy(cmd, config.RunCmd)
	cmd[len(cmd)-1] = req.Code

	u.runContainer(ctx, config.Image, cmd, req.Stdin, writer, language)
}

func (u *UseCase) runCompiledStdin(ctx context.Context, config domain.LanguageConfig, req domain.ExecuteRequest, writer domain.StreamWriter) {
	// shared volume between compile and run containers
	volumeName := fmt.Sprintf("compiler-%s", uuid.New().String())

	// create volume
	_, err := u.client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
	if err != nil {
		writer.Error(fmt.Errorf("failed to create volume: %w", err))
		metrics.TotalExecutions.WithLabelValues(req.Language, "failure").Inc()
		return
	}
	defer u.client.VolumeRemove(context.Background(), volumeName, true)

	// step 1: compile
	compileSuccess := u.compileWithStdin(ctx, config, req.Code, volumeName, writer, req.Language)
	if !compileSuccess {
		metrics.TotalExecutions.WithLabelValues(req.Language, "failure").Inc()
		return
	}

	// step 2: run
	u.runWithVolume(ctx, config, req.Stdin, volumeName, writer, req.Language)
}

func (u *UseCase) compileWithStdin(ctx context.Context, config domain.LanguageConfig, code string,
	volumeName string, writer domain.StreamWriter, language string) bool {
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.CompileCmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
		WorkingDir:   "/app",
	}

	hostConfig := &container.HostConfig{
		AutoRemove: false,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/app",
			},
		},
		Resources: container.Resources{
			Memory:    u.config.MemoryLimit,
			CPUQuota:  u.config.CPUQuota,
			CPUPeriod: 100000,
			PidsLimit: func() *int64 { v := int64(64); return &v }(),
		},
		SecurityOpt: []string{"no-new-privileges"},
	}

	resp, err := u.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		writer.Error(fmt.Errorf("failed to create compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "failure").Inc()
		return false
	}
	defer u.client.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	attachResp, err := u.client.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		writer.Error(fmt.Errorf("failed to attach compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "compile").Inc()
		return false
	}
	defer attachResp.Close()

	// write code to stdin synchronously before start
	attachResp.Conn.Write([]byte(code))
	attachResp.CloseWrite()

	if err = u.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		writer.Error(fmt.Errorf("failed to start compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "compile").Inc()
		return false
	}

	// collect compile errors — pipe stderr to writer.Stderr
	done := make(chan struct{})
	go func() {
		defer close(done)
		stdout := linewriter.NewLinewriter(writer.Stderr) // compile stdout → stderr stream
		stderr := linewriter.NewLinewriter(writer.Stderr)
		stdcopy.StdCopy(stdout, stderr, attachResp.Reader)
		stdout.Flush()
		stderr.Flush()
	}()

	statusCh, errCh := u.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	var status container.WaitResponse
	select {
	case err = <-errCh:
		if err != nil {
			writer.Error(fmt.Errorf("compile wait error: %w", err))
			<-done
			return false
		}
	case status = <-statusCh:
	case <-ctx.Done():
		u.client.ContainerKill(context.Background(), resp.ID, "SIGKILL")
		<-done
		metrics.TotalExecutions.WithLabelValues(language, "timeout").Inc()
		writer.Error(fmt.Errorf("compile timeout"))
		return false
	}

	<-done

	// non-zero exit = compile error
	if status.StatusCode != 0 {
		writer.Done(int(status.StatusCode))
		return false
	}

	return true
}

func (u *UseCase) runWithVolume(ctx context.Context, config domain.LanguageConfig,
	stdin string, volumeName string, writer domain.StreamWriter, language string) {
	containerConfig := &container.Config{
		Image:           config.Image,
		Cmd:             config.RunCmd,
		AttachStdin:     stdin != "",
		AttachStdout:    true,
		AttachStderr:    true,
		OpenStdin:       stdin != "",
		StdinOnce:       true,
		WorkingDir:      "/app",
		NetworkDisabled: u.config.NetworkDisabled,
	}

	hostConfig := &container.HostConfig{
		AutoRemove: false,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/app",
			},
		},
		SecurityOpt: []string{"no-new-privileges"},
		Resources: container.Resources{
			Memory:    u.config.MemoryLimit,
			CPUQuota:  u.config.CPUQuota,
			CPUPeriod: 100000,
			PidsLimit: func() *int64 { v := int64(64); return &v }(),
		},
	}

	resp, err := u.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		writer.Error(fmt.Errorf("failed to create run container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "run").Inc()
		return
	}
	defer u.client.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	attachResp, err := u.client.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  stdin != "",
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		writer.Error(fmt.Errorf("failed to attach run container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "run").Inc()
		return
	}
	defer attachResp.Close()

	// write user stdin synchronously before start
	if stdin != "" {
		attachResp.Conn.Write([]byte(stdin))
		attachResp.CloseWrite()
	}

	if err = u.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		metrics.ContainerFailures.WithLabelValues(language, "run").Inc()
		writer.Error(fmt.Errorf("failed to start run container: %w", err))
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		stdout := linewriter.NewLinewriter(writer.Stdout)
		stderr := linewriter.NewLinewriter(writer.Stderr)
		_, err := stdcopy.StdCopy(stdout, stderr, attachResp.Reader)
		if err != nil && err != io.EOF {
			writer.Error(fmt.Errorf("stream error: %w", err))
		}
		stdout.Flush()
		stderr.Flush()
	}()

	statusCh, errCh := u.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	var status container.WaitResponse
	select {
	case err = <-errCh:
		if err != nil {
			writer.Error(fmt.Errorf("run wait error: %w", err))
			<-done
			return
		}
	case status = <-statusCh:
	case <-ctx.Done():
		u.client.ContainerKill(context.Background(), resp.ID, "SIGKILL")
		<-done
		metrics.TotalExecutions.WithLabelValues(language, "timeout").Inc()
		writer.Error(fmt.Errorf("execution timeout"))
		return
	}

	<-done
	writer.Done(int(status.StatusCode))
	metrics.TotalExecutions.WithLabelValues(language, "success").Inc()
}

func (u *UseCase) runCompiledTmpfs(ctx context.Context, config domain.LanguageConfig, req domain.ExecuteRequest, writer domain.StreamWriter) {
	// shared volume for the binary between compile and run containers
	volumeName := fmt.Sprintf("compiler-%s", uuid.New().String())

	_, err := u.client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
	if err != nil {
		writer.Error(fmt.Errorf("failed to create volume: %w", err))
		metrics.TotalExecutions.WithLabelValues(req.Language, "failure").Inc()
		return
	}
	defer u.client.VolumeRemove(context.Background(), volumeName, true)

	// step 1: compile (write code to tmpfs, compile to volume)
	compileSuccess := u.compileWithTmpfs(ctx, config, req.Code, volumeName, writer, req.Language)
	if !compileSuccess {
		metrics.TotalExecutions.WithLabelValues(req.Language, "failure").Inc()
		return
	}

	// step 2: run binary from volume
	u.runWithVolume(ctx, config, req.Stdin, volumeName, writer, req.Language)
}

func (u *UseCase) compileWithTmpfs(ctx context.Context, config domain.LanguageConfig, code string,
	volumeName string, writer domain.StreamWriter, language string) bool {
	// inject code as env var, entrypoint script writes it to file
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.CompileCmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/app",
		Env:          []string{fmt.Sprintf("CODE=%s", code)},
		// entrypoint writes $CODE to the source file before compiling
		Entrypoint: []string{"/bin/sh", "-c", fmt.Sprintf("echo \"$CODE\" > /app/%s && %s",
			config.SourceFile,
			joinCmd(config.CompileCmd),
		)},
	}

	hostConfig := &container.HostConfig{
		AutoRemove: false,
		Mounts: []mount.Mount{
			{
				// tmpfs for source file — in-memory, no host path binding
				Type:   mount.TypeTmpfs,
				Target: "/src",
				TmpfsOptions: &mount.TmpfsOptions{
					SizeBytes: 10 * 1024 * 1024, // 10MB
					Mode:      0700,
				},
			},
			{
				// named volume for compiled binary
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/app",
			},
		},
		Resources: container.Resources{
			Memory:    u.config.MemoryLimit,
			CPUQuota:  u.config.CPUQuota,
			CPUPeriod: 100000,
			PidsLimit: func() *int64 { v := int64(64); return &v }(),
		},
		SecurityOpt: []string{"no-new-privileges"},
	}

	resp, err := u.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		writer.Error(fmt.Errorf("failed to create compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "compile").Inc()
		return false
	}
	defer u.client.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	attachResp, err := u.client.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		writer.Error(fmt.Errorf("failed to attach compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "compile").Inc()
		return false
	}
	defer attachResp.Close()

	if err = u.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		writer.Error(fmt.Errorf("failed to start compile container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "compile").Inc()
		return false
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		stdout := linewriter.NewLinewriter(writer.Stderr)
		stderr := linewriter.NewLinewriter(writer.Stderr)
		stdcopy.StdCopy(stdout, stderr, attachResp.Reader)
		stdout.Flush()
		stderr.Flush()
	}()

	statusCh, errCh := u.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	var status container.WaitResponse
	select {
	case err = <-errCh:
		if err != nil {
			writer.Error(fmt.Errorf("compile wait error: %w", err))
			<-done
			return false
		}
	case status = <-statusCh:
	case <-ctx.Done():
		u.client.ContainerKill(context.Background(), resp.ID, "SIGKILL")
		<-done
		metrics.TotalExecutions.WithLabelValues(language, "timeout").Inc()
		writer.Error(fmt.Errorf("compile timeout"))
		return false
	}

	<-done

	if status.StatusCode != 0 {
		writer.Done(int(status.StatusCode))
		return false
	}

	return true
}

// runContainer container configuration with resource limits
func (u *UseCase) runContainer(ctx context.Context, image string, cmd []string,
	stdin string, writer domain.StreamWriter, language string) error {
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
		return err
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
		return err
	}
	defer attachResp.Close()

	// Write stdin if provided
	if stdin != "" {
		attachResp.Conn.Write([]byte(stdin))
		attachResp.CloseWrite()
	}

	// Start container
	if err = u.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("failed to start container:", err)
		writer.Error(fmt.Errorf("failed to start container: %w", err))
		metrics.ContainerFailures.WithLabelValues(language, "run").Inc()
		return err
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
			<-done
			return err
		}
	case status = <-statusCh:
	case <-ctx.Done():
		killCtx := context.Background() // don't use cancelled ctx
		_ = u.client.ContainerKill(killCtx, resp.ID, "SIGKILL")
		<-done
		writer.Error(fmt.Errorf("execution timeout"))
		metrics.TotalExecutions.WithLabelValues(language, "timeout").Inc()
		return errors.New("execution timeout")
	}
	<-done

	writer.Done(int(status.StatusCode))
	metrics.TotalExecutions.WithLabelValues(language, "success").Inc()
	return nil
}

func joinCmd(cmd []string) string {
	return strings.Join(cmd, " ")
}
