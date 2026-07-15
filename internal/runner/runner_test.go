package runner_test

import (
	"context"
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/runner"
)

func TestRunnerRejectsCommandsOutsideAllowlist(t *testing.T) {
	if _, err := (runner.Runner{}).Run(context.Background(), "sh", []string{"-c", "true"}, nil); err == nil {
		t.Fatal("Run() succeeded for non-allowlisted shell")
	}
}
