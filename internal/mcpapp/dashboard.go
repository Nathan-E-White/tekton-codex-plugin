package mcpapp

import (
	"context"
	_ "embed"
	"encoding/base64"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const DashboardURI = "ui://tekton/operations-dashboard-v1"

//go:embed dashboard.html
var dashboardHTML string

//go:embed tekton-icon.png
var tektonIcon []byte

// Register adds the self-contained MCP Apps dashboard to the server.
func Register(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		URI:         DashboardURI,
		Name:        "tekton-operations-dashboard",
		Title:       "Tekton Operations Dashboard",
		Description: "Interactive control surface for every Tekton MCP inspection, planning, execution, provenance, Results, and evidence tool.",
		MIMEType:    "text/html;profile=mcp-app",
	}, readDashboard)
}

// ToolMeta connects an MCP tool to the dashboard and supplies host status text.
func ToolMeta(invoking, invoked string) mcp.Meta {
	return mcp.Meta{
		"ui": mcp.Meta{
			"resourceUri": DashboardURI,
			"visibility":  []string{"model", "app"},
		},
		"openai/outputTemplate":          DashboardURI,
		"openai/toolInvocation/invoking": invoking,
		"openai/toolInvocation/invoked":  invoked,
		"openai/widgetAccessible":        true,
	}
}

// ResultMeta identifies the tool result projected by the dashboard.
func ResultMeta(tool string) mcp.Meta {
	return mcp.Meta{"tekton/tool": tool}
}

func readDashboard(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	html := strings.ReplaceAll(dashboardHTML, "{{TEKTON_ICON}}", base64.StdEncoding.EncodeToString(tektonIcon))
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI:      DashboardURI,
		MIMEType: "text/html;profile=mcp-app",
		Text:     html,
		Meta: mcp.Meta{
			"ui": mcp.Meta{
				"prefersBorder": true,
				"csp": mcp.Meta{
					"connectDomains":  []string{},
					"resourceDomains": []string{},
				},
			},
			"openai/widgetDescription":   "Tekton cluster scope, run status, validation, provenance, Results, immutable plans, and guarded execution dashboard.",
			"openai/widgetPrefersBorder": true,
			"openai/widgetCSP": mcp.Meta{
				"connect_domains":  []string{},
				"resource_domains": []string{},
			},
		},
	}}}, nil
}
