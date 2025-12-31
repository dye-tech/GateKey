package policy

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/models"
)

// mockPolicyRepository is a mock implementation for testing.
type mockPolicyRepository struct {
	policies []models.Policy
	rules    map[uuid.UUID][]models.PolicyRule
}

func (m *mockPolicyRepository) List(ctx context.Context) ([]models.Policy, error) {
	return m.policies, nil
}

func (m *mockPolicyRepository) GetRules(ctx context.Context, policyID uuid.UUID) ([]models.PolicyRule, error) {
	return m.rules[policyID], nil
}

func TestEvaluate_AllowByGroup(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{
		Groups: []string{"engineering"},
	}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Networks: []string{"10.0.0.0/8"},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{
				ID:        policyID,
				Name:      "engineering-access",
				Priority:  10,
				IsEnabled: true,
			},
		},
		rules: map[uuid.UUID][]models.PolicyRule{
			policyID: {
				{
					ID:         uuid.New(),
					PolicyID:   policyID,
					Action:     "allow",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: []byte("{}"),
					Priority:   10,
				},
			},
		},
	}

	engine := NewEngine(ModeStrict, repo)
	err := engine.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh policies: %v", err)
	}

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{"engineering", "vpn-users"},
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "prod-gateway",
	}

	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	req := EvaluationRequest{
		User:    user,
		Gateway: gateway,
		Resource: Resource{
			Network: *network,
		},
		Time: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Expected access to be allowed, got denied: %s", result.Reason)
	}
}

func TestEvaluate_DenyByDefault(t *testing.T) {
	repo := &mockPolicyRepository{
		policies: []models.Policy{},
		rules:    map[uuid.UUID][]models.PolicyRule{},
	}

	engine := NewEngine(ModeStrict, repo)
	err := engine.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh policies: %v", err)
	}

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{"engineering"},
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "prod-gateway",
	}

	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	req := EvaluationRequest{
		User:    user,
		Gateway: gateway,
		Resource: Resource{
			Network: *network,
		},
		Time: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected access to be denied in strict mode with no matching rules")
	}
}

func TestEvaluate_PermissiveMode(t *testing.T) {
	repo := &mockPolicyRepository{
		policies: []models.Policy{},
		rules:    map[uuid.UUID][]models.PolicyRule{},
	}

	engine := NewEngine(ModePermissive, repo)
	err := engine.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh policies: %v", err)
	}

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{"engineering"},
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "prod-gateway",
	}

	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	req := EvaluationRequest{
		User:    user,
		Gateway: gateway,
		Resource: Resource{
			Network: *network,
		},
		Time: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected access to be allowed in permissive mode with no matching deny rules")
	}
}

func TestEvaluate_DenyRule(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{
		Everyone: true,
	}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Networks: []string{"192.168.0.0/16"},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{
				ID:        policyID,
				Name:      "deny-internal",
				Priority:  5,
				IsEnabled: true,
			},
		},
		rules: map[uuid.UUID][]models.PolicyRule{
			policyID: {
				{
					ID:         uuid.New(),
					PolicyID:   policyID,
					Action:     "deny",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: []byte("{}"),
					Priority:   10,
				},
			},
		},
	}

	engine := NewEngine(ModePermissive, repo)
	err := engine.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh policies: %v", err)
	}

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{"engineering"},
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "prod-gateway",
	}

	_, network, _ := net.ParseCIDR("192.168.1.100/32")

	req := EvaluationRequest{
		User:    user,
		Gateway: gateway,
		Resource: Resource{
			Network: *network,
		},
		Time: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected access to be denied by explicit deny rule")
	}
}

func TestEvaluate_PolicyPriority(t *testing.T) {
	allowPolicyID := uuid.New()
	denyPolicyID := uuid.New()

	subject := models.PolicySubject{
		Everyone: true,
	}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Networks: []string{"10.0.0.0/8"},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{
				ID:        denyPolicyID,
				Name:      "deny-all",
				Priority:  100, // Lower priority (higher number)
				IsEnabled: true,
			},
			{
				ID:        allowPolicyID,
				Name:      "allow-all",
				Priority:  10, // Higher priority (lower number)
				IsEnabled: true,
			},
		},
		rules: map[uuid.UUID][]models.PolicyRule{
			allowPolicyID: {
				{
					ID:         uuid.New(),
					PolicyID:   allowPolicyID,
					Action:     "allow",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: []byte("{}"),
					Priority:   10,
				},
			},
			denyPolicyID: {
				{
					ID:         uuid.New(),
					PolicyID:   denyPolicyID,
					Action:     "deny",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: []byte("{}"),
					Priority:   10,
				},
			},
		},
	}

	engine := NewEngine(ModeStrict, repo)
	err := engine.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh policies: %v", err)
	}

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{},
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "prod-gateway",
	}

	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	req := EvaluationRequest{
		User:    user,
		Gateway: gateway,
		Resource: Resource{
			Network: *network,
		},
		Time: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Higher priority allow policy should win
	if !result.Allowed {
		t.Errorf("Expected access to be allowed by higher priority policy, got: %s", result.Reason)
	}

	if result.MatchedPolicy != "allow-all" {
		t.Errorf("Expected matched policy 'allow-all', got '%s'", result.MatchedPolicy)
	}
}
