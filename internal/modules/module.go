package modules

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ataiva-software/runestone/internal/config"
)

// Module represents a reusable infrastructure module
type Module struct {
	Name    string                 `yaml:"name"`
	Source  string                 `yaml:"source"`
	Version string                 `yaml:"version"`
	Inputs  map[string]interface{} `yaml:"inputs"`
}

// ModuleRegistry manages available modules
type ModuleRegistry struct {
	modules map[string]*Module
}

// NewRegistry creates a new module registry (alias for NewModuleRegistry)
func NewRegistry() *ModuleRegistry {
	return NewModuleRegistry()
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]*Module),
	}
}

// RegisterModule registers a module in the registry
func (r *ModuleRegistry) RegisterModule(module *Module) error {
	if module.Name == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	
	r.modules[module.Name] = module
	return nil
}

// GetModule retrieves a module by name
func (r *ModuleRegistry) GetModule(name string) (*Module, bool) {
	module, exists := r.modules[name]
	return module, exists
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("module name is required")
	}
	
	if m.Source == "" {
		return fmt.Errorf("module source is required")
	}
	
	return nil
}

// Load loads the module from its source
func (m *Module) Load() error {
	// For now, we support local file-based modules
	if strings.HasPrefix(m.Source, "./") || strings.HasPrefix(m.Source, "/") {
		return m.loadLocalModule()
	}
	
	return fmt.Errorf("unsupported module source: %s", m.Source)
}

// loadLocalModule loads a module from the local filesystem
func (m *Module) loadLocalModule() error {
	// Check if the source path exists
	if _, err := os.Stat(m.Source); os.IsNotExist(err) {
		return fmt.Errorf("module source path does not exist: %s", m.Source)
	}
	
	// For now, just validate that it's a directory
	info, err := os.Stat(m.Source)
	if err != nil {
		return fmt.Errorf("failed to stat module source: %w", err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("module source must be a directory: %s", m.Source)
	}
	
	return nil
}

// LoadModule loads a module from source
func (r *ModuleRegistry) LoadModule(ctx context.Context, name, source, version string) (*Module, error) {
	// For now, we'll implement local file-based modules
	if !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "/") {
		return nil, fmt.Errorf("unsupported module source: %s", source)
	}

	// Check if source exists
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil, fmt.Errorf("module source not found: %s", source)
	}

	module := &Module{
		Name:    name,
		Source:  source,
		Version: version,
		Inputs:  make(map[string]interface{}),
	}

	return module, nil
}

// ExpandModule expands a module into resource instances
func (r *ModuleRegistry) ExpandModule(ctx context.Context, name string, inputs map[string]interface{}) ([]config.ResourceInstance, error) {
	_, exists := r.GetModule(name)
	if !exists {
		return nil, fmt.Errorf("module not found: %s", name)
	}

	// For now, return empty slice as modules are not fully implemented
	// In the future, this would parse the module's configuration and expand it
	return []config.ResourceInstance{}, nil
}
