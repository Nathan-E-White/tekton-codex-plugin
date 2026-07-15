package mcpapp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDashboardResourceContract(t *testing.T) {
	result, err := readDashboard(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("dashboard returned %d contents, want 1", len(result.Contents))
	}
	content := result.Contents[0]
	if content.URI != DashboardURI || content.MIMEType != "text/html;profile=mcp-app" {
		t.Fatalf("dashboard contract = %#v", content)
	}
	for _, expected := range []string{
		"Tekton Operations", "window.openai.callTool", "tools/call", "ui/notifications/tool-result", "tekton_preflight", "tekton_validate",
		"tekton_list_runs", "tekton_get_run", "tekton_get_logs", "tekton_query_results",
		"tekton_verify_attestation", "tekton_export_teardown_backup", "tekton_plan_platform",
		"tekton_plan_resources", "tekton_plan_run", "tekton_execute_plan",
		"Tekton™ is a trademark of The Linux Foundation",
	} {
		if !strings.Contains(content.Text, expected) {
			t.Fatalf("dashboard HTML missing %q", expected)
		}
	}
	if strings.Contains(content.Text, "{{TEKTON_ICON}}") {
		t.Fatal("dashboard icon placeholder was not resolved")
	}
	if got := content.Meta["openai/widgetPrefersBorder"]; got != true {
		t.Fatalf("widget border metadata = %#v", got)
	}
}

func TestToolMetaTargetsDashboard(t *testing.T) {
	meta := ToolMeta("Inspecting Tekton", "Tekton inspection ready")
	if got := meta["openai/outputTemplate"]; got != DashboardURI {
		t.Fatalf("output template = %#v", got)
	}
	ui, ok := meta["ui"].(mcp.Meta)
	if !ok || ui["resourceUri"] != DashboardURI {
		t.Fatalf("standard UI metadata = %#v", meta["ui"])
	}
	if got := meta["openai/widgetAccessible"]; got != true {
		t.Fatalf("widget access = %#v", got)
	}
}
