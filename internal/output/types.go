package output

import (
	"time"

	"github.com/ataiva-software/runestone/internal/policy"
)

// Formatter defines the interface for output formatters
type Formatter interface {
	FormatBootstrapResult(result BootstrapResult) (string, error)
	FormatPreviewResult(result PreviewResult) (string, error)
	FormatCommitResult(result CommitResult) (string, error)
	FormatAlignResult(result AlignResult) (string, error)
}

// BootstrapResult represents the result of a bootstrap operation
type BootstrapResult struct {
	Success            bool
	ProvidersInstalled []string
	ResourceCount      int
	ModulesLoaded      int
	PolicyViolations   []policy.PolicyViolation
	Duration           time.Duration
	Error              error
}

// PreviewResult represents the result of a preview operation
type PreviewResult struct {
	Success      bool
	ChangesCount int
	Changes      []Change
	DriftResults []DriftResult
	Duration     time.Duration
	Error        error
}

// CommitResult represents the result of a commit operation
type CommitResult struct {
	Success          bool
	ResourcesApplied int
	ExecutionLevels  []ExecutionLevel
	TotalDuration    time.Duration
	Error            error
}

// AlignResult represents the result of an align operation
type AlignResult struct {
	Success       bool
	DriftDetected bool
	ActionsApplied int
	Resources     []ResourceStatus
	Duration      time.Duration
	Error         error
}

// Change represents a planned infrastructure change
type Change struct {
	Type         string // create, update, delete
	ResourceKind string
	ResourceName string
	Description  string
}

// DriftResult represents drift detection results for a resource
type DriftResult struct {
	ResourceName string
	HasDrift     bool
	Changes      []string
}

// ExecutionLevel represents a level in the DAG execution
type ExecutionLevel struct {
	Level     int
	Resources []string
	Duration  time.Duration
}

// ResourceStatus represents the status of a resource during alignment
type ResourceStatus struct {
	Name     string
	Status   string // aligned, drifted, healed, error
	Changes  []string
	Duration time.Duration
}

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatHuman    OutputFormat = "human"
	FormatJSON     OutputFormat = "json"
	FormatMarkdown OutputFormat = "markdown"
)

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format OutputFormat) Formatter {
	switch format {
	case FormatJSON:
		return NewJSONFormatter()
	case FormatMarkdown:
		return NewMarkdownFormatter()
	default:
		return NewHumanFormatter()
	}
}
