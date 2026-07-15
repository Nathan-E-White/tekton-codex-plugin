package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s, err := server.New()
	if err == nil {
		err = s.Run(context.Background(), &mcp.StdioTransport{})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
