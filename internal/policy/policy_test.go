package policy

import (
	"context"
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyEngine(t *testing.T) {
	engine := NewPolicyEngine()
	
	t.Run("AddRule", func(t *testing.T) {
		rule := PolicyRule{
			Name:        "test-rule",
			Description: "Test rule",
			Severity:    "warning",
			Condition:   "resource.kind == 'aws:s3:bucket'",
			Message:     "Test message",
		}
		
		err := engine.AddRule(rule)
		assert.NoError(t, err)
		assert.Len(t, engine.rules, 1)
	})
	
	t.Run("AddRule_EmptyName", func(t *testing.T) {
		rule := PolicyRule{
			Condition: "resource.kind == 'aws:s3:bucket'",
		}
		
		err := engine.AddRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
	
	t.Run("AddRule_EmptyCondition", func(t *testing.T) {
		rule := PolicyRule{
			Name: "test-rule",
		}
		
		err := engine.AddRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "condition cannot be empty")
	})
	
	t.Run("AddRule_InvalidSeverity", func(t *testing.T) {
		rule := PolicyRule{
			Name:      "test-rule",
			Severity:  "invalid",
			Condition: "resource.kind == 'aws:s3:bucket'",
		}
		
		err := engine.AddRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid severity")
	})
	
	t.Run("AddRule_DefaultSeverity", func(t *testing.T) {
		rule := PolicyRule{
			Name:      "test-rule-2",
			Condition: "resource.kind == 'aws:s3:bucket'",
		}
		
		err := engine.AddRule(rule)
		assert.NoError(t, err)
		assert.Equal(t, "warning", engine.rules[len(engine.rules)-1].Severity)
	})
}

func TestPolicyEngine_EvaluateResource(t *testing.T) {
	engine := NewPolicyEngine()
	ctx := context.Background()
	
	// Add test rules
	rules := []PolicyRule{
		{
			Name:      "s3-versioning-required",
			Severity:  "error",
			Condition: "resource.kind == 'aws:s3:bucket' && !properties.versioning",
			Message:   "S3 bucket must have versioning enabled",
		},
		{
			Name:      "environment-tag-required",
			Severity:  "warning",
			Condition: "!tags.Environment",
			Message:   "Resource must have Environment tag",
		},
	}
	
	for _, rule := range rules {
		err := engine.AddRule(rule)
		require.NoError(t, err)
	}
	
	t.Run("S3Bucket_NoVersioning_Violation", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:s3:bucket.test-bucket",
			Kind: "aws:s3:bucket",
			Name: "test-bucket",
			Properties: map[string]interface{}{
				"versioning": false,
			},
		}
		
		violations, err := engine.EvaluateResource(ctx, instance)
		require.NoError(t, err)
		assert.Len(t, violations, 2) // versioning + no environment tag
		
		// Check versioning violation
		assert.Equal(t, "s3-versioning-required", violations[0].Rule.Name)
		assert.Equal(t, "error", violations[0].Severity)
	})
	
	t.Run("S3Bucket_WithVersioning_NoViolation", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:s3:bucket.test-bucket",
			Kind: "aws:s3:bucket",
			Name: "test-bucket",
			Properties: map[string]interface{}{
				"versioning": true,
				"tags": map[string]interface{}{
					"Environment": "dev",
				},
			},
		}
		
		violations, err := engine.EvaluateResource(ctx, instance)
		require.NoError(t, err)
		assert.Len(t, violations, 0)
	})
	
	t.Run("EC2Instance_NoEnvironmentTag_Violation", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:ec2:instance.test-instance",
			Kind: "aws:ec2:instance",
			Name: "test-instance",
			Properties: map[string]interface{}{
				"instance_type": "t3.micro",
				"tags": map[string]interface{}{
					"Name": "test-instance",
				},
			},
		}
		
		violations, err := engine.EvaluateResource(ctx, instance)
		require.NoError(t, err)
		assert.Len(t, violations, 1)
		assert.Equal(t, "environment-tag-required", violations[0].Rule.Name)
		assert.Equal(t, "warning", violations[0].Severity)
	})
}

func TestPolicyEngine_BuiltinPolicies(t *testing.T) {
	engine := NewPolicyEngine()
	
	err := engine.LoadBuiltinPolicies()
	assert.NoError(t, err)
	assert.Greater(t, len(engine.rules), 0)
	
	// Test that builtin policies work
	ctx := context.Background()
	
	t.Run("BuiltinPolicy_S3Versioning", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:s3:bucket.test-bucket",
			Kind: "aws:s3:bucket",
			Name: "test-bucket",
			Properties: map[string]interface{}{
				"versioning": false,
			},
		}
		
		violations, err := engine.EvaluateResource(ctx, instance)
		require.NoError(t, err)
		
		// Should have violations for versioning and environment tag
		assert.Greater(t, len(violations), 0)
		
		// Check that we have the S3 versioning violation
		hasVersioningViolation := false
		for _, violation := range violations {
			if violation.Rule.Name == "s3-versioning-enabled" {
				hasVersioningViolation = true
				break
			}
		}
		assert.True(t, hasVersioningViolation)
	})
}

func TestPolicyEngine_Utilities(t *testing.T) {
	violations := []PolicyViolation{
		{
			ResourceID: "resource1",
			Severity:   "error",
			Rule:       &PolicyRule{Name: "rule1"},
		},
		{
			ResourceID: "resource1",
			Severity:   "warning",
			Rule:       &PolicyRule{Name: "rule2"},
		},
		{
			ResourceID: "resource2",
			Severity:   "error",
			Rule:       &PolicyRule{Name: "rule3"},
		},
	}
	
	engine := NewPolicyEngine()
	
	t.Run("GetViolationsByResource", func(t *testing.T) {
		byResource := engine.GetViolationsByResource(violations)
		
		assert.Len(t, byResource, 2)
		assert.Len(t, byResource["resource1"], 2)
		assert.Len(t, byResource["resource2"], 1)
	})
	
	t.Run("GetViolationsBySeverity", func(t *testing.T) {
		bySeverity := engine.GetViolationsBySeverity(violations)
		
		assert.Len(t, bySeverity["error"], 2)
		assert.Len(t, bySeverity["warning"], 1)
	})
	
	t.Run("HasErrors", func(t *testing.T) {
		assert.True(t, engine.HasErrors(violations))
		
		warningOnly := []PolicyViolation{
			{Severity: "warning", Rule: &PolicyRule{Name: "rule1"}},
		}
		assert.False(t, engine.HasErrors(warningOnly))
	})
}
