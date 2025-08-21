package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRDSInstanceOperations(t *testing.T) {
	provider := &Provider{}
	err := provider.Initialize(context.Background(), map[string]interface{}{
		"region": "us-east-1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("GetRDSInstanceState_NonExistentInstance", func(t *testing.T) {
		instance := config.ResourceInstance{
			ID:   "aws:rds:instance.test-nonexistent",
			Kind: "aws:rds:instance",
			Name: "test-nonexistent-db",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"engine":                "mysql",
				"master_username":       "admin",
				"master_user_password":  "password123",
				"allocated_storage":     20,
			},
		}

		state, err := provider.GetCurrentState(ctx, instance)
		
		// If we get an auth error, skip the test
		if err != nil && (strings.Contains(err.Error(), "AuthFailure") || 
			strings.Contains(err.Error(), "InvalidClientTokenId") ||
			strings.Contains(err.Error(), "security token")) {
			t.Skip("Skipping integration test - AWS credentials not available or invalid")
		}
		
		assert.NoError(t, err, "Getting state of non-existent RDS instance should not error")
		assert.Nil(t, state, "Non-existent RDS instance should return nil state")
	})

	t.Run("ValidateRDSInstance_ValidConfig", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"engine":                "mysql",
				"master_username":       "admin",
				"master_user_password":  "password123",
				"allocated_storage":     20,
				"db_name":               "testdb",
			},
		}

		err := provider.ValidateResource(instance)
		assert.NoError(t, err)
	})

	t.Run("ValidateRDSInstance_MissingInstanceClass", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"engine":                "mysql",
				"master_username":       "admin",
				"master_user_password":  "password123",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db_instance_class is required")
	})

	t.Run("ValidateRDSInstance_MissingEngine", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"master_username":       "admin",
				"master_user_password":  "password123",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine is required")
	})

	t.Run("ValidateRDSInstance_InvalidEngine", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"engine":                "invalid-engine",
				"master_username":       "admin",
				"master_user_password":  "password123",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid engine type")
	})

	t.Run("ValidateRDSInstance_MissingMasterUsername", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"engine":                "mysql",
				"master_user_password":  "password123",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master_username is required")
	})

	t.Run("ValidateRDSInstance_MissingMasterPassword", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "test-db",
			Properties: map[string]interface{}{
				"db_instance_class": "db.t3.micro",
				"engine":            "mysql",
				"master_username":   "admin",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master_user_password is required")
	})

	t.Run("ValidateRDSInstance_EmptyName", func(t *testing.T) {
		instance := config.ResourceInstance{
			Kind: "aws:rds:instance",
			Name: "",
			Properties: map[string]interface{}{
				"db_instance_class":     "db.t3.micro",
				"engine":                "mysql",
				"master_username":       "admin",
				"master_user_password":  "password123",
			},
		}

		err := provider.ValidateResource(instance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
}
