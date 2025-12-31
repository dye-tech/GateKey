// Package policy provides policy evaluation for access control.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/models"
)

// Engine evaluates access control policies.
type Engine struct {
	mode       EvaluationMode
	policies   []PolicyWithRules
	policyRepo PolicyRepository
}

// PolicyRepository defines the interface for policy storage.
type PolicyRepository interface {
	List(ctx context.Context) ([]models.Policy, error)
	GetRules(ctx context.Context, policyID uuid.UUID) ([]models.PolicyRule, error)
}

// EvaluationMode determines how policies are evaluated.
type EvaluationMode string

const (
	// ModeStrict denies access if no explicit allow rule matches.
	ModeStrict EvaluationMode = "strict"
	// ModePermissive allows access if no explicit deny rule matches.
	ModePermissive EvaluationMode = "permissive"
)

// PolicyWithRules combines a policy with its rules.
type PolicyWithRules struct {
	Policy models.Policy
	Rules  []models.PolicyRule
}

// NewEngine creates a new policy engine.
func NewEngine(mode EvaluationMode, repo PolicyRepository) *Engine {
	return &Engine{
		mode:       mode,
		policyRepo: repo,
	}
}

// Refresh reloads policies from the repository.
func (e *Engine) Refresh(ctx context.Context) error {
	policies, err := e.policyRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}

	var withRules []PolicyWithRules
	for _, p := range policies {
		if !p.IsEnabled {
			continue
		}

		rules, err := e.policyRepo.GetRules(ctx, p.ID)
		if err != nil {
			return fmt.Errorf("failed to get rules for policy %s: %w", p.Name, err)
		}

		withRules = append(withRules, PolicyWithRules{
			Policy: p,
			Rules:  rules,
		})
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(withRules, func(i, j int) bool {
		return withRules[i].Policy.Priority < withRules[j].Policy.Priority
	})

	e.policies = withRules
	return nil
}

// EvaluationRequest contains the context for policy evaluation.
type EvaluationRequest struct {
	User     *models.User
	Gateway  *models.Gateway
	Resource Resource
	SourceIP net.IP
	Time     time.Time
}

// Resource represents a network resource being accessed.
type Resource struct {
	Network  net.IPNet
	Port     int
	Protocol string // tcp, udp, or empty for any
	Service  string // Optional service name
}

// EvaluationResult contains the result of policy evaluation.
type EvaluationResult struct {
	Allowed       bool
	MatchedPolicy string
	MatchedRule   string
	Reason        string
	AppliedRules  []AppliedRule
}

// AppliedRule represents a rule that was evaluated.
type AppliedRule struct {
	PolicyName string
	RuleID     string
	Action     string
	Matched    bool
}

// Evaluate evaluates policies for a given request.
func (e *Engine) Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResult, error) {
	result := &EvaluationResult{
		AppliedRules: []AppliedRule{},
	}

	for _, p := range e.policies {
		for _, rule := range p.Rules {
			matched, err := e.matchRule(rule, req)
			if err != nil {
				continue // Skip rules that can't be evaluated
			}

			result.AppliedRules = append(result.AppliedRules, AppliedRule{
				PolicyName: p.Policy.Name,
				RuleID:     rule.ID.String(),
				Action:     rule.Action,
				Matched:    matched,
			})

			if matched {
				result.MatchedPolicy = p.Policy.Name
				result.MatchedRule = rule.ID.String()

				if rule.Action == "allow" {
					result.Allowed = true
					result.Reason = fmt.Sprintf("Allowed by policy '%s'", p.Policy.Name)
					return result, nil
				} else if rule.Action == "deny" {
					result.Allowed = false
					result.Reason = fmt.Sprintf("Denied by policy '%s'", p.Policy.Name)
					return result, nil
				}
			}
		}
	}

	// No explicit match - use default based on mode
	if e.mode == ModePermissive {
		result.Allowed = true
		result.Reason = "No matching deny rule (permissive mode)"
	} else {
		result.Allowed = false
		result.Reason = "No matching allow rule (strict mode)"
	}

	return result, nil
}

// matchRule checks if a rule matches the request.
func (e *Engine) matchRule(rule models.PolicyRule, req EvaluationRequest) (bool, error) {
	// Parse subject
	var subject models.PolicySubject
	if err := json.Unmarshal(rule.Subject, &subject); err != nil {
		return false, err
	}

	// Check subject match
	if !e.matchSubject(subject, req.User) {
		return false, nil
	}

	// Parse resource
	var resource models.PolicyResource
	if err := json.Unmarshal(rule.Resource, &resource); err != nil {
		return false, err
	}

	// Check resource match
	if !e.matchResource(resource, req) {
		return false, nil
	}

	// Parse and check conditions
	var conditions models.PolicyCondition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return false, err
	}

	if !e.matchConditions(conditions, req) {
		return false, nil
	}

	return true, nil
}

// matchSubject checks if the user matches the subject criteria.
func (e *Engine) matchSubject(subject models.PolicySubject, user *models.User) bool {
	if subject.Everyone {
		return true
	}

	// Check user match
	for _, u := range subject.Users {
		if strings.EqualFold(u, user.Email) || u == user.ID.String() {
			return true
		}
	}

	// Check group match
	for _, g := range subject.Groups {
		for _, ug := range user.Groups {
			if strings.EqualFold(g, ug) {
				return true
			}
		}
	}

	return false
}

// matchResource checks if the request resource matches the policy resource.
func (e *Engine) matchResource(resource models.PolicyResource, req EvaluationRequest) bool {
	// If no resource constraints, match all
	if len(resource.Gateways) == 0 && len(resource.Networks) == 0 &&
		len(resource.Ports) == 0 && len(resource.Services) == 0 {
		return true
	}

	// Check gateway match
	if len(resource.Gateways) > 0 {
		matched := false
		for _, g := range resource.Gateways {
			if g == req.Gateway.Name || g == req.Gateway.ID.String() {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check network match
	if len(resource.Networks) > 0 {
		matched := false
		for _, n := range resource.Networks {
			_, network, err := net.ParseCIDR(n)
			if err != nil {
				continue
			}
			if network.Contains(req.Resource.Network.IP) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check port match
	if len(resource.Ports) > 0 && req.Resource.Port > 0 {
		matched := false
		for _, p := range resource.Ports {
			if e.matchPort(p, req.Resource.Port, req.Resource.Protocol) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check service match
	if len(resource.Services) > 0 && req.Resource.Service != "" {
		matched := false
		for _, s := range resource.Services {
			if strings.EqualFold(s, req.Resource.Service) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// matchPort checks if a port matches the policy port.
func (e *Engine) matchPort(p models.PolicyPort, port int, protocol string) bool {
	// Check protocol
	if p.Protocol != "" && p.Protocol != "both" {
		if !strings.EqualFold(p.Protocol, protocol) {
			return false
		}
	}

	// Check port
	if p.Port > 0 {
		return port == p.Port
	}

	// Check port range
	if p.FromPort > 0 && p.ToPort > 0 {
		return port >= p.FromPort && port <= p.ToPort
	}

	return true
}

// matchConditions checks if the request matches additional conditions.
func (e *Engine) matchConditions(conditions models.PolicyCondition, req EvaluationRequest) bool {
	// Check time windows
	if len(conditions.TimeWindows) > 0 {
		matched := false
		for _, tw := range conditions.TimeWindows {
			if e.matchTimeWindow(tw, req.Time) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check source IPs
	if len(conditions.SourceIPs) > 0 && req.SourceIP != nil {
		matched := false
		for _, cidr := range conditions.SourceIPs {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				// Try as single IP
				if ip := net.ParseIP(cidr); ip != nil && ip.Equal(req.SourceIP) {
					matched = true
					break
				}
				continue
			}
			if network.Contains(req.SourceIP) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// matchTimeWindow checks if the time falls within a time window.
func (e *Engine) matchTimeWindow(tw models.TimeWindow, t time.Time) bool {
	// Load timezone
	loc, err := time.LoadLocation(tw.Timezone)
	if err != nil {
		loc = time.UTC
	}

	localTime := t.In(loc)

	// Check day of week
	if len(tw.Days) > 0 {
		dayName := strings.ToLower(localTime.Weekday().String()[:3])
		matched := false
		for _, d := range tw.Days {
			if strings.EqualFold(d, dayName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check time range
	if tw.StartTime != "" && tw.EndTime != "" {
		currentTime := localTime.Format("15:04")
		if currentTime < tw.StartTime || currentTime > tw.EndTime {
			return false
		}
	}

	return true
}

// GetAllowedNetworks returns all networks a user is allowed to access through a gateway.
func (e *Engine) GetAllowedNetworks(ctx context.Context, user *models.User, gateway *models.Gateway) ([]net.IPNet, error) {
	var networks []net.IPNet

	for _, p := range e.policies {
		for _, rule := range p.Rules {
			if rule.Action != "allow" {
				continue
			}

			// Parse subject
			var subject models.PolicySubject
			if err := json.Unmarshal(rule.Subject, &subject); err != nil {
				continue
			}

			if !e.matchSubject(subject, user) {
				continue
			}

			// Parse resource
			var resource models.PolicyResource
			if err := json.Unmarshal(rule.Resource, &resource); err != nil {
				continue
			}

			// Check gateway match
			if len(resource.Gateways) > 0 {
				matched := false
				for _, g := range resource.Gateways {
					if g == gateway.Name || g == gateway.ID.String() {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			// Add networks from this rule
			for _, n := range resource.Networks {
				_, network, err := net.ParseCIDR(n)
				if err != nil {
					continue
				}
				networks = append(networks, *network)
			}
		}
	}

	return networks, nil
}
