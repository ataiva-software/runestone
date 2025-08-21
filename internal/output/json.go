package output

import (
	"encoding/json"

	"github.com/ataiva-software/runestone/internal/policy"
)

// JSONFormatter implements the Formatter interface for JSON output
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// FormatBootstrapResult formats a bootstrap result as JSON
func (f *JSONFormatter) FormatBootstrapResult(result BootstrapResult) (string, error) {
	output := map[string]interface{}{
		"success":             result.Success,
		"providers_installed": result.ProvidersInstalled,
		"resource_count":      result.ResourceCount,
		"modules_loaded":      result.ModulesLoaded,
		"policy_violations":   f.formatPolicyViolations(result.PolicyViolations),
		"duration_seconds":    result.Duration.Seconds(),
		"has_errors":         f.hasErrors(result.PolicyViolations),
	}

	if result.Error != nil {
		output["error"] = result.Error.Error()
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatPreviewResult formats a preview result as JSON
func (f *JSONFormatter) FormatPreviewResult(result PreviewResult) (string, error) {
	output := map[string]interface{}{
		"success":          result.Success,
		"changes_count":    result.ChangesCount,
		"changes":          f.formatChanges(result.Changes),
		"drift_results":    f.formatDriftResults(result.DriftResults),
		"duration_seconds": result.Duration.Seconds(),
		"has_drift":        f.hasDrift(result.DriftResults),
	}

	if result.Error != nil {
		output["error"] = result.Error.Error()
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatCommitResult formats a commit result as JSON
func (f *JSONFormatter) FormatCommitResult(result CommitResult) (string, error) {
	output := map[string]interface{}{
		"success":                result.Success,
		"resources_applied":      result.ResourcesApplied,
		"execution_levels":       f.formatExecutionLevels(result.ExecutionLevels),
		"total_duration_seconds": result.TotalDuration.Seconds(),
	}

	if result.Error != nil {
		output["error"] = result.Error.Error()
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatAlignResult formats an align result as JSON
func (f *JSONFormatter) FormatAlignResult(result AlignResult) (string, error) {
	output := map[string]interface{}{
		"success":          result.Success,
		"drift_detected":   result.DriftDetected,
		"actions_applied":  result.ActionsApplied,
		"resources":        f.formatResourceStatuses(result.Resources),
		"duration_seconds": result.Duration.Seconds(),
	}

	if result.Error != nil {
		output["error"] = result.Error.Error()
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Helper methods

func (f *JSONFormatter) formatPolicyViolations(violations []policy.PolicyViolation) []map[string]interface{} {
	result := make([]map[string]interface{}, len(violations))
	for i, v := range violations {
		result[i] = map[string]interface{}{
			"resource_name": v.ResourceID,
			"rule_name":     v.Rule.Name,
			"message":       v.Message,
			"severity":      v.Severity,
		}
	}
	return result
}

func (f *JSONFormatter) formatChanges(changes []Change) []map[string]interface{} {
	result := make([]map[string]interface{}, len(changes))
	for i, c := range changes {
		result[i] = map[string]interface{}{
			"type":          c.Type,
			"resource_kind": c.ResourceKind,
			"resource_name": c.ResourceName,
			"description":   c.Description,
		}
	}
	return result
}

func (f *JSONFormatter) formatDriftResults(driftResults []DriftResult) []map[string]interface{} {
	result := make([]map[string]interface{}, len(driftResults))
	for i, d := range driftResults {
		result[i] = map[string]interface{}{
			"resource_name": d.ResourceName,
			"has_drift":     d.HasDrift,
			"changes":       d.Changes,
		}
	}
	return result
}

func (f *JSONFormatter) formatExecutionLevels(levels []ExecutionLevel) []map[string]interface{} {
	result := make([]map[string]interface{}, len(levels))
	for i, l := range levels {
		result[i] = map[string]interface{}{
			"level":            l.Level,
			"resources":        l.Resources,
			"duration_seconds": l.Duration.Seconds(),
		}
	}
	return result
}

func (f *JSONFormatter) formatResourceStatuses(resources []ResourceStatus) []map[string]interface{} {
	result := make([]map[string]interface{}, len(resources))
	for i, r := range resources {
		result[i] = map[string]interface{}{
			"name":             r.Name,
			"status":           r.Status,
			"changes":          r.Changes,
			"duration_seconds": r.Duration.Seconds(),
		}
	}
	return result
}

func (f *JSONFormatter) hasErrors(violations []policy.PolicyViolation) bool {
	for _, v := range violations {
		if v.Severity == "error" {
			return true
		}
	}
	return false
}

func (f *JSONFormatter) hasDrift(driftResults []DriftResult) bool {
	for _, d := range driftResults {
		if d.HasDrift {
			return true
		}
	}
	return false
}
