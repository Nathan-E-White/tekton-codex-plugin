package safety_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
)

func TestPathPolicyAllowsFilesInsideConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "pipeline.yaml")
	if err := os.WriteFile(manifest, []byte("kind: Pipeline\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	policy, err := safety.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	got, err := policy.Resolve(manifest)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want, err := filepath.EvalSymlinks(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Resolve() = %q, want canonical path %q", got, want)
	}
}

func TestPathPolicyRejectsTraversalAndEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.yaml")
	if err := os.WriteFile(outsideFile, []byte("kind: Secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.yaml")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Fatal(err)
	}
	policy, err := safety.NewPathPolicy([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{filepath.Join(root, "..", filepath.Base(outside), "secret.yaml"), link} {
		if _, err := policy.Resolve(candidate); err == nil {
			t.Fatalf("Resolve(%q) succeeded, want rejection", candidate)
		}
	}
}
