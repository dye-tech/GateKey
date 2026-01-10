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

func TestEvaluate_DisabledPolicy(t *testing.T) {
	policyID := uuid.New()

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
				ID:        policyID,
				Name:      "disabled-allow",
				Priority:  10,
				IsEnabled: false, // Disabled
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
		ID:    uuid.New(),
		Email: "alice@example.com",
	}

	gateway := &models.Gateway{
		ID:   uuid.New(),
		Name: "gateway",
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

	// Disabled policy should be ignored, strict mode denies by default
	if result.Allowed {
		t.Error("Expected access to be denied when policy is disabled")
	}
}

func TestEvaluate_SpecificGateway(t *testing.T) {
	policyID := uuid.New()
	targetGatewayID := uuid.New()

	subject := models.PolicySubject{
		Everyone: true,
	}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Gateways: []string{targetGatewayID.String()},
		Networks: []string{"10.0.0.0/8"},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{
				ID:        policyID,
				Name:      "gateway-specific",
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
	_ = engine.Refresh(context.Background())

	user := &models.User{ID: uuid.New(), Email: "test@example.com"}

	// Test with matching gateway
	matchingGateway := &models.Gateway{ID: targetGatewayID, Name: "matching"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  matchingGateway,
		Resource: Resource{Network: *network},
		Time:     time.Now(),
	})

	if !result.Allowed {
		t.Error("Expected access allowed for matching gateway")
	}

	// Test with non-matching gateway
	otherGateway := &models.Gateway{ID: uuid.New(), Name: "other"}
	result2, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  otherGateway,
		Resource: Resource{Network: *network},
		Time:     time.Now(),
	})

	if result2.Allowed {
		t.Error("Expected access denied for non-matching gateway")
	}
}

func TestEvaluate_PortMatching(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{Everyone: true}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Networks: []string{"10.0.0.0/8"},
		Ports: []models.PolicyPort{
			{Protocol: "tcp", Port: 443},
			{Protocol: "tcp", FromPort: 8000, ToPort: 9000},
		},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "port-policy", Priority: 10, IsEnabled: true},
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
	_ = engine.Refresh(context.Background())

	user := &models.User{ID: uuid.New(), Email: "test@example.com"}
	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	tests := []struct {
		name     string
		port     int
		protocol string
		expected bool
	}{
		{"port 443 tcp", 443, "tcp", true},
		{"port 443 udp", 443, "udp", false},
		{"port 8500 in range", 8500, "tcp", true},
		{"port 80 not in policy", 80, "tcp", false},
		{"port 9001 outside range", 9001, "tcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
				User:    user,
				Gateway: gateway,
				Resource: Resource{
					Network:  *network,
					Port:     tt.port,
					Protocol: tt.protocol,
				},
				Time: time.Now(),
			})

			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v for %s, got %v", tt.expected, tt.name, result.Allowed)
			}
		})
	}
}

func TestEvaluate_SourceIPCondition(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{Everyone: true}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{Networks: []string{"10.0.0.0/8"}}
	resourceJSON, _ := json.Marshal(resource)

	conditions := models.PolicyCondition{
		SourceIPs: []string{"192.168.1.0/24", "10.10.10.10"},
	}
	conditionsJSON, _ := json.Marshal(conditions)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "source-ip-policy", Priority: 10, IsEnabled: true},
		},
		rules: map[uuid.UUID][]models.PolicyRule{
			policyID: {
				{
					ID:         uuid.New(),
					PolicyID:   policyID,
					Action:     "allow",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: conditionsJSON,
					Priority:   10,
				},
			},
		},
	}

	engine := NewEngine(ModeStrict, repo)
	_ = engine.Refresh(context.Background())

	user := &models.User{ID: uuid.New(), Email: "test@example.com"}
	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	tests := []struct {
		name     string
		sourceIP string
		expected bool
	}{
		{"IP in 192.168.1.0/24", "192.168.1.50", true},
		{"exact IP match", "10.10.10.10", true},
		{"IP outside allowed ranges", "172.16.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
				User:     user,
				Gateway:  gateway,
				Resource: Resource{Network: *network},
				SourceIP: net.ParseIP(tt.sourceIP),
				Time:     time.Now(),
			})

			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v for sourceIP %s, got %v", tt.expected, tt.sourceIP, result.Allowed)
			}
		})
	}
}

func TestEvaluate_TimeWindowCondition(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{Everyone: true}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{Networks: []string{"10.0.0.0/8"}}
	resourceJSON, _ := json.Marshal(resource)

	// Allow access Monday-Friday 09:00-17:00 UTC
	conditions := models.PolicyCondition{
		TimeWindows: []models.TimeWindow{
			{
				Days:      []string{"mon", "tue", "wed", "thu", "fri"},
				StartTime: "09:00",
				EndTime:   "17:00",
				Timezone:  "UTC",
			},
		},
	}
	conditionsJSON, _ := json.Marshal(conditions)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "time-window-policy", Priority: 10, IsEnabled: true},
		},
		rules: map[uuid.UUID][]models.PolicyRule{
			policyID: {
				{
					ID:         uuid.New(),
					PolicyID:   policyID,
					Action:     "allow",
					Subject:    subjectJSON,
					Resource:   resourceJSON,
					Conditions: conditionsJSON,
					Priority:   10,
				},
			},
		},
	}

	engine := NewEngine(ModeStrict, repo)
	_ = engine.Refresh(context.Background())

	user := &models.User{ID: uuid.New(), Email: "test@example.com"}
	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	// Test during business hours on a weekday (Wednesday at 10:00 UTC)
	businessTime := time.Date(2024, 1, 17, 10, 0, 0, 0, time.UTC) // Wednesday
	result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     businessTime,
	})

	if !result.Allowed {
		t.Error("Expected access allowed during business hours on weekday")
	}

	// Test outside business hours (Wednesday at 20:00 UTC)
	afterHours := time.Date(2024, 1, 17, 20, 0, 0, 0, time.UTC)
	result2, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     afterHours,
	})

	if result2.Allowed {
		t.Error("Expected access denied outside business hours")
	}

	// Test on weekend (Saturday at 10:00 UTC)
	weekend := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC) // Saturday
	result3, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     weekend,
	})

	if result3.Allowed {
		t.Error("Expected access denied on weekend")
	}
}

func TestEvaluate_UserEmailMatch(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{
		Users: []string{"alice@example.com", "bob@example.com"},
	}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{Networks: []string{"10.0.0.0/8"}}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "user-specific", Priority: 10, IsEnabled: true},
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
	_ = engine.Refresh(context.Background())

	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	// Test matching user
	alice := &models.User{ID: uuid.New(), Email: "alice@example.com"}
	result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     alice,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     time.Now(),
	})

	if !result.Allowed {
		t.Error("Expected access allowed for matching user email")
	}

	// Test non-matching user
	charlie := &models.User{ID: uuid.New(), Email: "charlie@example.com"}
	result2, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     charlie,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     time.Now(),
	})

	if result2.Allowed {
		t.Error("Expected access denied for non-matching user email")
	}
}

func TestGetAllowedNetworks(t *testing.T) {
	policyID := uuid.New()
	gatewayID := uuid.New()

	subject := models.PolicySubject{Groups: []string{"engineering"}}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{
		Gateways: []string{gatewayID.String()},
		Networks: []string{"10.0.0.0/8", "192.168.0.0/16"},
	}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "engineering-networks", Priority: 10, IsEnabled: true},
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
	_ = engine.Refresh(context.Background())

	user := &models.User{
		ID:     uuid.New(),
		Email:  "alice@example.com",
		Groups: []string{"engineering"},
	}

	gateway := &models.Gateway{ID: gatewayID, Name: "gateway"}

	networks, err := engine.GetAllowedNetworks(context.Background(), user, gateway)
	if err != nil {
		t.Fatalf("GetAllowedNetworks failed: %v", err)
	}

	if len(networks) != 2 {
		t.Errorf("Expected 2 networks, got %d", len(networks))
	}
}

func TestGetAllowedNetworks_NoMatchingRules(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{Groups: []string{"admin"}}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{Networks: []string{"10.0.0.0/8"}}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "admin-only", Priority: 10, IsEnabled: true},
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
	_ = engine.Refresh(context.Background())

	user := &models.User{
		ID:     uuid.New(),
		Email:  "user@example.com",
		Groups: []string{"users"}, // Not admin
	}

	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}

	networks, err := engine.GetAllowedNetworks(context.Background(), user, gateway)
	if err != nil {
		t.Fatalf("GetAllowedNetworks failed: %v", err)
	}

	if len(networks) != 0 {
		t.Errorf("Expected 0 networks for non-matching user, got %d", len(networks))
	}
}

func TestNewEngine(t *testing.T) {
	repo := &mockPolicyRepository{}

	engine := NewEngine(ModeStrict, repo)
	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	engine2 := NewEngine(ModePermissive, repo)
	if engine2 == nil {
		t.Fatal("Expected non-nil engine for permissive mode")
	}
}

func TestEvaluationMode_Constants(t *testing.T) {
	if ModeStrict != "strict" {
		t.Errorf("Expected ModeStrict 'strict', got '%s'", ModeStrict)
	}
	if ModePermissive != "permissive" {
		t.Errorf("Expected ModePermissive 'permissive', got '%s'", ModePermissive)
	}
}

func TestEvaluationResult_AppliedRules(t *testing.T) {
	policyID := uuid.New()

	subject := models.PolicySubject{Everyone: true}
	subjectJSON, _ := json.Marshal(subject)

	resource := models.PolicyResource{Networks: []string{"10.0.0.0/8"}}
	resourceJSON, _ := json.Marshal(resource)

	repo := &mockPolicyRepository{
		policies: []models.Policy{
			{ID: policyID, Name: "test-policy", Priority: 10, IsEnabled: true},
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
	_ = engine.Refresh(context.Background())

	user := &models.User{ID: uuid.New(), Email: "test@example.com"}
	gateway := &models.Gateway{ID: uuid.New(), Name: "gateway"}
	_, network, _ := net.ParseCIDR("10.0.0.5/32")

	result, _ := engine.Evaluate(context.Background(), EvaluationRequest{
		User:     user,
		Gateway:  gateway,
		Resource: Resource{Network: *network},
		Time:     time.Now(),
	})

	if len(result.AppliedRules) == 0 {
		t.Error("Expected AppliedRules to be populated")
	}

	if result.AppliedRules[0].PolicyName != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got '%s'", result.AppliedRules[0].PolicyName)
	}

	if !result.AppliedRules[0].Matched {
		t.Error("Expected first rule to be matched")
	}
}
