package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/drift"
	"github.com/ataiva-software/runestone/internal/output"
	"github.com/ataiva-software/runestone/internal/providers"
	"github.com/ataiva-software/runestone/internal/providers/aws"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview changes and detect drift",
	Long: `Preview performs a dry-run to show what changes would be made:
- Detects drift between current and desired state
- Shows planned changes in Option A format
- Validates resources without making changes`,
	RunE: runPreview,
}

func init() {
	previewCmd.Flags().StringP("config", "c", "infra.yaml", "Path to the configuration file")
	previewCmd.Flags().StringP("output", "o", "human", "Output format (human, json, markdown)")
}

func runPreview(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	outputFormat, _ := cmd.Flags().GetString("output")
	
	startTime := time.Now()
	
	// Create output formatter
	formatter := output.NewFormatter(output.OutputFormat(outputFormat))
	
	// Initialize result
	result := output.PreviewResult{
		Success:      false,
		ChangesCount: 0,
		Changes:      []output.Change{},
		DriftResults: []output.DriftResult{},
		Duration:     0,
		Error:        nil,
	}

	// Only show progress messages for human output
	showProgress := outputFormat == "human"
	
	if showProgress {
		fmt.Println(" Inspecting live infrastructure...")
	}

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.ParseFile(configFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse configuration: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatPreviewResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Set up provider registry
	registry := providers.NewProviderRegistry()
	ctx := context.Background()

	// Initialize providers
	for providerName, providerConfig := range cfg.Providers {
		var provider providers.Provider
		switch providerName {
		case "aws":
			provider = aws.NewProvider()
		default:
			result.Error = fmt.Errorf("unsupported provider: %s", providerName)
			result.Duration = time.Since(startTime)
			output, _ := formatter.FormatPreviewResult(result)
			fmt.Print(output)
			return result.Error
		}

		providerConfigMap := make(map[string]interface{})
		providerConfigMap["region"] = providerConfig.Region
		providerConfigMap["profile"] = providerConfig.Profile

		if err := provider.Initialize(ctx, providerConfigMap); err != nil {
			result.Error = fmt.Errorf("failed to initialize provider %s: %w", providerName, err)
			result.Duration = time.Since(startTime)
			output, _ := formatter.FormatPreviewResult(result)
			fmt.Print(output)
			return result.Error
		}

		registry.Register(providerName, provider)
	}

	// Expand resources using the same parser that has the variables
	instances, err := parser.ExpandResources(cfg.Resources)
	if err != nil {
		result.Error = fmt.Errorf("failed to expand resources: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatPreviewResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Detect drift
	detector := drift.NewDetector(registry)
	driftResults, err := detector.DetectDriftBatch(ctx, instances)
	if err != nil {
		result.Error = fmt.Errorf("failed to detect drift: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatPreviewResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Convert results to output format
	result.Changes, result.DriftResults = convertToOutputFormat(instances, driftResults)
	result.ChangesCount = len(result.Changes)
	result.Success = true
	result.Duration = time.Since(startTime)

	// Display results using formatter
	outputStr, err := formatter.FormatPreviewResult(result)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}
	
	fmt.Print(outputStr)
	return nil
}

func convertToOutputFormat(instances []config.ResourceInstance, driftResults map[string]*providers.DriftResult) ([]output.Change, []output.DriftResult) {
	changes := make([]output.Change, 0)
	driftResultsOutput := make([]output.DriftResult, 0)

	for _, instance := range instances {
		driftResult, exists := driftResults[instance.ID]
		if !exists {
			continue
		}

		// Add drift result
		driftChanges := make([]string, 0)
		if driftResult.HasDrift {
			for _, diff := range driftResult.Differences {
				switch diff.DriftType {
				case providers.DriftTypeAdded:
					driftChanges = append(driftChanges, fmt.Sprintf("Missing property: %s (expected: %v)", diff.Property, diff.DesiredValue))
				case providers.DriftTypeModified:
					driftChanges = append(driftChanges, fmt.Sprintf("Property %s: %v → %v", diff.Property, diff.CurrentValue, diff.DesiredValue))
				case providers.DriftTypeRemoved:
					driftChanges = append(driftChanges, fmt.Sprintf("Extra property: %s (current: %v)", diff.Property, diff.CurrentValue))
				}
			}
		}

		driftResultsOutput = append(driftResultsOutput, output.DriftResult{
			ResourceName: instance.ID,
			HasDrift:     driftResult.HasDrift,
			Changes:      driftChanges,
		})

		// Add change if needed
		if driftResult.CurrentState == nil {
			// Resource doesn't exist - needs to be created
			changes = append(changes, output.Change{
				Type:         "create",
				ResourceKind: instance.Kind,
				ResourceName: instance.Name,
				Description:  fmt.Sprintf("Create %s %s", instance.Kind, instance.Name),
			})
		} else if driftResult.HasDrift {
			// Resource exists but has drift - needs to be updated
			changes = append(changes, output.Change{
				Type:         "update",
				ResourceKind: instance.Kind,
				ResourceName: instance.Name,
				Description:  fmt.Sprintf("Update %s %s", instance.Kind, instance.Name),
			})
		}
	}

	return changes, driftResultsOutput
}

// Legacy function for commit command compatibility
func generateChangeSummary(instances []config.ResourceInstance, driftResults map[string]*providers.DriftResult) *config.ChangeSummary {
	summary := &config.ChangeSummary{
		Changes: make([]config.Change, 0),
	}

	for _, instance := range instances {
		driftResult, exists := driftResults[instance.ID]
		if !exists {
			continue
		}

		if driftResult.CurrentState == nil {
			// Resource doesn't exist - needs to be created
			summary.Create++
			summary.Changes = append(summary.Changes, config.Change{
				Type:         config.ChangeTypeCreate,
				ResourceID:   instance.ID,
				ResourceKind: instance.Kind,
				ResourceName: instance.Name,
				Properties:   instance.Properties,
				NewValues:    instance.Properties,
			})
		} else if driftResult.HasDrift {
			// Resource exists but has drift - needs to be updated
			summary.Update++
			
			oldValues := make(map[string]interface{})
			newValues := make(map[string]interface{})
			
			for _, diff := range driftResult.Differences {
				oldValues[diff.Property] = diff.CurrentValue
				newValues[diff.Property] = diff.DesiredValue
			}

			summary.Changes = append(summary.Changes, config.Change{
				Type:         config.ChangeTypeUpdate,
				ResourceID:   instance.ID,
				ResourceKind: instance.Kind,
				ResourceName: instance.Name,
				Properties:   instance.Properties,
				OldValues:    oldValues,
				NewValues:    newValues,
			})
		}
	}

	return summary
}

// Legacy function for commit command compatibility
func displayPreviewResults(summary *config.ChangeSummary, driftResults map[string]*providers.DriftResult) {
	// Display drift information
	driftCount := 0
	for _, result := range driftResults {
		if result.HasDrift {
			driftCount++
		}
	}

	if driftCount > 0 {
		fmt.Println("\nDrift detected:")
		for resourceID, result := range driftResults {
			if result.HasDrift && result.CurrentState != nil {
				fmt.Printf("  • %s has configuration drift\n", resourceID)
				for _, diff := range result.Differences {
					switch diff.DriftType {
					case providers.DriftTypeAdded:
						fmt.Printf("    - Missing property: %s (expected: %v)\n", diff.Property, diff.DesiredValue)
					case providers.DriftTypeModified:
						fmt.Printf("    - Property %s: %v → %v\n", diff.Property, diff.CurrentValue, diff.DesiredValue)
					case providers.DriftTypeRemoved:
						fmt.Printf("    - Extra property: %s (current: %v)\n", diff.Property, diff.CurrentValue)
					}
				}
			}
		}
		fmt.Println()
	}

	// Display change summary in Option A format
	fmt.Println("Changes detected:")
	fmt.Println()

	if summary.Create > 0 {
		fmt.Printf("+ %d new resource%s will be created\n", summary.Create, pluralize(summary.Create))
	}

	if summary.Update > 0 {
		fmt.Printf("~ %d resource%s will be updated\n", summary.Update, pluralize(summary.Update))
	}

	if summary.Delete > 0 {
		fmt.Printf("- %d resource%s will be removed\n", summary.Delete, pluralize(summary.Delete))
	}

	if summary.Create == 0 && summary.Update == 0 && summary.Delete == 0 {
		fmt.Println("No changes detected - infrastructure is up to date")
	}

	fmt.Println()

	// Show detailed changes
	if len(summary.Changes) > 0 {
		fmt.Println("Detailed changes:")
		for _, change := range summary.Changes {
			switch change.Type {
			case config.ChangeTypeCreate:
				fmt.Printf("+ Create %s (%s)\n", change.ResourceID, change.ResourceKind)
			case config.ChangeTypeUpdate:
				fmt.Printf("~ Update %s (%s)\n", change.ResourceID, change.ResourceKind)
				for property, newValue := range change.NewValues {
					if oldValue, exists := change.OldValues[property]; exists {
						fmt.Printf("    %s: %v → %v\n", property, oldValue, newValue)
					} else {
						fmt.Printf("    %s: %v (new)\n", property, newValue)
					}
				}
			case config.ChangeTypeDelete:
				fmt.Printf("- Delete %s (%s)\n", change.ResourceID, change.ResourceKind)
			}
		}
	}

	if summary.Create > 0 || summary.Update > 0 || summary.Delete > 0 {
		fmt.Println("\nNext: run 'runestone commit' to apply these changes.")
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
