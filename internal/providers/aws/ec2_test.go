package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEC2InstanceStateRetrieval(t *testing.T) {
	// Skip integration tests if AWS credentials are not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider := NewProvider()
	ctx := context.Background()

	// Initialize provider with test configuration
	providerConfig := map[string]interface{}{
		"region":  "us-east-1",
		"profile": "default",
	}

	err := provider.Initialize(ctx, providerConfig)
	require.NoError(t, err, "Provider initialization should succeed")

	t.Run("GetEC2InstanceState_NonExistentInstance", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:ec2:instance.test-nonexistent",
			Kind: "aws:ec2:instance",
			Name: "test-nonexistent-instance",
			Properties: map[string]interface{}{
				"instance_type": "t3.micro",
				"ami":           "ami-0abcdef1234567890",
				"tags": map[string]interface{}{
					"Name": "test-nonexistent-instance",
				},
			},
		}

		state, err := provider.GetCurrentState(ctx, instance)
		
		// If we get an auth error, skip the test
		if err != nil && strings.Contains(err.Error(), "AuthFailure") {
			t.Skip("Skipping integration test - AWS credentials not available or invalid")
		}
		
		assert.NoError(t, err, "Getting state of non-existent instance should not error")
		assert.Nil(t, state, "Non-existent instance should return nil state")
	})

	t.Run("ValidateEC2Instance_ValidConfig", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:ec2:instance.test-valid",
			Kind: "aws:ec2:instance",
			Name: "test-valid-instance",
			Properties: map[string]interface{}{
				"instance_type": "t3.micro",
				"ami":           "ami-0abcdef1234567890",
				"tags": map[string]interface{}{
					"Name": "test-valid-instance",
				},
			},
		}

		err := provider.ValidateResource(instance)
		assert.NoError(t, err, "Valid EC2 instance should pass validation")
	})

	t.Run("ValidateEC2Instance_MissingInstanceType", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:ec2:instance.test-invalid",
			Kind: "aws:ec2:instance",
			Name: "test-invalid-instance",
			Properties: map[string]interface{}{
				"ami": "ami-0abcdef1234567890",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err, "EC2 instance without instance_type should fail validation")
		assert.Contains(t, err.Error(), "instance_type is required")
	})

	t.Run("ValidateEC2Instance_MissingAMI", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:ec2:instance.test-invalid",
			Kind: "aws:ec2:instance",
			Name: "test-invalid-instance",
			Properties: map[string]interface{}{
				"instance_type": "t3.micro",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err, "EC2 instance without AMI should fail validation")
		assert.Contains(t, err.Error(), "ami is required")
	})
}
