package output

import (
	"fmt"
	"strings"
	"time"
)

// MarkdownFormatter implements the Formatter interface for Markdown output
type MarkdownFormatter struct{}

// NewMarkdownFormatter creates a new Markdown formatter
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

// FormatBootstrapResult formats a bootstrap result as Markdown
func (f *MarkdownFormatter) FormatBootstrapResult(result BootstrapResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Infrastructure Bootstrap\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	if result.Success {
		sb.WriteString("**Status:** ✅ Success\n")
	} else {
		sb.WriteString("**Status:** ❌ Failed\n")
	}
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", f.formatDuration(result.Duration)))
	sb.WriteString(fmt.Sprintf("**Resources:** %d\n", result.ResourceCount))
	sb.WriteString(fmt.Sprintf("**Modules loaded:** %d\n", result.ModulesLoaded))
	sb.WriteString("\n")

	// Providers
	if len(result.ProvidersInstalled) > 0 {
		sb.WriteString("## Providers Installed\n\n")
		for _, provider := range result.ProvidersInstalled {
			sb.WriteString(fmt.Sprintf("- %s\n", provider))
		}
		sb.WriteString("\n")
	}

	// Policy violations
	if len(result.PolicyViolations) > 0 {
		sb.WriteString("## Policy Violations\n\n")
		for _, violation := range result.PolicyViolations {
			icon := f.getSeverityIcon(violation.Severity)
			sb.WriteString(fmt.Sprintf("- %s **%s** (%s): %s\n", 
				icon, violation.ResourceID, violation.Rule.Name, violation.Message))
		}
		sb.WriteString("\n")
	}

	// Error
	if result.Error != nil {
		sb.WriteString("## Error\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// FormatPreviewResult formats a preview result as Markdown
func (f *MarkdownFormatter) FormatPreviewResult(result PreviewResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Infrastructure Preview\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	if result.Success {
		sb.WriteString("**Status:** ✅ Success\n")
	} else {
		sb.WriteString("**Status:** ❌ Failed\n")
	}
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", f.formatDuration(result.Duration)))
	sb.WriteString(fmt.Sprintf("**Changes detected:** %d\n", result.ChangesCount))
	sb.WriteString(fmt.Sprintf("**Drift detected:** %t\n", f.hasDrift(result.DriftResults)))
	sb.WriteString("\n")

	// Planned changes
	if len(result.Changes) > 0 {
		sb.WriteString("## Planned Changes\n\n")
		for _, change := range result.Changes {
			icon := f.getChangeIcon(change.Type)
			title := strings.ToUpper(change.Type[:1]) + strings.ToLower(change.Type[1:])
			sb.WriteString(fmt.Sprintf("- %s **%s** `%s.%s`\n", 
				icon, title, change.ResourceKind, change.ResourceName))
			if change.Description != "" {
				sb.WriteString(fmt.Sprintf("  - %s\n", change.Description))
			}
		}
		sb.WriteString("\n")
	}

	// Drift detection
	if len(result.DriftResults) > 0 {
		sb.WriteString("## Drift Detection\n\n")
		for _, drift := range result.DriftResults {
			if drift.HasDrift {
				sb.WriteString(fmt.Sprintf("- 🔄 **%s** (drift detected)\n", drift.ResourceName))
				for _, change := range drift.Changes {
					sb.WriteString(fmt.Sprintf("  - %s\n", change))
				}
			} else {
				sb.WriteString(fmt.Sprintf("- ✅ **%s** (no drift)\n", drift.ResourceName))
			}
		}
		sb.WriteString("\n")
	}

	// Error
	if result.Error != nil {
		sb.WriteString("## Error\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// FormatCommitResult formats a commit result as Markdown
func (f *MarkdownFormatter) FormatCommitResult(result CommitResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Infrastructure Commit\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	if result.Success {
		sb.WriteString("**Status:** ✅ Success\n")
	} else {
		sb.WriteString("**Status:** ❌ Failed\n")
	}
	sb.WriteString(fmt.Sprintf("**Resources applied:** %d\n", result.ResourcesApplied))
	sb.WriteString(fmt.Sprintf("**Total duration:** %s\n", f.formatDuration(result.TotalDuration)))
	sb.WriteString("\n")

	// Execution details
	if len(result.ExecutionLevels) > 0 {
		sb.WriteString("## Execution Details\n\n")
		for _, level := range result.ExecutionLevels {
			sb.WriteString(fmt.Sprintf("### Level %d (%s)\n\n", 
				level.Level, f.formatDuration(level.Duration)))
			for _, resource := range level.Resources {
				sb.WriteString(fmt.Sprintf("- %s\n", resource))
			}
			sb.WriteString("\n")
		}
	}

	// Error
	if result.Error != nil {
		sb.WriteString("## Error\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// FormatAlignResult formats an align result as Markdown
func (f *MarkdownFormatter) FormatAlignResult(result AlignResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Infrastructure Alignment\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	if result.Success {
		sb.WriteString("**Status:** ✅ Success\n")
	} else {
		sb.WriteString("**Status:** ❌ Failed\n")
	}
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", f.formatDuration(result.Duration)))
	sb.WriteString(fmt.Sprintf("**Drift detected:** %t\n", result.DriftDetected))
	sb.WriteString(fmt.Sprintf("**Actions applied:** %d\n", result.ActionsApplied))
	sb.WriteString("\n")

	// Resource status
	if len(result.Resources) > 0 {
		sb.WriteString("## Resource Status\n\n")
		for _, resource := range result.Resources {
			icon := f.getStatusIcon(resource.Status)
			sb.WriteString(fmt.Sprintf("- %s **%s** (%s) - %s\n", 
				icon, resource.Name, resource.Status, f.formatDuration(resource.Duration)))
			for _, change := range resource.Changes {
				sb.WriteString(fmt.Sprintf("  - %s\n", change))
			}
		}
		sb.WriteString("\n")
	}

	// Error
	if result.Error != nil {
		sb.WriteString("## Error\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// Helper methods

func (f *MarkdownFormatter) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func (f *MarkdownFormatter) getSeverityIcon(severity string) string {
	switch severity {
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "•"
	}
}

func (f *MarkdownFormatter) getChangeIcon(changeType string) string {
	switch changeType {
	case "create":
		return "✅"
	case "update":
		return "🔄"
	case "delete":
		return "❌"
	default:
		return "•"
	}
}

func (f *MarkdownFormatter) getStatusIcon(status string) string {
	switch status {
	case "aligned":
		return "✅"
	case "drifted":
		return "🔄"
	case "healed":
		return "🔧"
	case "error":
		return "❌"
	default:
		return "•"
	}
}

func (f *MarkdownFormatter) hasDrift(driftResults []DriftResult) bool {
	for _, d := range driftResults {
		if d.HasDrift {
			return true
		}
	}
	return false
}
