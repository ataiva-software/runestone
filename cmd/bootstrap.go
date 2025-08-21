package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/modules"
	"github.com/ataiva-software/runestone/internal/output"
	"github.com/ataiva-software/runestone/internal/policy"
	"github.com/ataiva-software/runestone/internal/providers"
	"github.com/ataiva-software/runestone/internal/providers/aws"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Install providers, pull modules, and validate configuration",
	Long: `Bootstrap sets up the Runestone environment by:
- Installing required providers (AWS, etc.)
- Pulling modules from their sources
- Validating the configuration against schemas`,
	RunE: runBootstrap,
}

func init() {
	bootstrapCmd.Flags().StringP("config", "c", "infra.yaml", "Path to the configuration file")
	bootstrapCmd.Flags().StringP("output", "o", "human", "Output format (human, json, markdown)")
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	outputFormat, _ := cmd.Flags().GetString("output")
	
	startTime := time.Now()
	
	// Create output formatter
	formatter := output.NewFormatter(output.OutputFormat(outputFormat))
	
	// Initialize result
	result := output.BootstrapResult{
		Success:            false,
		ProvidersInstalled: []string{},
		ResourceCount:      0,
		ModulesLoaded:      0,
		PolicyViolations:   []policy.PolicyViolation{},
		Duration:           0,
		Error:              nil,
	}

	// Only show progress messages for human output
	showProgress := outputFormat == "human"
	
	if showProgress {
		fmt.Println("🔧 Bootstrapping Runestone environment...")
	}

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.ParseFile(configFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse configuration: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatBootstrapResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Set up provider registry
	registry := providers.NewProviderRegistry()

	// Initialize providers
	ctx := context.Background()
	for providerName, providerConfig := range cfg.Providers {
		if showProgress {
			fmt.Printf("✔ Installing provider %s...\n", providerName)
		}

		var provider providers.Provider
		switch providerName {
		case "aws":
			provider = aws.NewProvider()
		default:
			result.Error = fmt.Errorf("unsupported provider: %s", providerName)
			result.Duration = time.Since(startTime)
			output, _ := formatter.FormatBootstrapResult(result)
			fmt.Print(output)
			return result.Error
		}

		// Convert provider config to map[string]interface{}
		providerConfigMap := make(map[string]interface{})
		providerConfigMap["region"] = providerConfig.Region
		providerConfigMap["profile"] = providerConfig.Profile

		if err := provider.Initialize(ctx, providerConfigMap); err != nil {
			result.Error = fmt.Errorf("failed to initialize provider %s: %w", providerName, err)
			result.Duration = time.Since(startTime)
			output, _ := formatter.FormatBootstrapResult(result)
			fmt.Print(output)
			return result.Error
		}

		registry.Register(providerName, provider)
		result.ProvidersInstalled = append(result.ProvidersInstalled, providerName)
	}

	// Validate configuration
	if showProgress {
		fmt.Println("✔ Validating configuration...")
	}
	if err := validateConfiguration(cfg, registry, parser); err != nil {
		result.Error = fmt.Errorf("configuration validation failed: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatBootstrapResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Expand resources to check for errors
	instances, err := parser.ExpandResources(cfg.Resources)
	if err != nil {
		result.Error = fmt.Errorf("failed to expand resources: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatBootstrapResult(result)
		fmt.Print(output)
		return result.Error
	}

	result.ResourceCount = len(instances)

	if showProgress {
		fmt.Printf("✔ Configuration validated successfully\n")
		fmt.Printf("✔ Found %d resource instances\n", len(instances))
		fmt.Printf("🔍 Evaluating policies...\n")
	}

	// Evaluate policies
	policyEngine := policy.NewPolicyEngine()
	if err := policyEngine.LoadBuiltinPolicies(); err != nil {
		result.Error = fmt.Errorf("failed to load builtin policies: %w", err)
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatBootstrapResult(result)
		fmt.Print(output)
		return result.Error
	}

	allViolations := make([]policy.PolicyViolation, 0)
	for _, instance := range instances {
		violations, err := policyEngine.EvaluateResource(ctx, instance)
		if err != nil {
			result.Error = fmt.Errorf("failed to evaluate policies for resource %s: %w", instance.ID, err)
			result.Duration = time.Since(startTime)
			output, _ := formatter.FormatBootstrapResult(result)
			fmt.Print(output)
			return result.Error
		}
		allViolations = append(allViolations, violations...)
	}

	result.PolicyViolations = allViolations

	// Report policy violations for human output
	if showProgress && len(allViolations) > 0 {
		fmt.Printf("⚠️  Found %d policy violations:\n", len(allViolations))
		
		bySeverity := policyEngine.GetViolationsBySeverity(allViolations)
		
		if errors, hasErrors := bySeverity["error"]; hasErrors {
			fmt.Printf("  🚨 %d errors\n", len(errors))
			for _, violation := range errors {
				fmt.Printf("    - %s: %s\n", violation.ResourceID, violation.Message)
			}
		}
		
		if warnings, hasWarnings := bySeverity["warning"]; hasWarnings {
			fmt.Printf("  ⚠️  %d warnings\n", len(warnings))
			for _, violation := range warnings {
				fmt.Printf("    - %s: %s\n", violation.ResourceID, violation.Message)
			}
		}
		
		if info, hasInfo := bySeverity["info"]; hasInfo {
			fmt.Printf("  ℹ️  %d info\n", len(info))
		}
	} else if showProgress {
		fmt.Printf("✔ No policy violations found\n")
	}

	// Fail bootstrap if there are error-level violations
	if policyEngine.HasErrors(allViolations) {
		result.Error = fmt.Errorf("bootstrap failed due to policy violations")
		result.Duration = time.Since(startTime)
		output, _ := formatter.FormatBootstrapResult(result)
		fmt.Print(output)
		return result.Error
	}

	// Pull and validate modules
	if len(cfg.Modules) > 0 {
		if showProgress {
			fmt.Printf("📦 Loading %d modules...\n", len(cfg.Modules))
		}
		
		moduleRegistry := modules.NewRegistry()
		
		for moduleName, moduleConfig := range cfg.Modules {
			if showProgress {
				fmt.Printf("  📦 Loading module: %s\n", moduleName)
			}
			
			module := &modules.Module{
				Name:    moduleName,
				Source:  moduleConfig.Source,
				Version: moduleConfig.Version,
				Inputs:  moduleConfig.Inputs,
			}
			
			// Validate module configuration
			if err := module.Validate(); err != nil {
				result.Error = fmt.Errorf("invalid module configuration for '%s': %w", moduleName, err)
				result.Duration = time.Since(startTime)
				output, _ := formatter.FormatBootstrapResult(result)
				fmt.Print(output)
				return result.Error
			}
			
			// Load the module
			if err := module.Load(); err != nil {
				result.Error = fmt.Errorf("failed to load module '%s': %w", moduleName, err)
				result.Duration = time.Since(startTime)
				output, _ := formatter.FormatBootstrapResult(result)
				fmt.Print(output)
				return result.Error
			}
			
			// Register the module
			if err := moduleRegistry.RegisterModule(module); err != nil {
				result.Error = fmt.Errorf("failed to register module '%s': %w", moduleName, err)
				result.Duration = time.Since(startTime)
				output, _ := formatter.FormatBootstrapResult(result)
				fmt.Print(output)
				return result.Error
			}
			
			if showProgress {
				fmt.Printf("  ✔ Module '%s' loaded successfully\n", moduleName)
			}
			result.ModulesLoaded++
		}
		
		if showProgress {
			fmt.Printf("✔ All modules loaded successfully\n")
		}
	}

	if showProgress {
		fmt.Println("✔ Bootstrap complete!")
	}

	// Success!
	result.Success = true
	result.Duration = time.Since(startTime)
	
	// Output result using formatter
	output, err := formatter.FormatBootstrapResult(result)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}
	
	// For non-human formats, output the formatted result
	if !showProgress {
		fmt.Print(output)
	}

	return nil
}

func validateConfiguration(cfg *config.Config, registry *providers.ProviderRegistry, parser *config.Parser) error {
	// Validate that all required providers are available
	for providerName := range cfg.Providers {
		if _, exists := registry.Get(providerName); !exists {
			return fmt.Errorf("provider %s is not available", providerName)
		}
	}

	// Validate resources using the parser that has the variables
	instances, err := parser.ExpandResources(cfg.Resources)
	if err != nil {
		return fmt.Errorf("failed to expand resources: %w", err)
	}

	for _, instance := range instances {
		// Extract provider name from resource kind
		providerName := extractProviderName(instance.Kind)
		provider, exists := registry.Get(providerName)
		if !exists {
			return fmt.Errorf("provider %s not found for resource %s", providerName, instance.ID)
		}

		// Validate resource with provider
		if err := provider.ValidateResource(instance); err != nil {
			return fmt.Errorf("validation failed for resource %s: %w", instance.ID, err)
		}
	}

	return nil
}

func extractProviderName(kind string) string {
	parts := strings.Split(kind, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
