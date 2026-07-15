package cluster_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/cluster"
)

func TestNewRequiresExplicitContextAndNamespace(t *testing.T) {
	kubeconfig := filepath.Join(t.TempDir(), "config")
	data := []byte("apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\n")
	if err := os.WriteFile(kubeconfig, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", kubeconfig)
	if _, err := cluster.New("", "ci"); err == nil || !strings.Contains(err.Error(), "explicit kubeconfig context") {
		t.Fatalf("New() error = %v, want explicit context error", err)
	}
	if _, err := cluster.New("kind-tekton", ""); err == nil || !strings.Contains(err.Error(), "explicit namespace") {
		t.Fatalf("New() error = %v, want explicit namespace error", err)
	}
}
