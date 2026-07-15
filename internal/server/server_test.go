package server

import "testing"

func TestNewRegistersTypedTools(t *testing.T) {
	t.Setenv("TEKTON_MCP_ARTIFACT_DIR", t.TempDir())
	t.Setenv("TEKTON_MCP_MANIFEST_ROOTS", "")
	if _, err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}
}
