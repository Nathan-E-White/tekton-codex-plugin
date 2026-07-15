package server

import (
	"context"
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/mcpapp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewAdvertisesDashboardForEveryTool(t *testing.T) {
	t.Setenv("TEKTON_MCP_ARTIFACT_DIR", t.TempDir())
	t.Setenv("TEKTON_MCP_MANIFEST_ROOTS", "")
	server, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "tekton-test", Version: "0.2.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	want := map[string]bool{
		"tekton_preflight": true, "tekton_validate": true,
		"tekton_list_runs": true, "tekton_get_run": true, "tekton_get_logs": true,
		"tekton_query_results": true, "tekton_verify_attestation": true,
		"tekton_export_teardown_backup": true, "tekton_plan_platform": true,
		"tekton_plan_resources": true, "tekton_plan_run": true, "tekton_execute_plan": true,
	}
	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != len(want) {
		t.Fatalf("listed %d tools, want %d", len(listed.Tools), len(want))
	}
	for _, tool := range listed.Tools {
		if !want[tool.Name] {
			t.Errorf("unexpected tool %q", tool.Name)
		}
		if got := tool.Meta["openai/outputTemplate"]; got != mcpapp.DashboardURI {
			t.Errorf("%s output template = %#v", tool.Name, got)
		}
		if got := tool.Meta["openai/widgetAccessible"]; got != true {
			t.Errorf("%s widget access = %#v", tool.Name, got)
		}
		if tool.Annotations == nil || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Errorf("%s missing closed-world annotation", tool.Name)
		}
	}

	resources, err := clientSession.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources.Resources) != 1 || resources.Resources[0].URI != mcpapp.DashboardURI {
		t.Fatalf("resources = %#v", resources.Resources)
	}
	resource, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: mcpapp.DashboardURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].MIMEType != "text/html;profile=mcp-app" {
		t.Fatalf("dashboard resource = %#v", resource.Contents)
	}
}
