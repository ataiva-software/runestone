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

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Apply infrastructure changes",
	Long: `Commit applies the planned changes to your infrastructure:
- Creates, updates, or deletes resources as needed
- Executes changes in dependency order using DAG
- Shows progress and results`,
	RunE: runCommit,
}

func init() {
	commitCmd.Flags().StringP("config", "c", "infra.yaml", "Path to the configuration file")
	commitCmd.Flags().Bool("graph", false, "Show DAG visualization during execution")
	commitCmd.Flags().Bool("auto-approve", false, "Skip interactive approval")
	commitCmd.Flags().StringP("output", "o", "human", "Output format (human, json, markdown)")
}

func runCommit(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	showGraph, _ := cmd.Flags().GetBool("graph")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")

	fmt.Println("⏳ Committing infrastructure changes...")

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

	// Detect drift to determine what needs to be done
	detector := drift.NewDetector(registry)
	driftResults, err := detector.DetectDriftBatch(ctx, instances)
	if err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	// Generate change summary
	changeSummary := generateChangeSummary(instances, driftResults)

	// Show preview and ask for confirmation
	if !autoApprove {
		displayPreviewResults(changeSummary, driftResults)
		fmt.Print("\nDo you want to apply these changes? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Create DAG for execution
	dag, err := executor.NewDAG(instances)
	if err != nil {
		return fmt.Errorf("failed to create execution DAG: %w", err)
	}

	if showGraph {
		displayDAGVisualization(dag)
	}

	// Execute changes
	startTime := time.Now()
	result, err := executeChanges(ctx, dag, registry, driftResults)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Display results
	displayExecutionResults(result, duration)

	return nil
}

func executeChanges(ctx context.Context, dag *executor.DAG, registry *providers.ProviderRegistry, driftResults map[string]*providers.DriftResult) (*config.ExecutionResult, error) {
	result := &config.ExecutionResult{
		Success:  true,
		Changes:  make([]config.Change, 0),
		Errors:   make([]error, 0),
	}

	// Execute in topological order with parallel execution within each level
	executionOrder := dag.GetExecutionOrder()

	for levelIndex, level := range executionOrder {
		fmt.Printf("\n--- Execution Level %d ---\n", levelIndex+1)

		// Execute all nodes in this level in parallel
		type nodeResult struct {
			nodeID string
			change *config.Change
			err    error
		}

		resultChan := make(chan nodeResult, len(level))
		
		// Start goroutines for each node in the level
		for _, nodeID := range level {
			go func(nodeID string) {
				node, exists := dag.GetNode(nodeID)
				if !exists {
					resultChan <- nodeResult{nodeID: nodeID, err: fmt.Errorf("node %s not found", nodeID)}
					return
				}

				driftResult, hasDrift := driftResults[nodeID]
				if !hasDrift {
					resultChan <- nodeResult{nodeID: nodeID}
					return
				}

				// Set node status to running
				dag.SetNodeStatus(nodeID, executor.StatusRunning, nil)

				// Extract provider name
				providerName := extractProviderName(node.Instance.Kind)
				provider, exists := registry.Get(providerName)
				if !exists {
					err := fmt.Errorf("provider %s not found", providerName)
					dag.SetNodeStatus(nodeID, executor.StatusFailed, err)
					resultChan <- nodeResult{nodeID: nodeID, err: err}
					return
				}

				// Execute the appropriate action
				var err error
				var change *config.Change
				
				if driftResult.CurrentState == nil {
					// Create resource
					fmt.Printf("+ Creating %s\n", nodeID)
					err = provider.Create(ctx, node.Instance)
					if err == nil {
						change = &config.Change{
							Type:         config.ChangeTypeCreate,
							ResourceID:   nodeID,
							ResourceKind: node.Instance.Kind,
							ResourceName: node.Instance.Name,
						}
					}
				} else if driftResult.HasDrift {
					// Update resource
					fmt.Printf("~ Updating %s\n", nodeID)
					err = provider.Update(ctx, node.Instance, driftResult.CurrentState)
					if err == nil {
						change = &config.Change{
							Type:         config.ChangeTypeUpdate,
							ResourceID:   nodeID,
							ResourceKind: node.Instance.Kind,
							ResourceName: node.Instance.Name,
						}
					}
				}

				// Update node status
				if err != nil {
					fmt.Printf("✗ Failed to process %s: %v\n", nodeID, err)
					dag.SetNodeStatus(nodeID, executor.StatusFailed, err)
				} else {
					fmt.Printf("✓ Completed %s\n", nodeID)
					dag.SetNodeStatus(nodeID, executor.StatusCompleted, nil)
				}

				resultChan <- nodeResult{nodeID: nodeID, change: change, err: err}
			}(nodeID)
		}

		// Collect results from all goroutines
		for i := 0; i < len(level); i++ {
			res := <-resultChan
			if res.err != nil {
				result.Errors = append(result.Errors, res.err)
				result.Success = false
			}
			if res.change != nil {
				result.Changes = append(result.Changes, *res.change)
			}
		}
	}

	return result, nil
}

func displayDAGVisualization(dag *executor.DAG) {
	fmt.Println("\n--- Execution Plan (DAG) ---")
	
	executionOrder := dag.GetExecutionOrder()
	for levelIndex, level := range executionOrder {
		fmt.Printf("Level %d: ", levelIndex+1)
		for i, nodeID := range level {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(nodeID)
		}
		fmt.Println()
	}
	fmt.Println()
}

func displayExecutionResults(result *config.ExecutionResult, duration time.Duration) {
	fmt.Printf("\n--- Execution Complete ---\n")
	
	if result.Success {
		fmt.Printf("✔ Commit complete (duration: %v)\n", duration.Round(time.Second))
	} else {
		fmt.Printf("✗ Commit completed with errors (duration: %v)\n", duration.Round(time.Second))
	}

	if len(result.Changes) > 0 {
		fmt.Printf("\nChanges applied:\n")
		for _, change := range result.Changes {
			switch change.Type {
			case config.ChangeTypeCreate:
				fmt.Printf("+ Created %s\n", change.ResourceID)
			case config.ChangeTypeUpdate:
				fmt.Printf("~ Updated %s\n", change.ResourceID)
			case config.ChangeTypeDelete:
				fmt.Printf("- Deleted %s\n", change.ResourceID)
			}
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range result.Errors {
			fmt.Printf("✗ %v\n", err)
		}
	}
}
