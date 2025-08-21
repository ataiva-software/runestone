package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/drift"
	"github.com/ataiva-software/runestone/internal/executor"
	"github.com/ataiva-software/runestone/internal/providers"
	"github.com/ataiva-software/runestone/internal/providers/aws"
	"github.com/spf13/cobra"
)

var dismantleCmd = &cobra.Command{
	Use:   "dismantle",
	Short: "Destroy infrastructure resources",
	Long: `Dismantle safely destroys infrastructure resources:
- Deletes resources in reverse dependency order
- Shows what will be destroyed before proceeding
- Handles dependencies to avoid orphaned resources`,
	RunE: runDismantle,
}

func init() {
	dismantleCmd.Flags().StringP("config", "c", "infra.yaml", "Path to the configuration file")
	dismantleCmd.Flags().Bool("auto-approve", false, "Skip interactive approval")
	dismantleCmd.Flags().Bool("force", false, "Force deletion even if resources have dependencies")
	dismantleCmd.Flags().StringP("output", "o", "human", "Output format (human, json, markdown)")
}

func runDismantle(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	force, _ := cmd.Flags().GetBool("force")

	fmt.Println("️  Preparing to dismantle infrastructure...")

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.ParseFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
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
			return fmt.Errorf("unsupported provider: %s", providerName)
		}

		providerConfigMap := make(map[string]interface{})
		providerConfigMap["region"] = providerConfig.Region
		providerConfigMap["profile"] = providerConfig.Profile

		if err := provider.Initialize(ctx, providerConfigMap); err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", providerName, err)
		}

		registry.Register(providerName, provider)
	}

	// Expand resources
	instances, err := parser.ExpandResources(cfg.Resources)
	if err != nil {
		return fmt.Errorf("failed to expand resources: %w", err)
	}

	// Detect which resources actually exist
	detector := drift.NewDetector(registry)
	driftResults, err := detector.DetectDriftBatch(ctx, instances)
	if err != nil {
		return fmt.Errorf("failed to detect existing resources: %w", err)
	}

	// Filter to only existing resources
	var existingInstances []config.ResourceInstance
	for _, instance := range instances {
		if driftResult, exists := driftResults[instance.ID]; exists && driftResult.CurrentState != nil {
			existingInstances = append(existingInstances, instance)
		}
	}

	if len(existingInstances) == 0 {
		fmt.Println(" No resources found to dismantle")
		return nil
	}

	// Show what will be destroyed
	fmt.Printf("\n️  The following resources will be destroyed:\n\n")
	for _, instance := range existingInstances {
		fmt.Printf("- %s (%s)\n", instance.ID, instance.Kind)
	}

	// Ask for confirmation
	if !autoApprove {
		fmt.Printf("\nThis action cannot be undone. Do you want to proceed? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Create DAG for deletion (reverse order)
	dag, err := executor.NewDAG(existingInstances)
	if err != nil {
		return fmt.Errorf("failed to create execution DAG: %w", err)
	}

	// Execute deletions
	startTime := time.Now()
	result, err := executeDeletions(ctx, dag, registry, force)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("dismantle failed: %w", err)
	}

	// Display results
	displayDismantleResults(result, duration)

	return nil
}

func executeDeletions(ctx context.Context, dag *executor.DAG, registry *providers.ProviderRegistry, force bool) (*config.ExecutionResult, error) {
	result := &config.ExecutionResult{
		Success:  true,
		Changes:  make([]config.Change, 0),
		Errors:   make([]error, 0),
	}

	// Get execution order and reverse it for deletion
	executionOrder := dag.GetExecutionOrder()
	
	// Reverse the order for safe deletion
	for i := len(executionOrder) - 1; i >= 0; i-- {
		level := executionOrder[i]
		fmt.Printf("\n--- Deletion Level %d ---\n", len(executionOrder)-i)

		// Delete all nodes in this level
		for _, nodeID := range level {
			node, exists := dag.GetNode(nodeID)
			if !exists {
				continue
			}

			// Set node status to running
			dag.SetNodeStatus(nodeID, executor.StatusRunning, nil)

			// Extract provider name
			providerName := extractProviderName(node.Instance.Kind)
			provider, exists := registry.Get(providerName)
			if !exists {
				err := fmt.Errorf("provider %s not found", providerName)
				dag.SetNodeStatus(nodeID, executor.StatusFailed, err)
				result.Errors = append(result.Errors, err)
				result.Success = false
				continue
			}

			// Delete resource
			fmt.Printf("- Deleting %s\n", nodeID)
			err := provider.Delete(ctx, node.Instance)

			// Update node status
			if err != nil {
				fmt.Printf("✗ Failed to delete %s: %v\n", nodeID, err)
				dag.SetNodeStatus(nodeID, executor.StatusFailed, err)
				result.Errors = append(result.Errors, err)
				if !force {
					result.Success = false
				}
			} else {
				fmt.Printf("✓ Deleted %s\n", nodeID)
				dag.SetNodeStatus(nodeID, executor.StatusCompleted, nil)
				result.Changes = append(result.Changes, config.Change{
					Type:         config.ChangeTypeDelete,
					ResourceID:   nodeID,
					ResourceKind: node.Instance.Kind,
					ResourceName: node.Instance.Name,
				})
			}
		}
	}

	return result, nil
}

func displayDismantleResults(result *config.ExecutionResult, duration time.Duration) {
	fmt.Printf("\n--- Dismantle Complete ---\n")
	
	if result.Success {
		fmt.Printf(" Dismantle complete (duration: %v)\n", duration.Round(time.Second))
	} else {
		fmt.Printf("✗ Dismantle completed with errors (duration: %v)\n", duration.Round(time.Second))
	}

	if len(result.Changes) > 0 {
		fmt.Printf("\nResources destroyed:\n")
		for _, change := range result.Changes {
			fmt.Printf("- Deleted %s\n", change.ResourceID)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range result.Errors {
			fmt.Printf("✗ %v\n", err)
		}
	}
}
