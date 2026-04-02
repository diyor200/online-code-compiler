package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type UseCase struct{}

func (u *UseCase) Execute(ctx context.Context, data domain.ExecCode) (domain.ExecResult, error) {
	// 1. Create isolated temp directory
	id := uuid.NewString()
	tempDir := filepath.Join(os.TempDir(), "exec-"+id)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		logger.Error("failed to make temp dir:", err.Error())
		return domain.ExecResult{}, err
	}
	defer os.RemoveAll(tempDir)

	// 2. Get file name, docker image, and run command
	fileName, img, runCmd, err := u.prepare(data.Lang)
	if err != nil {
		logger.Error("failed to prepare code:", err.Error())
		return domain.ExecResult{}, err
	}

	// 3. Write code to temp directory
	filePath := filepath.Join(tempDir, fileName)
	if err = os.WriteFile(filePath, []byte(data.Code), 0644); err != nil {
		logger.Error("failed to write file:", err.Error())
		return domain.ExecResult{}, err
	}

	// 4. Get absolute path for Docker mount (important on Mac)
	absDir, err := filepath.Abs(tempDir)
	if err != nil {
		logger.Error("failed to get absolute path:", err.Error())
		return domain.ExecResult{}, err
	}

	// 5. Build Docker command
	args := []string{
		"run",
		"--rm",

		// resource limits
		"--cpus=1.0",
		"--memory=512m",
		"--pids-limit=64",
		"--network=none",
		"--read-only",

		// tmpfs for Go build & temp files
		"--tmpfs", "/tmp:exec",

		// non-root
		"--user", "1000:1000",

		// set Go cache
		"-e", "GOCACHE=/tmp/go-build",

		// mount code
		"-v", fmt.Sprintf("%s:/app", absDir),
		"-w", "/app",

		img,
	}
	args = append(args, runCmd...)

	// 6. Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return domain.ExecResult{}, errors.New("execution timeout")
	}

	if err != nil {
		logger.Error("failed to run docker command:", err.Error())
		if stderr.Len() > 0 {
			return domain.ExecResult{}, errors.New(stderr.String())
		}
		return domain.ExecResult{}, err
	}

	// 7. Return stdout as result
	return domain.ExecResult{
		Result: stdout.String(),
	}, nil
}

// prepare returns file name, docker image, and execution command
func (u *UseCase) prepare(language string) (filename, image string, cmd []string, err error) {
	switch language {

	case "go":
		return "main.go", "golang:1.22", []string{"go", "run", "main.go"}, nil

	case "python":
		return "main.py", "python:3.11", []string{"python", "main.py"}, nil

	case "javascript":
		return "main.js", "node:20", []string{"node", "main.js"}, nil

	default:
		return "", "", nil, fmt.Errorf("unsupported language: %s", language)
	}
}

func New() *UseCase {
	return &UseCase{}
}
