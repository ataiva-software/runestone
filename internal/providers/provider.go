package providers

import (
	"context"

	"github.com/ataiva-software/runestone/internal/config"
)

// Provider defines the interface for cloud providers
type Provider interface {
	// Initialize sets up the provider with configuration
	Initialize(ctx context.Context, config map[string]interface{}) error

	// Create creates a new resource
	Create(ctx context.Context, instance config.ResourceInstance) error

	// Update updates an existing resource
	Update(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error

	// Delete deletes a resource
	Delete(ctx context.Context, instance config.ResourceInstance) error

	// GetCurrentState retrieves the current state of a resource
	GetCurrentState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error)

	// ValidateResource validates a resource configuration
	ValidateResource(instance config.ResourceInstance) error

	// GetSupportedResourceTypes returns the resource types supported by this provider
	GetSupportedResourceTypes() []string
}

// ResourceState represents the current state of a resource
type ResourceState struct {
	ID         string
	Kind       string
	Name       string
	Properties map[string]interface{}
	Exists     bool
	Metadata   map[string]interface{}
}

// DriftResult represents the result of drift detection
type DriftResult struct {
	HasDrift     bool
	Changes      []string                   // Human-readable list of changes
	Differences  map[string]DriftDifference
	CurrentState map[string]interface{}
	DesiredState map[string]interface{}
}

// DriftDifference represents a difference between current and desired state
type DriftDifference struct {
	Property     string
	CurrentValue interface{}
	DesiredValue interface{}
	DriftType    DriftType
}

// DriftType represents the type of drift
type DriftType string

const (
	DriftTypeAdded    DriftType = "added"
	DriftTypeRemoved  DriftType = "removed"
	DriftTypeModified DriftType = "modified"
)

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// NewRegistry creates a new provider registry (alias for NewProviderRegistry)
func NewRegistry() *ProviderRegistry {
	return NewProviderRegistry()
}

// Register registers a provider
func (r *ProviderRegistry) Register(name string, provider Provider) {
	r.providers[name] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	provider, exists := r.providers[name]
	return provider, exists
}

// GetAll returns all registered providers
func (r *ProviderRegistry) GetAll() map[string]Provider {
	result := make(map[string]Provider)
	for name, provider := range r.providers {
		result[name] = provider
	}
	return result
}
