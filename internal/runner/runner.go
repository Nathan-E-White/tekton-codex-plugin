package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
)

var allowed = map[string]bool{"kubectl": true, "tkn": true, "tkn-results": true, "cosign": true}

type Result struct {
	Command   string        `json:"command"`
	ExitCode  int           `json:"exit_code"`
	Output    string        `json:"output"`
	Truncated bool          `json:"truncated"`
	Duration  time.Duration `json:"duration"`
}

type Runner struct {
	Timeout  time.Duration
	MaxBytes int
}

func (r Runner) Run(ctx context.Context, command string, args []string, stdin []byte) (Result, error) {
	if !allowed[command] {
		return Result{}, fmt.Errorf("command %q is not allowlisted", command)
	}
	if r.Timeout <= 0 {
		r.Timeout = 30 * time.Second
	}
	if r.MaxBytes <= 0 {
		r.MaxBytes = 256 * 1024
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	buf := &boundedBuffer{limit: r.MaxBytes}
	cmd.Stdout, cmd.Stderr = buf, buf
	started := time.Now()
	err := cmd.Run()
	result := Result{Command: command, Output: safety.Redact(buf.String()), Truncated: buf.truncated, Duration: time.Since(started)}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if ctx.Err() != nil {
		return result, fmt.Errorf("%s timed out: %w", command, ctx.Err())
	}
	return result, fmt.Errorf("%s failed with exit code %d", command, result.ExitCode)
}

type boundedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, err := io.Copy(&b.buf, bytes.NewReader(p))
	return original, err
}

func (b *boundedBuffer) String() string { return b.buf.String() }
