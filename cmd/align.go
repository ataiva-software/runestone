package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/drift"
	"github.com/ataiva-software/runestone/internal/providers"
	"github.com/ataiva-software/runestone/internal/providers/aws"
	"github.com/spf13/cobra"
)

var alignCmd = &cobra.Command{
	Use:   "align",
	Short: "Continuously reconcile drift",
	Long: `Align reconciles infrastructure drift by:
- Detecting differences between current and desired state
- Automatically healing drift for resources with auto-heal enabled
- Reporting drift for resources with notify-only policy`,
	RunE: runAlign,
}

func init() {
	alignCmd.Flags().StringP("config", "c", "infra.yaml", "Path to the configuration file")
	alignCmd.Flags().Bool("once", false, "Run alignment once instead of continuously")
	alignCmd.Flags().Duration("interval", 5*time.Minute, "Interval between alignment checks (ignored with --once)")
	alignCmd.Flags().StringP("output", "o", "human", "Output format (human, json, markdown)")
}

func runAlign(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	runOnce, _ := cmd.Flags().GetBool("once")
	interval, _ := cmd.Flags().GetDuration("interval")

	if runOnce {
		return runAlignmentOnce(configFile)
	}

	fmt.Printf("🔄 Starting continuous alignment (interval: %v)\n", interval)
	fmt.Println("Press Ctrl+C to stop")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial alignment
	if err := runAlignmentOnce(configFile); err != nil {
		fmt.Printf("Initial alignment failed: %v\n", err)
	}

	// Run continuous alignment
	for {
		select {
		case <-ticker.C:
			if err := runAlignmentOnce(configFile); err != nil {
				fmt.Printf("Alignment failed: %v\n", err)
			}
		}
	}
}

func runAlignmentOnce(configFile string) error {
	fmt.Printf("\n🔄 Aligning desired state with reality... (%s)\n", time.Now().Format("15:04:05"))

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

	// Detect drift
	detector := drift.NewDetector(registry)
	driftResults, err := detector.DetectDriftBatch(ctx, instances)
	if err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	// Process drift results
	driftCount := 0
	healedCount := 0
	errorCount := 0

	for _, instance := range instances {
		driftResult, exists := driftResults[instance.ID]
		if !exists || !driftResult.HasDrift {
			continue
		}

		driftCount++

		// Check drift policy
		if instance.DriftPolicy == nil {
			fmt.Printf("  • %s has drift (no policy defined)\n", instance.ID)
			continue
		}

		if instance.DriftPolicy.NotifyOnly {
			fmt.Printf("  • %s has drift (notify-only policy)\n", instance.ID)
			displayDriftDetails(driftResult)
			continue
		}

		if instance.DriftPolicy.AutoHeal {
			fmt.Printf("  • %s has drift - attempting auto-heal...\n", instance.ID)
			
			if err := detector.AutoHeal(ctx, instance, driftResult); err != nil {
				fmt.Printf("    ✗ Auto-heal failed: %v\n", err)
				errorCount++
			} else {
				fmt.Printf("    ✓ Auto-heal successful\n")
				healedCount++
			}
		}
	}

	// Display summary
	if driftCount == 0 {
		fmt.Println(" Infrastructure aligned (no drift detected)")
	} else {
		fmt.Printf(" Infrastructure alignment complete\n")
		fmt.Printf("  - %d resource%s with drift detected\n", driftCount, pluralize(driftCount))
		if healedCount > 0 {
			fmt.Printf("  - %d resource%s auto-healed\n", healedCount, pluralize(healedCount))
		}
		if errorCount > 0 {
			fmt.Printf("  - %d error%s during auto-heal\n", errorCount, pluralize(errorCount))
		}
	}

	return nil
}

func displayDriftDetails(driftResult *providers.DriftResult) {
	for _, diff := range driftResult.Differences {
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
