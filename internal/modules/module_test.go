package modules

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleRegistry(t *testing.T) {
	registry := NewModuleRegistry()
	
	t.Run("RegisterModule", func(t *testing.T) {
		module := &Module{
			Name:    "vpc",
			Source:  "./modules/vpc",
			Version: "1.0.0",
			Inputs: map[string]interface{}{
				"cidr_block": "10.0.0.0/16",
			},
		}
		
		err := registry.RegisterModule(module)
		assert.NoError(t, err)
		
		retrieved, exists := registry.GetModule("vpc")
		assert.True(t, exists)
		assert.Equal(t, module.Name, retrieved.Name)
		assert.Equal(t, module.Source, retrieved.Source)
		assert.Equal(t, module.Version, retrieved.Version)
	})
	
	t.Run("RegisterModule_EmptyName", func(t *testing.T) {
		module := &Module{
			Name:   "",
			Source: "./modules/vpc",
		}
		
		err := registry.RegisterModule(module)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
	
	t.Run("GetModule_NotFound", func(t *testing.T) {
		_, exists := registry.GetModule("nonexistent")
		assert.False(t, exists)
	})
}

func TestModule_Validation(t *testing.T) {
	tests := []struct {
		name    string
		module  *Module
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid module",
			module: &Module{
				Name:    "vpc",
				Source:  "./modules/vpc",
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			module: &Module{
				Source:  "./modules/vpc",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing source",
			module: &Module{
				Name:    "vpc",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "source is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.module.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModule_LoadModule(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := context.Background()
	
	t.Run("LoadLocalModule", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		
		module, err := registry.LoadModule(ctx, "test-vpc", tempDir, "1.0.0")
		require.NoError(t, err)
		
		assert.Equal(t, "test-vpc", module.Name)
		assert.Equal(t, tempDir, module.Source)
		assert.Equal(t, "1.0.0", module.Version)
		assert.NotNil(t, module.Inputs)
	})
	
	t.Run("UnsupportedSource", func(t *testing.T) {
		_, err := registry.LoadModule(ctx, "test", "https://example.com/module", "1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported module source")
	})
}

func TestModule_ExpandModule(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := context.Background()
	
	// Register a test module
	module := &Module{
		Name:    "vpc",
		Source:  "./modules/vpc",
		Version: "1.0.0",
	}
	err := registry.RegisterModule(module)
	require.NoError(t, err)
	
	t.Run("ExpandExistingModule", func(t *testing.T) {
		inputs := map[string]interface{}{
			"cidr_block": "10.0.0.0/16",
		}
		
		instances, err := registry.ExpandModule(ctx, "vpc", inputs)
		assert.NoError(t, err)
		assert.NotNil(t, instances)
		// For now, we expect empty slice since we haven't implemented full expansion
		assert.Len(t, instances, 0)
	})
	
	t.Run("ExpandNonexistentModule", func(t *testing.T) {
		_, err := registry.ExpandModule(ctx, "nonexistent", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "module not found")
	})
}
