package drift

import (
	"context"
	"os"
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/ataiva-software/runestone/internal/providers"
	"github.com/ataiva-software/runestone/internal/providers/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetector_DetectDrift_Integration(t *testing.T) {
	// Skip if no AWS credentials
	if !hasValidAWSCredentials() {
		t.Skip("Skipping integration test - AWS credentials not available or invalid")
	}

	// Initialize AWS provider
	awsProvider := &aws.Provider{}
	ctx := context.Background()
	
	err := awsProvider.Initialize(ctx, map[string]interface{}{
		"region":  "us-east-1",
		"profile": "default",
	})
	require.NoError(t, err)

	// Register provider
	registry := providers.NewRegistry()
	registry.Register("aws", awsProvider)

	detector := NewDetector(registry)

	tests := []struct {
		name           string
		instance       config.ResourceInstance
		expectedDrift  bool
		skipReason     string
	}{
		{
			name: "S3 bucket - no drift when bucket doesn't exist",
			instance: config.ResourceInstance{
				Kind: "aws:s3:bucket",
				Name: "test-nonexistent-bucket-drift-" + generateRandomSuffix(),
				Properties: map[string]interface{}{
					"versioning": true,
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
				DriftPolicy: &config.DriftPolicy{
					AutoHeal:   false,
					NotifyOnly: true,
				},
			},
			expectedDrift: true, // Resource doesn't exist, so drift detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			result, err := detector.DetectDrift(ctx, tt.instance)
			require.NoError(t, err)

			if tt.expectedDrift {
				assert.True(t, result.HasDrift, "Expected drift to be detected")
			} else {
				assert.False(t, result.HasDrift, "Expected no drift")
			}
		})
	}
}

func TestDetector_DetectDrift_Unit(t *testing.T) {
	// Create a test provider that implements the interface without mocks
	testProvider := &TestProvider{
		states: make(map[string]map[string]interface{}),
	}

	registry := providers.NewRegistry()
	registry.Register("test", testProvider)

	detector := NewDetector(registry)
	ctx := context.Background()

	tests := []struct {
		name          string
		instance      config.ResourceInstance
		currentState  map[string]interface{}
		expectedDrift bool
		description   string
	}{
		{
			name: "no drift - states match",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "test-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
					"property2": "value2",
				},
			},
			currentState: map[string]interface{}{
				"property1": "value1",
				"property2": "value2",
			},
			expectedDrift: false,
			description:   "When desired and current states match exactly",
		},
		{
			name: "drift detected - property modified",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "test-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
					"property2": "value2",
				},
			},
			currentState: map[string]interface{}{
				"property1": "value1",
				"property2": "different_value",
			},
			expectedDrift: true,
			description:   "When a property value differs between desired and current state",
		},
		{
			name: "drift detected - property added in current",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "test-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
				},
			},
			currentState: map[string]interface{}{
				"property1": "value1",
				"property2": "extra_value",
			},
			expectedDrift: true,
			description:   "When current state has extra properties not in desired state",
		},
		{
			name: "drift detected - property removed (missing in current)",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "test-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
					"property2": "value2",
				},
			},
			currentState: map[string]interface{}{
				"property1": "value1",
			},
			expectedDrift: true,
			description:   "When desired state has properties missing from current state",
		},
		{
			name: "resource doesn't exist",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "nonexistent-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
				},
			},
			currentState:  nil, // Resource doesn't exist
			expectedDrift: true,
			description:   "When resource doesn't exist in current state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the test provider's state
			if tt.currentState != nil {
				testProvider.states[tt.instance.Name] = tt.currentState
			} else {
				delete(testProvider.states, tt.instance.Name)
			}

			result, err := detector.DetectDrift(ctx, tt.instance)
			require.NoError(t, err, "DetectDrift should not return an error")

			assert.Equal(t, tt.expectedDrift, result.HasDrift, 
				"Drift detection result mismatch: %s", tt.description)

			if tt.expectedDrift {
				assert.NotEmpty(t, result.Changes, "Expected changes to be reported when drift is detected")
			}
		})
	}
}

func TestDetector_AutoHeal_Unit(t *testing.T) {
	testProvider := &TestProvider{
		states: make(map[string]map[string]interface{}),
	}

	registry := providers.NewRegistry()
	registry.Register("test", testProvider)

	detector := NewDetector(registry)
	ctx := context.Background()

	tests := []struct {
		name         string
		instance     config.ResourceInstance
		driftResult  *providers.DriftResult
		expectAction bool
		description  string
	}{
		{
			name: "auto-heal disabled",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "test-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
				},
				DriftPolicy: &config.DriftPolicy{
					AutoHeal:   false,
					NotifyOnly: true,
				},
			},
			driftResult: &providers.DriftResult{
				HasDrift: true,
				Changes:  []string{"property1 changed"},
			},
			expectAction: false,
			description:  "When auto-heal is disabled, no action should be taken",
		},
		{
			name: "auto-heal create missing resource",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "missing-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
				},
				DriftPolicy: &config.DriftPolicy{
					AutoHeal:   true,
					NotifyOnly: false,
				},
			},
			driftResult: &providers.DriftResult{
				HasDrift:     true,
				Changes:      []string{"Resource does not exist"},
				CurrentState: nil,
			},
			expectAction: true,
			description:  "When resource is missing and auto-heal is enabled, should create resource",
		},
		{
			name: "auto-heal update existing resource",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "existing-resource",
				Properties: map[string]interface{}{
					"property1": "new_value",
				},
				DriftPolicy: &config.DriftPolicy{
					AutoHeal:   true,
					NotifyOnly: false,
				},
			},
			driftResult: &providers.DriftResult{
				HasDrift: true,
				Changes:  []string{"property1 changed from old_value to new_value"},
				CurrentState: map[string]interface{}{
					"property1": "old_value",
				},
			},
			expectAction: true,
			description:  "When resource exists but has drift and auto-heal is enabled, should update resource",
		},
		{
			name: "no drift - no action needed",
			instance: config.ResourceInstance{
				Kind: "test:resource:type",
				Name: "aligned-resource",
				Properties: map[string]interface{}{
					"property1": "value1",
				},
				DriftPolicy: &config.DriftPolicy{
					AutoHeal:   true,
					NotifyOnly: false,
				},
			},
			driftResult: &providers.DriftResult{
				HasDrift: false,
				Changes:  []string{},
				CurrentState: map[string]interface{}{
					"property1": "value1",
				},
			},
			expectAction: false,
			description:  "When no drift is detected, no action should be taken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset provider state for each test
			testProvider.createCalled = false
			testProvider.updateCalled = false
			testProvider.states = make(map[string]map[string]interface{})

			err := detector.AutoHeal(ctx, tt.instance, tt.driftResult)
			require.NoError(t, err, "AutoHeal should not return an error")

			if tt.expectAction {
				actionTaken := testProvider.createCalled || testProvider.updateCalled
				assert.True(t, actionTaken, "Expected an action to be taken: %s", tt.description)
			} else {
				assert.False(t, testProvider.createCalled, "Expected no create action: %s", tt.description)
				assert.False(t, testProvider.updateCalled, "Expected no update action: %s", tt.description)
			}
		})
	}
}

func TestDetector_compareStates(t *testing.T) {
	detector := &Detector{}

	tests := []struct {
		name        string
		desired     map[string]interface{}
		current     map[string]interface{}
		expectDrift bool
		description string
	}{
		{
			name: "no differences",
			desired: map[string]interface{}{
				"property1": "value1",
				"property2": "value2",
			},
			current: map[string]interface{}{
				"property1": "value1",
				"property2": "value2",
			},
			expectDrift: false,
			description: "Identical states should show no drift",
		},
		{
			name: "property modified",
			desired: map[string]interface{}{
				"property1": "value1",
				"property2": "value2",
			},
			current: map[string]interface{}{
				"property1": "value1",
				"property2": "different_value",
			},
			expectDrift: true,
			description: "Modified property should show drift",
		},
		{
			name: "property added in current",
			desired: map[string]interface{}{
				"property1": "value1",
			},
			current: map[string]interface{}{
				"property1": "value1",
				"property2": "extra_value",
			},
			expectDrift: true,
			description: "Extra property in current state should show drift",
		},
		{
			name: "property removed (ignores metadata)",
			desired: map[string]interface{}{
				"property1": "value1",
				"property2": "value2",
			},
			current: map[string]interface{}{
				"property1": "value1",
				"arn":       "aws:arn:...", // metadata field, should be ignored
			},
			expectDrift: true,
			description: "Missing non-metadata property should show drift",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differences := detector.compareStates(tt.desired, tt.current)
			hasDrift := len(differences) > 0

			assert.Equal(t, tt.expectDrift, hasDrift, 
				"Drift detection mismatch: %s", tt.description)
		})
	}
}

func TestDetector_isMetadataField(t *testing.T) {
	detector := &Detector{}

	metadataFields := []string{
		"arn", "id", "creation_date", "last_modified", 
		"status", "state", "availability_zone",
	}

	for _, field := range metadataFields {
		assert.True(t, detector.isMetadataField(field), 
			"Field '%s' should be recognized as metadata", field)
	}

	nonMetadataFields := []string{
		"name", "versioning", "tags", "instance_type", 
		"ami", "engine", "allocated_storage",
	}

	for _, field := range nonMetadataFields {
		assert.False(t, detector.isMetadataField(field), 
			"Field '%s' should not be recognized as metadata", field)
	}
}

// TestProvider implements the Provider interface for unit testing without mocks
type TestProvider struct {
	states       map[string]map[string]interface{}
	createCalled bool
	updateCalled bool
}

func (tp *TestProvider) Initialize(ctx context.Context, config map[string]interface{}) error {
	return nil
}

func (tp *TestProvider) Create(ctx context.Context, instance config.ResourceInstance) error {
	tp.createCalled = true
	tp.states[instance.Name] = instance.Properties
	return nil
}

func (tp *TestProvider) Update(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error {
	tp.updateCalled = true
	tp.states[instance.Name] = instance.Properties
	return nil
}

func (tp *TestProvider) Delete(ctx context.Context, instance config.ResourceInstance) error {
	delete(tp.states, instance.Name)
	return nil
}

func (tp *TestProvider) GetCurrentState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	state, exists := tp.states[instance.Name]
	if !exists {
		return nil, nil // Resource doesn't exist
	}
	return state, nil
}

func (tp *TestProvider) ValidateResource(instance config.ResourceInstance) error {
	return nil
}

func (tp *TestProvider) GetSupportedResourceTypes() []string {
	return []string{"test:resource:type"}
}

// Helper functions
func hasValidAWSCredentials() bool {
	// Check for AWS credentials in environment or default profile
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}
	
	// Check for AWS profile
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	
	credentialsFile := homeDir + "/.aws/credentials"
	if _, err := os.Stat(credentialsFile); err == nil {
		return true
	}
	
	return false
}

func generateRandomSuffix() string {
	// Simple random suffix for test resources
	return "test123"
}
