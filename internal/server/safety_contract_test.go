package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/runner"
)

func TestDeploymentClassificationAndVersionNormalization(t *testing.T) {
	if !deploymentBelongsTo("pipelines", "tekton-pipelines", "tekton-pipelines-controller") {
		t.Fatal("pipeline controller was not classified")
	}
	if deploymentBelongsTo("pipelines", "tekton-pipelines", "tekton-triggers-controller") {
		t.Fatal("trigger controller was classified as pipelines")
	}
	if !versionSetContains(map[string]bool{"1.14.0": true}, "v1.14.0") {
		t.Fatal("equivalent release versions did not match")
	}
}

func TestInspectTKNRejectsUnsupportedClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tkn")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho 'Client version: 0.44.0'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	service := &Service{runner: runner.Runner{Timeout: time.Second, MaxBytes: 1024}}
	if report, supported := service.inspectTKN(context.Background()); supported {
		t.Fatalf("inspectTKN() accepted unsupported client: %#v", report)
	}
}

func TestBackupProofMustBindScopeAndExportHash(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.yaml")
	export := []byte("apiVersion: tekton.dev/v1\nkind: Pipeline\n")
	if err := os.WriteFile(exportPath, export, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(export)
	proof := map[string]any{
		"context": "prod-us", "namespace": "ci", "resource_export": exportPath,
		"sha256": hex.EncodeToString(digest[:]), "external_database_reference": "s3://backups/results-42",
	}
	b, _ := json.Marshal(proof)
	proofDigest := sha256.Sum256(b)
	id := hex.EncodeToString(proofDigest[:])
	if err := os.WriteFile(filepath.Join(dir, "backup-proof-"+id+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	service := &Service{artifacts: dir}
	if err := service.verifyBackupProof(id, "prod-us", "ci"); err != nil {
		t.Fatalf("verifyBackupProof() error = %v", err)
	}
	if err := service.verifyBackupProof(id, "prod-us", "wrong"); err == nil {
		t.Fatal("verifyBackupProof() accepted the wrong namespace")
	}
	if err := os.WriteFile(exportPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := service.verifyBackupProof(id, "prod-us", "ci"); err == nil {
		t.Fatal("verifyBackupProof() accepted a tampered export")
	}
}
