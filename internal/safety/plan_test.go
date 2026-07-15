package safety_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
)

func TestMutationPlanRequiresExactConfirmationAndIsSingleUse(t *testing.T) {
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	store, err := safety.NewStore(t.TempDir(), 15*time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	plan, err := store.Create(safety.PlanInput{
		Action: "cancel", Context: "kind-tekton", Namespace: "ci", Profile: safety.ProfileDev,
		ClusterIdentity: "cluster-a", StateHash: "state-a", Destructive: true,
	}, []safety.Operation{{Command: "tkn", Args: []string{"pipelinerun", "cancel", "run-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.ID) != 64 {
		t.Fatalf("plan ID length = %d, want 64", len(plan.ID))
	}
	if _, _, err := store.Consume(plan.ID, "CONFIRM cancel wrong", "cluster-a", "state-a"); !errors.Is(err, safety.ErrConfirmation) {
		t.Fatalf("Consume() error = %v, want ErrConfirmation", err)
	}
	if _, _, err := store.Consume(plan.ID, plan.ConfirmationToken(), "cluster-a", "different"); !errors.Is(err, safety.ErrDrift) {
		t.Fatalf("Consume() error = %v, want ErrDrift", err)
	}
	if _, _, err := store.Consume(plan.ID, plan.ConfirmationToken(), "cluster-a", "state-a"); err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if _, _, err := store.Consume(plan.ID, plan.ConfirmationToken(), "cluster-a", "state-a"); !errors.Is(err, safety.ErrConsumed) {
		t.Fatalf("second Consume() error = %v, want ErrConsumed", err)
	}
}

func TestMutationPlanExpiresAndRequiresBackupProof(t *testing.T) {
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	store, err := safety.NewStore(t.TempDir(), 15*time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Create(safety.PlanInput{
		Action: "teardown", Context: "prod-us", Profile: safety.ProfileProd,
		ClusterIdentity: "cluster-a", StateHash: "state-a", Destructive: true,
	}, nil)
	if !errors.Is(err, safety.ErrBackupRequired) {
		t.Fatalf("Create() error = %v, want ErrBackupRequired", err)
	}
	plan, err := store.Create(safety.PlanInput{
		Action: "apply", Context: "kind-tekton", Profile: safety.ProfileDev,
		ClusterIdentity: "cluster-a", StateHash: "state-a",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(16 * time.Minute)
	if _, _, err := store.Consume(plan.ID, plan.ConfirmationToken(), "cluster-a", "state-a"); !errors.Is(err, safety.ErrExpired) {
		t.Fatalf("Consume() error = %v, want ErrExpired", err)
	}
}

func TestPlanEvidenceNeverPersistsOperationStdin(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	store, err := safety.NewStore(dir, 15*time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	plan, err := store.Create(safety.PlanInput{
		Action: "apply", Context: "kind-tekton", Profile: safety.ProfileDev,
		ClusterIdentity: "cluster-a", StateHash: "state-a",
	}, []safety.Operation{{Command: "kubectl", Args: []string{"apply", "-f", "-"}, Stdin: []byte("private execution payload")}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "plans", plan.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "private execution payload") {
		t.Fatalf("plan evidence persisted operation stdin: %s", b)
	}
}

func TestPlanBindsEvidenceAndDataLossInventory(t *testing.T) {
	store, err := safety.NewStore(t.TempDir(), 15*time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := store.Create(safety.PlanInput{
		Action: "teardown", Context: "kind-tekton", Namespace: "ci", Profile: safety.ProfileDev,
		ClusterIdentity: "cluster-a", StateHash: "state-a", Destructive: true,
		EvidenceLocation: "/tmp/evidence.jsonl", DataLossInventory: map[string]int64{"tekton.dev/pipelineruns": 7},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.EvidenceLocation != "/tmp/evidence.jsonl" || plan.DataLossInventory["tekton.dev/pipelineruns"] != 7 {
		t.Fatalf("plan did not bind safety evidence: %#v", plan)
	}
}
