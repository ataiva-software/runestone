package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/ataiva-software/runestone/internal/config"
)

// PolicyRule represents a single policy rule
type PolicyRule struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Severity    string                 `yaml:"severity"` // error, warning, info
	Condition   string                 `yaml:"condition"`
	Message     string                 `yaml:"message"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Rule         *PolicyRule
	ResourceID   string
	ResourceKind string
	Message      string
	Severity     string
	Metadata     map[string]interface{}
}

// PolicyEngine evaluates policies against resources
type PolicyEngine struct {
	rules []PolicyRule
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		rules: make([]PolicyRule, 0),
	}
}

// AddRule adds a policy rule to the engine
func (e *PolicyEngine) AddRule(rule PolicyRule) error {
	if rule.Name == "" {
		return fmt.Errorf("policy rule name cannot be empty")
	}
	
	if rule.Condition == "" {
		return fmt.Errorf("policy rule condition cannot be empty")
	}
	
	// Set default severity
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	
	// Validate severity
	validSeverities := []string{"error", "warning", "info"}
	valid := false
	for _, severity := range validSeverities {
		if rule.Severity == severity {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid severity: %s", rule.Severity)
	}
	
	e.rules = append(e.rules, rule)
	return nil
}

// EvaluateResource evaluates all policies against a resource
func (e *PolicyEngine) EvaluateResource(ctx context.Context, instance config.ResourceInstance) ([]PolicyViolation, error) {
	violations := make([]PolicyViolation, 0)
	
	for _, rule := range e.rules {
		violated, err := e.evaluateRule(ctx, rule, instance)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule %s: %w", rule.Name, err)
		}
		
		if violated {
			violation := PolicyViolation{
				Rule:         &rule,
				ResourceID:   instance.ID,
				ResourceKind: instance.Kind,
				Message:      rule.Message,
				Severity:     rule.Severity,
				Metadata:     rule.Metadata,
			}
			violations = append(violations, violation)
		}
	}
	
	return violations, nil
}

// evaluateRule evaluates a single rule against a resource
func (e *PolicyEngine) evaluateRule(ctx context.Context, rule PolicyRule, instance config.ResourceInstance) (bool, error) {
	// Simple condition evaluation - in a real implementation, this would use a proper expression evaluator
	condition := rule.Condition
	
	// Replace placeholders with actual values
	condition = strings.ReplaceAll(condition, "${resource.kind}", instance.Kind)
	condition = strings.ReplaceAll(condition, "${resource.name}", instance.Name)
	
	// Simple built-in policy checks
	switch {
	case strings.Contains(condition, "resource.kind == 'aws:s3:bucket' && !properties.versioning"):
		if instance.Kind == "aws:s3:bucket" {
			if versioning, ok := instance.Properties["versioning"].(bool); !ok || !versioning {
				return true, nil
			}
		}
		
	case strings.Contains(condition, "resource.kind == 'aws:ec2:instance' && properties.instance_type == 't3.large'"):
		if instance.Kind == "aws:ec2:instance" {
			if instanceType, ok := instance.Properties["instance_type"].(string); ok && instanceType == "t3.large" {
				return true, nil
			}
		}
		
	case strings.Contains(condition, "!tags.Environment"):
		if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok {
			if _, hasEnv := tags["Environment"]; !hasEnv {
				return true, nil
			}
		} else {
			return true, nil // No tags at all
		}
	}
	
	return false, nil
}

// GetViolationsByResource returns violations grouped by resource
func (e *PolicyEngine) GetViolationsByResource(violations []PolicyViolation) map[string][]PolicyViolation {
	result := make(map[string][]PolicyViolation)
	
	for _, violation := range violations {
		resourceID := violation.ResourceID
		result[resourceID] = append(result[resourceID], violation)
	}
	
	return result
}

// GetViolationsBySeverity returns violations grouped by severity
func (e *PolicyEngine) GetViolationsBySeverity(violations []PolicyViolation) map[string][]PolicyViolation {
	result := make(map[string][]PolicyViolation)
	
	for _, violation := range violations {
		severity := violation.Severity
		result[severity] = append(result[severity], violation)
	}
	
	return result
}

// HasErrors returns true if there are any error-level violations
func (e *PolicyEngine) HasErrors(violations []PolicyViolation) bool {
	for _, violation := range violations {
		if violation.Severity == "error" {
			return true
		}
	}
	return false
}

// LoadBuiltinPolicies loads common built-in policies
func (e *PolicyEngine) LoadBuiltinPolicies() error {
	builtinRules := []PolicyRule{
		{
			Name:        "s3-versioning-enabled",
			Description: "S3 buckets should have versioning enabled",
			Severity:    "warning",
			Condition:   "resource.kind == 'aws:s3:bucket' && !properties.versioning",
			Message:     "S3 bucket should have versioning enabled for data protection",
			Metadata: map[string]interface{}{
				"category": "security",
				"cis":      "2.1.1",
			},
		},
		{
			Name:        "no-large-instances-in-dev",
			Description: "Large instances should not be used in development environments",
			Severity:    "error",
			Condition:   "resource.kind == 'aws:ec2:instance' && properties.instance_type == 't3.large'",
			Message:     "Large instances are not allowed in development environments",
			Metadata: map[string]interface{}{
				"category": "cost-optimization",
			},
		},
		{
			Name:        "resources-must-have-environment-tag",
			Description: "All resources must have an Environment tag",
			Severity:    "warning",
			Condition:   "!tags.Environment",
			Message:     "Resource must have an Environment tag for proper resource management",
			Metadata: map[string]interface{}{
				"category": "governance",
			},
		},
	}
	
	for _, rule := range builtinRules {
		if err := e.AddRule(rule); err != nil {
			return fmt.Errorf("failed to add builtin rule %s: %w", rule.Name, err)
		}
	}
	
	return nil
}
