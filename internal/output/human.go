package output

import (
	"fmt"
	"strings"
	"time"
)

// HumanFormatter implements the Formatter interface for human-readable output
type HumanFormatter struct{}

// NewHumanFormatter creates a new human-readable formatter
func NewHumanFormatter() *HumanFormatter {
	return &HumanFormatter{}
}

// FormatBootstrapResult formats a bootstrap result for human reading
func (f *HumanFormatter) FormatBootstrapResult(result BootstrapResult) (string, error) {
	var sb strings.Builder

	if result.Success {
		sb.WriteString("✔ Bootstrap complete!\n")
	} else {
		sb.WriteString("❌ Bootstrap failed!\n")
	}

	if len(result.ProvidersInstalled) > 0 {
		sb.WriteString(fmt.Sprintf("✔ Installed %d providers: %s\n", 
			len(result.ProvidersInstalled), strings.Join(result.ProvidersInstalled, ", ")))
	}

	sb.WriteString(fmt.Sprintf("✔ Found %d resource instances\n", result.ResourceCount))

	if result.ModulesLoaded > 0 {
		sb.WriteString(fmt.Sprintf("✔ Loaded %d modules\n", result.ModulesLoaded))
	}

	if len(result.PolicyViolations) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  Found %d policy violations:\n", len(result.PolicyViolations)))
		for _, violation := range result.PolicyViolations {
			icon := f.getSeverityIcon(violation.Severity)
			sb.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, violation.ResourceID, violation.Message))
		}
	} else {
		sb.WriteString("✔ No policy violations found\n")
	}

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("❌ Error: %s\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// FormatPreviewResult formats a preview result for human reading
func (f *HumanFormatter) FormatPreviewResult(result PreviewResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("🔍 Inspecting live infrastructure...\n\n")

	if result.ChangesCount == 0 {
		sb.WriteString("✔ No changes detected\n")
	} else {
		sb.WriteString(fmt.Sprintf("Changes detected:\n\n+ %d new resources will be created\n", result.ChangesCount))
		
		if len(result.Changes) > 0 {
			sb.WriteString("\nDetailed changes:\n")
			for _, change := range result.Changes {
				icon := f.getChangeIcon(change.Type)
				title := strings.ToUpper(change.Type[:1]) + strings.ToLower(change.Type[1:])
				sb.WriteString(fmt.Sprintf("%s %s %s.%s (%s)\n", 
					icon, title, change.ResourceKind, change.ResourceName, change.ResourceKind))
			}
		}
	}

	if len(result.DriftResults) > 0 {
		hasDrift := false
		for _, drift := range result.DriftResults {
			if drift.HasDrift {
				hasDrift = true
				break
			}
		}

		if hasDrift {
			sb.WriteString("\n🔄 Drift detected:\n")
			for _, drift := range result.DriftResults {
				if drift.HasDrift {
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", drift.ResourceName, strings.Join(drift.Changes, ", ")))
				}
			}
		}
	}

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("\n❌ Error: %s\n", result.Error.Error()))
	} else {
		sb.WriteString("\nNext: run 'runestone commit' to apply these changes.\n")
	}

	return sb.String(), nil
}

// FormatCommitResult formats a commit result for human reading
func (f *HumanFormatter) FormatCommitResult(result CommitResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("⏳ Committing infrastructure changes...\n\n")

	for _, level := range result.ExecutionLevels {
		sb.WriteString(fmt.Sprintf("--- Execution Level %d ---\n", level.Level))
		for _, resource := range level.Resources {
			sb.WriteString(fmt.Sprintf("+ Creating %s\n", resource))
		}
		for _, resource := range level.Resources {
			sb.WriteString(fmt.Sprintf("✓ Completed %s\n", resource))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("--- Execution Complete ---\n")
	if result.Success {
		sb.WriteString(fmt.Sprintf("✔ Commit complete (duration: %s)\n\n", f.formatDuration(result.TotalDuration)))
		sb.WriteString("Changes applied:\n")
		// This would typically show the actual changes applied
		sb.WriteString(fmt.Sprintf("+ Applied %d resources\n", result.ResourcesApplied))
	} else {
		sb.WriteString("❌ Commit failed\n")
		if result.Error != nil {
			sb.WriteString(fmt.Sprintf("Error: %s\n", result.Error.Error()))
		}
	}

	return sb.String(), nil
}

// FormatAlignResult formats an align result for human reading
func (f *HumanFormatter) FormatAlignResult(result AlignResult) (string, error) {
	var sb strings.Builder

	sb.WriteString("🔄 Aligning desired state with reality...\n")

	if result.DriftDetected {
		sb.WriteString(fmt.Sprintf("🔄 Drift detected and %d actions applied\n", result.ActionsApplied))
		
		if len(result.Resources) > 0 {
			for _, resource := range result.Resources {
				icon := f.getStatusIcon(resource.Status)
				sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, resource.Name, resource.Status))
				for _, change := range resource.Changes {
					sb.WriteString(fmt.Sprintf("    - %s\n", change))
				}
			}
		}
	} else {
		sb.WriteString("✔ Infrastructure aligned (no drift detected)\n")
	}

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("❌ Error: %s\n", result.Error.Error()))
	}

	return sb.String(), nil
}

// Helper methods

func (f *HumanFormatter) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func (f *HumanFormatter) getSeverityIcon(severity string) string {
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

func (f *HumanFormatter) getChangeIcon(changeType string) string {
	switch changeType {
	case "create":
		return "+"
	case "update":
		return "~"
	case "delete":
		return "-"
	default:
		return "•"
	}
}

func (f *HumanFormatter) getStatusIcon(status string) string {
	switch status {
	case "aligned":
		return "✔"
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
