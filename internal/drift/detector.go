package drift

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/providers"
)

// Detector handles drift detection for resources
type Detector struct {
	providers map[string]providers.Provider
}

// NewDetector creates a new drift detector
func NewDetector(providerRegistry *providers.ProviderRegistry) *Detector {
	return &Detector{
		providers: providerRegistry.GetAll(),
	}
}

// DetectDrift detects drift for a single resource instance
func (d *Detector) DetectDrift(ctx context.Context, instance config.ResourceInstance) (*providers.DriftResult, error) {
	// Extract provider name from resource kind (e.g., "aws:s3:bucket" -> "aws")
	providerName := extractProviderName(instance.Kind)
	provider, exists := d.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found for resource %s", providerName, instance.ID)
	}

	// Get current state from the provider
	currentState, err := provider.GetCurrentState(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state for resource %s: %w", instance.ID, err)
	}

	// If resource doesn't exist, it's a drift (should be created)
	if currentState == nil {
		return &providers.DriftResult{
			HasDrift:     true,
			Changes:      []string{"Resource does not exist"},
			Differences:  map[string]providers.DriftDifference{},
			CurrentState: nil,
			DesiredState: instance.Properties,
		}, nil
	}

	// Compare current state with desired state
	differences := d.compareStates(currentState, instance.Properties)
	changes := d.differencesToChanges(differences)

	return &providers.DriftResult{
		HasDrift:     len(differences) > 0,
		Changes:      changes,
		Differences:  differences,
		CurrentState: currentState,
		DesiredState: instance.Properties,
	}, nil
}

// DetectDriftBatch detects drift for multiple resource instances
func (d *Detector) DetectDriftBatch(ctx context.Context, instances []config.ResourceInstance) (map[string]*providers.DriftResult, error) {
	results := make(map[string]*providers.DriftResult)

	for _, instance := range instances {
		result, err := d.DetectDrift(ctx, instance)
		if err != nil {
			return nil, fmt.Errorf("failed to detect drift for resource %s: %w", instance.ID, err)
		}
		results[instance.ID] = result
	}

	return results, nil
}

// AutoHeal attempts to automatically heal drift for resources with auto-heal enabled
func (d *Detector) AutoHeal(ctx context.Context, instance config.ResourceInstance, driftResult *providers.DriftResult) error {
	// Check if auto-heal is enabled for this resource
	if instance.DriftPolicy == nil || !instance.DriftPolicy.AutoHeal {
		// Auto-heal is disabled, nothing to do
		return nil
	}

	// Extract provider name from resource kind
	providerName := extractProviderName(instance.Kind)
	provider, exists := d.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found for resource %s", providerName, instance.ID)
	}

	// If resource doesn't exist, create it
	if driftResult.CurrentState == nil {
		return provider.Create(ctx, instance)
	}

	// If resource exists but has drift, update it
	if driftResult.HasDrift {
		return provider.Update(ctx, instance, driftResult.CurrentState)
	}

	return nil
}

// compareStates compares current state with desired state and returns differences
func (d *Detector) compareStates(current, desired map[string]interface{}) map[string]providers.DriftDifference {
	differences := make(map[string]providers.DriftDifference)

	// Check for properties that exist in desired but not in current (added)
	for key, desiredValue := range desired {
		currentValue, exists := current[key]
		if !exists {
			differences[key] = providers.DriftDifference{
				Property:     key,
				CurrentValue: nil,
				DesiredValue: desiredValue,
				DriftType:    providers.DriftTypeAdded,
			}
			continue
		}

		// Check if values are different (modified)
		if !d.valuesEqual(currentValue, desiredValue) {
			differences[key] = providers.DriftDifference{
				Property:     key,
				CurrentValue: currentValue,
				DesiredValue: desiredValue,
				DriftType:    providers.DriftTypeModified,
			}
		}
	}

	// Check for properties that exist in current but not in desired (removed)
	for key, currentValue := range current {
		if _, exists := desired[key]; !exists {
			// Skip certain metadata fields that shouldn't be considered drift
			if d.isMetadataField(key) {
				continue
			}

			differences[key] = providers.DriftDifference{
				Property:     key,
				CurrentValue: currentValue,
				DesiredValue: nil,
				DriftType:    providers.DriftTypeRemoved,
			}
		}
	}

	return differences
}

// differencesToChanges converts differences to human-readable change descriptions
func (d *Detector) differencesToChanges(differences map[string]providers.DriftDifference) []string {
	var changes []string
	
	for _, diff := range differences {
		switch diff.DriftType {
		case providers.DriftTypeAdded:
			changes = append(changes, fmt.Sprintf("Property '%s' added with value '%v'", diff.Property, diff.DesiredValue))
		case providers.DriftTypeRemoved:
			changes = append(changes, fmt.Sprintf("Property '%s' removed (was '%v')", diff.Property, diff.CurrentValue))
		case providers.DriftTypeModified:
			changes = append(changes, fmt.Sprintf("Property '%s' changed from '%v' to '%v'", diff.Property, diff.CurrentValue, diff.DesiredValue))
		}
	}
	
	return changes
}

// valuesEqual compares two values for equality, handling different types appropriately
func (d *Detector) valuesEqual(current, desired interface{}) bool {
	// Handle nil values
	if current == nil && desired == nil {
		return true
	}
	if current == nil || desired == nil {
		return false
	}

	// Use reflection for deep comparison
	return reflect.DeepEqual(current, desired)
}

// isMetadataField checks if a field is metadata that shouldn't be considered for drift
func (d *Detector) isMetadataField(fieldName string) bool {
	metadataFields := []string{
		"arn",
		"id",
		"creation_date",
		"last_modified",
		"status",
		"state",
		"availability_zone",
		"instance_id",
		"vpc_id",
		"subnet_id",
	}

	for _, field := range metadataFields {
		if fieldName == field {
			return true
		}
	}

	return false
}

// extractProviderName extracts the provider name from a resource kind
func extractProviderName(kind string) string {
	parts := strings.Split(kind, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// DriftSummary represents a summary of drift detection results
type DriftSummary struct {
	TotalResources    int
	ResourcesWithDrift int
	DriftResults      map[string]*providers.DriftResult
}

// GenerateDriftSummary generates a summary of drift detection results
func (d *Detector) GenerateDriftSummary(results map[string]*providers.DriftResult) *DriftSummary {
	summary := &DriftSummary{
		TotalResources: len(results),
		DriftResults:   results,
	}

	for _, result := range results {
		if result.HasDrift {
			summary.ResourcesWithDrift++
		}
	}

	return summary
}
