package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/bundle"
	"github.com/Nathan-E-White/tekton-codex-plugin/internal/cluster"
	"github.com/Nathan-E-White/tekton-codex-plugin/internal/manifest"
	"github.com/Nathan-E-White/tekton-codex-plugin/internal/runner"
	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const instructions = "Operate upstream Tekton only. Read local kubeconfig with an explicit context and never switch global context. All mutations require profile dev, stg, or prod and a fresh immutable 15-minute plan. Execute only with the plan ID and its exact confirmation token. Reject mixed or unsupported component versions, Secret data, private keys, tokens, and paths outside configured manifest roots. Logs are bounded, redacted, and never persisted by default. Staging and production teardown require an export plus external Results database backup reference."

type Service struct {
	plans     *safety.Store
	runner    runner.Runner
	paths     *safety.PathPolicy
	artifacts string
}

type Output struct {
	OK   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

type ScopeInput struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace,omitempty"`
}

type RunInput struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
}

type ValidateInput struct {
	Context    string `json:"context"`
	Namespace  string `json:"namespace"`
	InlineYAML string `json:"inline_yaml,omitempty"`
	Path       string `json:"path,omitempty"`
}

type LogsInput struct {
	Context        string `json:"context"`
	Namespace      string `json:"namespace"`
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	MaxBytes       int    `json:"max_bytes,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type ResultsInput struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
	Filter    string `json:"filter,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type VerifyInput struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
	Target    string `json:"target"`
	Identity  string `json:"identity,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
}

type BackupInput struct {
	Context                   string `json:"context"`
	Namespace                 string `json:"namespace"`
	ExternalDatabaseReference string `json:"external_database_reference"`
}

type PlatformPlanInput struct {
	Context         string         `json:"context"`
	Namespace       string         `json:"namespace"`
	Profile         safety.Profile `json:"profile"`
	Action          string         `json:"action"`
	BackupReference string         `json:"backup_reference,omitempty"`
}

type ResourcePlanInput struct {
	Context    string         `json:"context"`
	Namespace  string         `json:"namespace"`
	Profile    safety.Profile `json:"profile"`
	Action     string         `json:"action"`
	InlineYAML string         `json:"inline_yaml,omitempty"`
	Path       string         `json:"path,omitempty"`
}

type RunPlanInput struct {
	Context   string         `json:"context"`
	Namespace string         `json:"namespace"`
	Profile   safety.Profile `json:"profile"`
	Action    string         `json:"action"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
}

type ExecuteInput struct {
	PlanID       string `json:"plan_id"`
	Confirmation string `json:"confirmation"`
}

func New() (*mcp.Server, error) {
	artifacts := os.Getenv("TEKTON_MCP_ARTIFACT_DIR")
	if artifacts == "" {
		artifacts = filepath.Join(os.TempDir(), "tekton-codex-plugin")
	}
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		return nil, err
	}
	store, err := safety.NewStore(artifacts, 15*time.Minute, nil)
	if err != nil {
		return nil, err
	}
	var policy *safety.PathPolicy
	if roots := splitRoots(os.Getenv("TEKTON_MCP_MANIFEST_ROOTS")); len(roots) > 0 {
		policy, err = safety.NewPathPolicy(roots)
		if err != nil {
			return nil, err
		}
	}
	service := &Service{plans: store, runner: runner.Runner{Timeout: 30 * time.Second, MaxBytes: 256 * 1024}, paths: policy, artifacts: artifacts}
	server := mcp.NewServer(&mcp.Implementation{Name: "tekton", Title: "Tekton Codex Plugin", Version: "0.1.0", WebsiteURL: "https://github.com/Nathan-E-White/tekton-codex-plugin"}, &mcp.ServerOptions{Instructions: instructions})
	service.addTools(server)
	return server, nil
}

func (s *Service) addTools(server *mcp.Server) {
	readOnly := func(title string) *mcp.ToolAnnotations {
		return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true}
	}
	plan := func(title string) *mcp.ToolAnnotations {
		value := false
		return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, DestructiveHint: &value, IdempotentHint: true}
	}
	execute := func(title string) *mcp.ToolAnnotations {
		value := true
		return &mcp.ToolAnnotations{Title: title, DestructiveHint: &value}
	}
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_preflight", Description: "Resolve explicit cluster scope, RBAC reachability, pinned bundle health, and manifest roots.", Annotations: readOnly("Tekton preflight")}, s.preflight)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_validate", Description: "Validate inline or allowed-root YAML and run Kubernetes server-side dry-run.", Annotations: readOnly("Validate Tekton resources")}, s.validate)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_list_runs", Description: "List PipelineRuns or TaskRuns with status and timing.", Annotations: readOnly("List Tekton runs")}, s.listRuns)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_get_run", Description: "Get one PipelineRun or TaskRun, including conditions, steps, timing, and ownership.", Annotations: readOnly("Get Tekton run")}, s.getRun)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_get_logs", Description: "Fetch bounded, timed, redacted run logs without persistence.", Annotations: readOnly("Get bounded Tekton logs")}, s.getLogs)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_query_results", Description: "Query Tekton Results records and retention metadata.", Annotations: readOnly("Query Tekton Results")}, s.queryResults)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_verify_attestation", Description: "Inspect Chains status and verify attestation signatures without reading credential material.", Annotations: readOnly("Verify Tekton attestation")}, s.verifyAttestation)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_export_teardown_backup", Description: "Export non-secret Tekton resources with hashes and bind an external Results database backup reference.", Annotations: readOnly("Export teardown backup")}, s.exportBackup)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_plan_platform", Description: "Create a fresh immutable install, reconcile, repair, or teardown plan for the pinned bundle.", Annotations: plan("Plan Tekton platform change")}, s.planPlatform)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_plan_resources", Description: "Create a fresh immutable apply or delete plan for validated resources.", Annotations: plan("Plan Tekton resource change")}, s.planResources)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_plan_run", Description: "Create a fresh immutable start, retry, cancel, or cleanup plan.", Annotations: plan("Plan Tekton run change")}, s.planRun)
	mcp.AddTool(server, &mcp.Tool{Name: "tekton_execute_plan", Description: "Execute one fresh immutable plan using its exact confirmation token; drift invalidates it.", Annotations: execute("Execute Tekton plan")}, s.executePlan)
}

func (s *Service) preflight(ctx context.Context, _ *mcp.CallToolRequest, in ScopeInput) (*mcp.CallToolResult, Output, error) {
	c, identity, state, err := scoped(ctx, in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	components, supported, err := inspectBundle(ctx, c)
	if err != nil {
		return nil, Output{}, err
	}
	roots := []string{}
	if s.paths != nil {
		roots = s.paths.Roots()
	}
	tknReport, tknSupported := s.inspectTKN(ctx)
	rbac := map[string]any{}
	rbacReady := true
	for name, check := range map[string][3]string{
		"inspect_runs": {"list", "tekton.dev", "pipelineruns"},
		"create_runs":  {"create", "tekton.dev", "pipelineruns"},
		"read_logs":    {"get", "", "pods/log"},
	} {
		allowed, reason, reviewErr := c.AccessReview(ctx, check[0], check[1], check[2], in.Namespace)
		if reviewErr != nil {
			rbac[name] = map[string]any{"allowed": false, "error": safety.Redact(reviewErr.Error())}
			rbacReady = false
			continue
		}
		rbac[name] = map[string]any{"allowed": allowed, "reason": reason}
		rbacReady = rbacReady && allowed
	}
	return nil, Output{OK: supported && rbacReady && tknSupported, Data: map[string]any{"context": c.Context, "namespace": c.Namespace, "cluster_identity": identity, "state_hash": state, "bundle_supported": supported, "rbac_ready": rbacReady, "rbac": rbac, "components": components, "manifest_roots": roots, "tkn": tknReport}}, nil
}

func (s *Service) validate(ctx context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, Output, error) {
	doc, err := manifest.Load(in.InlineYAML, in.Path, s.paths)
	if err != nil {
		return nil, Output{}, err
	}
	args := []string{"--context", in.Context, "--namespace", in.Namespace, "apply", "--server-side", "--dry-run=server", "-f", "-"}
	result, err := s.runner.Run(ctx, "kubectl", args, doc.Bytes)
	if err != nil {
		return nil, Output{}, fmt.Errorf("server-side dry-run failed: %w: %s", err, result.Output)
	}
	return nil, Output{OK: true, Data: map[string]any{"sha256": doc.SHA256, "resources": doc.Resources, "dry_run": result}}, nil
}

func (s *Service) listRuns(ctx context.Context, _ *mcp.CallToolRequest, in RunInput) (*mcp.CallToolResult, Output, error) {
	c, err := cluster.New(in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	runs, err := c.ListRuns(ctx, in.Namespace, in.Kind, in.Limit)
	if err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: runs}, nil
}

func (s *Service) getRun(ctx context.Context, _ *mcp.CallToolRequest, in RunInput) (*mcp.CallToolResult, Output, error) {
	c, err := cluster.New(in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	run, err := c.GetRun(ctx, in.Namespace, in.Kind, in.Name)
	if err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: run}, nil
}

func (s *Service) getLogs(ctx context.Context, _ *mcp.CallToolRequest, in LogsInput) (*mcp.CallToolResult, Output, error) {
	if in.MaxBytes <= 0 || in.MaxBytes > 1024*1024 {
		in.MaxBytes = 256 * 1024
	}
	if in.TimeoutSeconds <= 0 || in.TimeoutSeconds > 60 {
		in.TimeoutSeconds = 30
	}
	r := runner.Runner{MaxBytes: in.MaxBytes, Timeout: time.Duration(in.TimeoutSeconds) * time.Second}
	resource := "pipelinerun"
	if strings.HasPrefix(strings.ToLower(in.Kind), "task") {
		resource = "taskrun"
	}
	result, err := r.Run(ctx, "tkn", []string{"--context", in.Context, "-n", in.Namespace, resource, "logs", in.Name, "--all"}, nil)
	if err != nil {
		return nil, Output{}, fmt.Errorf("log retrieval failed: %w: %s", err, result.Output)
	}
	return nil, Output{OK: true, Data: result}, nil
}

func (s *Service) queryResults(ctx context.Context, _ *mcp.CallToolRequest, in ResultsInput) (*mcp.CallToolResult, Output, error) {
	if in.Limit <= 0 || in.Limit > 200 {
		in.Limit = 50
	}
	args := []string{"records", "list", "--context", in.Context, "--namespace", in.Namespace, "--limit", fmt.Sprint(in.Limit), "--output", "json"}
	if in.Filter != "" {
		args = append(args, "--filter", in.Filter)
	}
	result, err := s.runner.Run(ctx, "tkn-results", args, nil)
	if err != nil {
		return nil, Output{}, fmt.Errorf("Results query unavailable: %w: %s", err, result.Output)
	}
	return nil, Output{OK: true, Data: result}, nil
}

func (s *Service) verifyAttestation(ctx context.Context, _ *mcp.CallToolRequest, in VerifyInput) (*mcp.CallToolResult, Output, error) {
	args := []string{"verify-attestation", in.Target, "--type", "slsaprovenance"}
	if in.Identity != "" {
		args = append(args, "--certificate-identity", in.Identity)
	}
	if in.Issuer != "" {
		args = append(args, "--certificate-oidc-issuer", in.Issuer)
	}
	result, err := s.runner.Run(ctx, "cosign", args, nil)
	if err != nil {
		return nil, Output{}, fmt.Errorf("attestation verification failed: %w: %s", err, result.Output)
	}
	return nil, Output{OK: true, Data: map[string]any{"context": in.Context, "namespace": in.Namespace, "verification": result}}, nil
}

func (s *Service) exportBackup(ctx context.Context, _ *mcp.CallToolRequest, in BackupInput) (*mcp.CallToolResult, Output, error) {
	if strings.TrimSpace(in.ExternalDatabaseReference) == "" {
		return nil, Output{}, errors.New("external Results database backup reference is required")
	}
	args := []string{"--context", in.Context, "get", "pipelines.tekton.dev,tasks.tekton.dev,pipelineruns.tekton.dev,taskruns.tekton.dev,eventlisteners.triggers.tekton.dev,triggerbindings.triggers.tekton.dev,triggertemplates.triggers.tekton.dev", "--all-namespaces", "-o", "yaml", "--ignore-not-found"}
	result, err := runner.Runner{Timeout: 60 * time.Second, MaxBytes: 8 * 1024 * 1024}.Run(ctx, "kubectl", args, nil)
	if err != nil {
		return nil, Output{}, fmt.Errorf("resource export failed: %w: %s", err, result.Output)
	}
	digest := sha256.Sum256([]byte(result.Output))
	hash := hex.EncodeToString(digest[:])
	path := filepath.Join(s.artifacts, "teardown-export-"+hash[:16]+".yaml")
	if err := os.WriteFile(path, []byte(result.Output), 0o600); err != nil {
		return nil, Output{}, err
	}
	proof := map[string]any{"context": in.Context, "namespace": in.Namespace, "resource_export": path, "sha256": hash, "external_database_reference": in.ExternalDatabaseReference, "created_at": time.Now().UTC()}
	b, _ := json.Marshal(proof)
	proofHash := sha256.Sum256(b)
	proofID := hex.EncodeToString(proofHash[:])
	if err := os.WriteFile(filepath.Join(s.artifacts, "backup-proof-"+proofID+".json"), append(b, '\n'), 0o600); err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: map[string]any{"proof": proof, "proof_sha256": proofID}}, nil
}

func (s *Service) planPlatform(ctx context.Context, _ *mcp.CallToolRequest, in PlatformPlanInput) (*mcp.CallToolResult, Output, error) {
	c, identity, state, err := scoped(ctx, in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	components, supported, err := inspectBundle(ctx, c)
	if err != nil {
		return nil, Output{}, err
	}
	if !supported {
		return nil, Output{}, fmt.Errorf("unsupported or mixed Tekton component bundle: %v", components)
	}
	operations := []safety.Operation{}
	var inventory map[string]int64
	switch in.Action {
	case "install", "reconcile", "repair":
		for _, item := range bundle.Components {
			operations = append(operations, safety.Operation{Command: "kubectl", Args: []string{"--context", in.Context, "apply", "--server-side", "-f", item.Manifest}})
		}
	case "teardown":
		inventory, err = c.DataLossInventory(ctx)
		if err != nil {
			return nil, Output{}, fmt.Errorf("teardown requires data-loss enumeration: %w", err)
		}
		if in.Profile == safety.ProfileStg || in.Profile == safety.ProfileProd {
			if err := s.verifyBackupProof(in.BackupReference, in.Context, in.Namespace); err != nil {
				return nil, Output{}, err
			}
		}
		for _, item := range bundle.Reverse() {
			operations = append(operations, safety.Operation{Command: "kubectl", Args: []string{"--context", in.Context, "delete", "--ignore-not-found", "-f", item.Manifest}})
		}
	default:
		return nil, Output{}, fmt.Errorf("unsupported platform action %q", in.Action)
	}
	plan, err := s.plans.Create(safety.PlanInput{Action: in.Action, Context: in.Context, Namespace: in.Namespace, Profile: in.Profile, ClusterIdentity: identity, StateHash: state, Destructive: in.Action == "teardown", BackupReference: in.BackupReference, EvidenceLocation: filepath.Join(s.artifacts, "evidence.jsonl"), DataLossInventory: inventory}, operations)
	if err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: planResponse(plan)}, nil
}

func (s *Service) planResources(ctx context.Context, _ *mcp.CallToolRequest, in ResourcePlanInput) (*mcp.CallToolResult, Output, error) {
	if in.Action != "apply" && in.Action != "delete" {
		return nil, Output{}, fmt.Errorf("action must be apply or delete")
	}
	doc, err := manifest.Load(in.InlineYAML, in.Path, s.paths)
	if err != nil {
		return nil, Output{}, err
	}
	_, identity, state, err := scoped(ctx, in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	dryRunArgs := []string{"--context", in.Context, "--namespace", in.Namespace, in.Action, "--dry-run=server", "-f", "-"}
	if in.Action == "apply" {
		dryRunArgs = []string{"--context", in.Context, "--namespace", in.Namespace, "apply", "--server-side", "--dry-run=server", "-f", "-"}
	}
	if result, runErr := s.runner.Run(ctx, "kubectl", dryRunArgs, doc.Bytes); runErr != nil {
		return nil, Output{}, fmt.Errorf("resource plan server-side validation failed: %w: %s", runErr, result.Output)
	}
	args := []string{"--context", in.Context, "--namespace", in.Namespace, in.Action, "-f", "-"}
	if in.Action == "apply" {
		args = []string{"--context", in.Context, "--namespace", in.Namespace, "apply", "--server-side", "-f", "-"}
	}
	plan, err := s.plans.Create(safety.PlanInput{Action: in.Action, Context: in.Context, Namespace: in.Namespace, Profile: in.Profile, ClusterIdentity: identity, StateHash: state, Destructive: in.Action == "delete", EvidenceLocation: filepath.Join(s.artifacts, "evidence.jsonl")}, []safety.Operation{{Command: "kubectl", Args: args, Stdin: doc.Bytes}})
	if err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: map[string]any{"plan": planResponse(plan), "resources": doc.Resources, "manifest_sha256": doc.SHA256}}, nil
}

func (s *Service) planRun(ctx context.Context, _ *mcp.CallToolRequest, in RunPlanInput) (*mcp.CallToolResult, Output, error) {
	_, identity, state, err := scoped(ctx, in.Context, in.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	if report, supported := s.inspectTKN(ctx); !supported {
		return nil, Output{}, fmt.Errorf("unsupported tkn client: %v", report)
	}
	resource := "pipeline"
	runResource := "pipelinerun"
	if strings.HasPrefix(strings.ToLower(in.Kind), "task") {
		resource, runResource = "task", "taskrun"
	}
	args := []string{"--context", in.Context, "-n", in.Namespace}
	switch in.Action {
	case "start":
		args = append(args, resource, "start", in.Name)
	case "retry":
		args = append(args, runResource, "retry", in.Name)
	case "cancel":
		args = append(args, runResource, "cancel", in.Name)
	case "run-cleanup":
		args = append(args, runResource, "delete", in.Name, "-f")
	default:
		return nil, Output{}, fmt.Errorf("unsupported run action %q", in.Action)
	}
	plan, err := s.plans.Create(safety.PlanInput{Action: in.Action, Context: in.Context, Namespace: in.Namespace, Profile: in.Profile, ClusterIdentity: identity, StateHash: state, Destructive: in.Action == "cancel" || in.Action == "run-cleanup", EvidenceLocation: filepath.Join(s.artifacts, "evidence.jsonl")}, []safety.Operation{{Command: "tkn", Args: args}})
	if err != nil {
		return nil, Output{}, err
	}
	return nil, Output{OK: true, Data: planResponse(plan)}, nil
}

func (s *Service) executePlan(ctx context.Context, _ *mcp.CallToolRequest, in ExecuteInput) (*mcp.CallToolResult, Output, error) {
	metadata, err := s.plans.Lookup(in.PlanID)
	if err != nil {
		return nil, Output{}, err
	}
	_, identity, state, err := scoped(ctx, metadata.Context, metadata.Namespace)
	if err != nil {
		return nil, Output{}, err
	}
	plan, operations, err := s.plans.Consume(in.PlanID, in.Confirmation, identity, state)
	if err != nil {
		return nil, Output{}, err
	}
	results := make([]runner.Result, 0, len(operations))
	for _, operation := range operations {
		result, runErr := s.runner.Run(ctx, operation.Command, operation.Args, operation.Stdin)
		results = append(results, result)
		if runErr != nil {
			_ = s.plans.RecordExecution(plan, "failed", results)
			return nil, Output{}, fmt.Errorf("plan execution failed: %w: %s", runErr, result.Output)
		}
	}
	_ = s.plans.RecordExecution(plan, "succeeded", results)
	return nil, Output{OK: true, Data: map[string]any{"plan_id": plan.ID, "status": "succeeded", "results": results}}, nil
}

func scoped(ctx context.Context, contextName, namespace string) (*cluster.Client, string, string, error) {
	c, err := cluster.New(contextName, namespace)
	if err != nil {
		return nil, "", "", err
	}
	identity, err := c.Identity(ctx)
	if err != nil {
		return nil, "", "", err
	}
	state, err := c.StateHash(ctx)
	if err != nil {
		return nil, "", "", err
	}
	return c, identity, state, nil
}

func planResponse(plan safety.Plan) map[string]any {
	return map[string]any{"plan": plan, "confirmation": plan.ConfirmationToken()}
}

func inspectBundle(ctx context.Context, c *cluster.Client) (map[string]any, bool, error) {
	reports := map[string]any{}
	supported := true
	for _, component := range bundle.Components {
		deployments, err := c.Deployments(ctx, component.Namespace)
		if err != nil {
			reports[component.Name] = map[string]any{"expected": component.Version, "status": "unavailable", "error": safety.Redact(err.Error())}
			supported = false
			continue
		}
		matched := make([]cluster.DeploymentInfo, 0)
		versions := map[string]bool{}
		unknown := false
		for _, deployment := range deployments {
			if !deploymentBelongsTo(component.Name, component.Namespace, deployment.Name) {
				continue
			}
			matched = append(matched, deployment)
			version := deployment.Labels["app.kubernetes.io/version"]
			if version == "" {
				for _, image := range deployment.Images {
					if strings.Contains(image, component.Version) {
						version = component.Version
						break
					}
				}
			}
			if version == "" {
				unknown = true
			} else {
				versions[version] = true
			}
		}
		status := "supported"
		if len(matched) == 0 {
			status = "absent"
		} else if unknown {
			status = "unknown"
			supported = false
		} else if len(versions) != 1 || !versionSetContains(versions, component.Version) {
			status = "unsupported"
			if len(versions) > 1 {
				status = "mixed"
			}
			supported = false
		}
		observed := make([]string, 0, len(versions))
		for version := range versions {
			observed = append(observed, version)
		}
		reports[component.Name] = map[string]any{"expected": component.Version, "status": status, "observed_versions": observed, "deployments": matched}
	}
	return reports, supported, nil
}

func versionSetContains(versions map[string]bool, expected string) bool {
	for version := range versions {
		if strings.TrimPrefix(version, "v") == strings.TrimPrefix(expected, "v") {
			return true
		}
	}
	return false
}

func deploymentBelongsTo(component, namespace, name string) bool {
	switch component {
	case "pipelines":
		return strings.HasPrefix(name, "tekton-pipelines-")
	case "triggers":
		return strings.HasPrefix(name, "tekton-triggers-")
	case "results":
		return strings.HasPrefix(name, "tekton-results-")
	case "chains":
		return namespace == "tekton-chains"
	case "pipelines-as-code":
		return namespace == "pipelines-as-code"
	default:
		return false
	}
}

func (s *Service) verifyBackupProof(id, contextName, namespace string) error {
	if len(id) != 64 || strings.Trim(id, "0123456789abcdef") != "" {
		return safety.ErrBackupRequired
	}
	b, err := os.ReadFile(filepath.Join(s.artifacts, "backup-proof-"+id+".json"))
	if err != nil {
		return fmt.Errorf("%w: proof %q is unavailable", safety.ErrBackupRequired, id)
	}
	var proof struct {
		Context                   string `json:"context"`
		Namespace                 string `json:"namespace"`
		ResourceExport            string `json:"resource_export"`
		SHA256                    string `json:"sha256"`
		ExternalDatabaseReference string `json:"external_database_reference"`
	}
	if err := json.Unmarshal(b, &proof); err != nil {
		return fmt.Errorf("%w: malformed proof", safety.ErrBackupRequired)
	}
	if proof.Context != contextName || proof.Namespace != namespace || proof.ResourceExport == "" || proof.SHA256 == "" || strings.TrimSpace(proof.ExternalDatabaseReference) == "" {
		return fmt.Errorf("%w: proof scope or fields do not match", safety.ErrBackupRequired)
	}
	exportBytes, err := os.ReadFile(proof.ResourceExport)
	if err != nil {
		return fmt.Errorf("%w: resource export is unavailable", safety.ErrBackupRequired)
	}
	digest := sha256.Sum256(exportBytes)
	if hex.EncodeToString(digest[:]) != proof.SHA256 {
		return fmt.Errorf("%w: resource export hash mismatch", safety.ErrBackupRequired)
	}
	return nil
}

func (s *Service) inspectTKN(ctx context.Context) (map[string]any, bool) {
	result, err := s.runner.Run(ctx, "tkn", []string{"version"}, nil)
	if err != nil {
		return map[string]any{"expected": bundle.TKNVersion, "status": "unavailable", "error": safety.Redact(err.Error())}, false
	}
	supported := strings.Contains(result.Output, bundle.TKNVersion) || strings.Contains(result.Output, strings.TrimPrefix(bundle.TKNVersion, "v"))
	status := "supported"
	if !supported {
		status = "unsupported"
	}
	return map[string]any{"expected": bundle.TKNVersion, "status": status, "version_output": result.Output}, supported
}

func splitRoots(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == os.PathListSeparator })
	result := []string{}
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}
