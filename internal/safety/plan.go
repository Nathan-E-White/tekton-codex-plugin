package safety

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Profile string

const (
	ProfileDev  Profile = "dev"
	ProfileStg  Profile = "stg"
	ProfileProd Profile = "prod"
)

var (
	ErrBackupRequired = errors.New("teardown backup proof is required")
	ErrConfirmation   = errors.New("confirmation token does not match plan")
	ErrConsumed       = errors.New("mutation plan has already been consumed")
	ErrDrift          = errors.New("cluster identity or state changed after planning")
	ErrExpired        = errors.New("mutation plan expired")
	ErrUnavailable    = errors.New("mutation plan execution payload is unavailable")
)

func (p Profile) Valid() bool {
	return p == ProfileDev || p == ProfileStg || p == ProfileProd
}

type PlanInput struct {
	Action            string
	Context           string
	Namespace         string
	Profile           Profile
	ClusterIdentity   string
	StateHash         string
	Destructive       bool
	BackupReference   string
	EvidenceLocation  string
	DataLossInventory map[string]int64
}

type Operation struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Stdin   []byte   `json:"-"`
}

type OperationSummary struct {
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	StdinSHA256 string   `json:"stdin_sha256,omitempty"`
}

type Plan struct {
	ID                string             `json:"id"`
	Action            string             `json:"action"`
	Context           string             `json:"context"`
	Namespace         string             `json:"namespace,omitempty"`
	Profile           Profile            `json:"profile"`
	ClusterIdentity   string             `json:"cluster_identity"`
	StateHash         string             `json:"state_hash"`
	Destructive       bool               `json:"destructive"`
	BackupReference   string             `json:"backup_reference,omitempty"`
	EvidenceLocation  string             `json:"evidence_location"`
	DataLossInventory map[string]int64   `json:"data_loss_inventory,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	ExpiresAt         time.Time          `json:"expires_at"`
	ConsumedAt        *time.Time         `json:"consumed_at,omitempty"`
	Operations        []OperationSummary `json:"operations"`
}

func (p Plan) ConfirmationToken() string {
	return fmt.Sprintf("CONFIRM %s %s %s", p.Action, p.Context, p.ID)
}

type Store struct {
	dir      string
	ttl      time.Duration
	now      func() time.Time
	mu       sync.Mutex
	plans    map[string]Plan
	payloads map[string][]Operation
}

func NewStore(dir string, ttl time.Duration, now func() time.Time) (*Store, error) {
	if ttl <= 0 {
		return nil, errors.New("plan TTL must be positive")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(filepath.Join(dir, "plans"), 0o700); err != nil {
		return nil, fmt.Errorf("create plan directory: %w", err)
	}
	return &Store{dir: dir, ttl: ttl, now: now, plans: map[string]Plan{}, payloads: map[string][]Operation{}}, nil
}

func (s *Store) Create(input PlanInput, operations []Operation) (Plan, error) {
	if strings.TrimSpace(input.Action) == "" || strings.TrimSpace(input.Context) == "" {
		return Plan{}, errors.New("action and context are required")
	}
	if !input.Profile.Valid() {
		return Plan{}, fmt.Errorf("invalid environment profile %q", input.Profile)
	}
	if input.ClusterIdentity == "" || input.StateHash == "" {
		return Plan{}, errors.New("cluster identity and state hash are required")
	}
	if input.Destructive && (input.Profile == ProfileStg || input.Profile == ProfileProd) && strings.TrimSpace(input.BackupReference) == "" && input.Action == "teardown" {
		return Plan{}, ErrBackupRequired
	}
	summaries := summarize(operations)
	created := s.now().UTC()
	plan := Plan{
		Action: input.Action, Context: input.Context, Namespace: input.Namespace, Profile: input.Profile,
		ClusterIdentity: input.ClusterIdentity, StateHash: input.StateHash, Destructive: input.Destructive,
		BackupReference: input.BackupReference, CreatedAt: created, ExpiresAt: created.Add(s.ttl), Operations: summaries,
		EvidenceLocation: input.EvidenceLocation, DataLossInventory: input.DataLossInventory,
	}
	plan.ID = planDigest(plan)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.ID] = plan
	s.payloads[plan.ID] = cloneOperations(operations)
	if err := s.persist(plan); err != nil {
		delete(s.plans, plan.ID)
		delete(s.payloads, plan.ID)
		return Plan{}, err
	}
	_ = s.appendEvidence(map[string]any{"event": "plan_created", "plan_id": plan.ID, "action": plan.Action, "context": plan.Context, "profile": plan.Profile, "created_at": created})
	return plan, nil
}

// Lookup returns immutable plan metadata without exposing the executable payload.
func (s *Store) Lookup(id string) (Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	if !ok {
		return Plan{}, ErrUnavailable
	}
	return plan, nil
}

func (s *Store) Consume(id, confirmation, clusterIdentity, stateHash string) (Plan, []Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	if !ok {
		return Plan{}, nil, ErrUnavailable
	}
	if plan.ConsumedAt != nil {
		return Plan{}, nil, ErrConsumed
	}
	if !s.now().Before(plan.ExpiresAt) {
		return Plan{}, nil, ErrExpired
	}
	if confirmation != plan.ConfirmationToken() {
		return Plan{}, nil, ErrConfirmation
	}
	if clusterIdentity != plan.ClusterIdentity || stateHash != plan.StateHash {
		return Plan{}, nil, ErrDrift
	}
	operations, ok := s.payloads[id]
	if !ok {
		return Plan{}, nil, ErrUnavailable
	}
	consumed := s.now().UTC()
	plan.ConsumedAt = &consumed
	s.plans[id] = plan
	delete(s.payloads, id)
	if err := s.persist(plan); err != nil {
		return Plan{}, nil, err
	}
	_ = s.appendEvidence(map[string]any{"event": "plan_consumed", "plan_id": plan.ID, "action": plan.Action, "context": plan.Context, "consumed_at": consumed})
	return plan, cloneOperations(operations), nil
}

func (s *Store) RecordExecution(plan Plan, status string, details any) error {
	return s.appendEvidence(map[string]any{
		"event": "plan_execution", "plan_id": plan.ID, "action": plan.Action, "context": plan.Context,
		"profile": plan.Profile, "status": status, "details": details, "recorded_at": s.now().UTC(),
	})
}

func summarize(operations []Operation) []OperationSummary {
	result := make([]OperationSummary, 0, len(operations))
	for _, operation := range operations {
		summary := OperationSummary{Command: operation.Command, Args: append([]string(nil), operation.Args...)}
		if len(operation.Stdin) > 0 {
			digest := sha256.Sum256(operation.Stdin)
			summary.StdinSHA256 = hex.EncodeToString(digest[:])
		}
		result = append(result, summary)
	}
	return result
}

func cloneOperations(operations []Operation) []Operation {
	result := make([]Operation, len(operations))
	for i, operation := range operations {
		result[i] = Operation{Command: operation.Command, Args: append([]string(nil), operation.Args...), Stdin: append([]byte(nil), operation.Stdin...)}
	}
	return result
}

func planDigest(plan Plan) string {
	plan.ID = ""
	plan.ConsumedAt = nil
	b, _ := json.Marshal(plan)
	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:])
}

func (s *Store) persist(plan Plan) error {
	b, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}
	path := filepath.Join(s.dir, "plans", plan.ID+".json")
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	return nil
}

func (s *Store) appendEvidence(event any) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, "evidence.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
