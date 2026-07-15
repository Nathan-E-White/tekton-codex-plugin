package manifest_test

import (
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/manifest"
)

func TestLoadRejectsSecretsAndUnsupportedTektonAPIs(t *testing.T) {
	tests := []string{
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: no\n",
		"apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: old\n",
	}
	for _, input := range tests {
		if _, err := manifest.Load(input, "", nil); err == nil {
			t.Fatalf("Load(%q) succeeded, want rejection", input)
		}
	}
}

func TestLoadSummarizesSupportedResources(t *testing.T) {
	input := "apiVersion: tekton.dev/v1\nkind: Pipeline\nmetadata:\n  name: build\n  namespace: ci\nspec: {}\n"
	doc, err := manifest.Load(input, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Resources) != 1 || doc.Resources[0]["name"] != "build" || len(doc.SHA256) != 64 {
		t.Fatalf("unexpected document summary: %#v", doc)
	}
}
