package config

import "time"

// Config represents the main Runestone configuration
type Config struct {
	Project   string                 `yaml:"project"`
	Environment string               `yaml:"environment"`
	Variables map[string]interface{} `yaml:"variables,omitempty"`
	Providers map[string]Provider    `yaml:"providers"`
	Modules   map[string]Module      `yaml:"modules,omitempty"`
	Resources []Resource             `yaml:"resources"`
}

// Provider represents a cloud provider configuration
type Provider struct {
	Region  string `yaml:"region,omitempty"`
	Profile string `yaml:"profile,omitempty"`
	// Additional provider-specific fields can be added here
}

// Module represents a reusable module
type Module struct {
	Source  string                 `yaml:"source"`
	Version string                 `yaml:"version"`
	Inputs  map[string]interface{} `yaml:"inputs,omitempty"`
}

// Resource represents an infrastructure resource
type Resource struct {
	Kind        string                 `yaml:"kind"`
	Name        string                 `yaml:"name"`
	Count       interface{}            `yaml:"count,omitempty"`       // Can be int or expression
	ForEach     interface{}            `yaml:"for_each,omitempty"`    // Can be array or expression
	Properties  map[string]interface{} `yaml:"properties,omitempty"`
	DriftPolicy *DriftPolicy           `yaml:"driftPolicy,omitempty"`
	DependsOn   []string               `yaml:"depends_on,omitempty"`
}

// DriftPolicy defines how to handle drift for a resource
type DriftPolicy struct {
	AutoHeal   bool `yaml:"autoHeal"`
	NotifyOnly bool `yaml:"notifyOnly"`
}

// ResourceInstance represents an expanded resource instance
type ResourceInstance struct {
	ID         string
	Kind       string
	Name       string
	Properties map[string]interface{}
	DriftPolicy *DriftPolicy
	DependsOn  []string
}

// ChangeType represents the type of change to be made
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeUpdate ChangeType = "update"
	ChangeTypeDelete ChangeType = "delete"
)

// Change represents a planned change to infrastructure
type Change struct {
	Type         ChangeType
	ResourceID   string
	ResourceKind string
	ResourceName string
	Properties   map[string]interface{}
	OldValues    map[string]interface{} // For updates
	NewValues    map[string]interface{} // For updates
}

// ChangeSummary represents a summary of planned changes
type ChangeSummary struct {
	Create int
	Update int
	Delete int
	Changes []Change
}

// ExecutionResult represents the result of executing changes
type ExecutionResult struct {
	Success   bool
	Duration  time.Duration
	Changes   []Change
	Errors    []error
}
