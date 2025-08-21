package output

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ataiva-software/runestone/internal/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONFormatter_FormatBootstrapResult(t *testing.T) {
	formatter := NewJSONFormatter()

	tests := []struct {
		name     string
		result   BootstrapResult
		expected map[string]interface{}
	}{
		{
			name: "successful bootstrap",
			result: BootstrapResult{
				Success:           true,
				ProvidersInstalled: []string{"aws", "kubernetes"},
				ResourceCount:     5,
				ModulesLoaded:     2,
				PolicyViolations:  []policy.PolicyViolation{},
				Duration:         time.Second * 2,
			},
			expected: map[string]interface{}{
				"success":             true,
				"providers_installed": []interface{}{"aws", "kubernetes"},
				"resource_count":      float64(5),
				"modules_loaded":      float64(2),
				"policy_violations":   []interface{}{},
				"duration_seconds":    float64(2),
				"has_errors":         false,
			},
		},
		{
			name: "bootstrap with policy violations",
			result: BootstrapResult{
				Success:           true,
				ProvidersInstalled: []string{"aws"},
				ResourceCount:     3,
				ModulesLoaded:     0,
				PolicyViolations: []policy.PolicyViolation{
					{
						ResourceID: "test-bucket",
						Rule: &policy.PolicyRule{
							Name: "s3-versioning",
						},
						Message:  "S3 bucket should have versioning enabled",
						Severity: "warning",
					},
				},
				Duration: time.Millisecond * 1500,
			},
			expected: map[string]interface{}{
				"success":             true,
				"providers_installed": []interface{}{"aws"},
				"resource_count":      float64(3),
				"modules_loaded":      float64(0),
				"policy_violations": []interface{}{
					map[string]interface{}{
						"resource_name": "test-bucket",
						"rule_name":     "s3-versioning",
						"message":       "S3 bucket should have versioning enabled",
						"severity":      "warning",
					},
				},
				"duration_seconds": 1.5,
				"has_errors":      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.FormatBootstrapResult(tt.result)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(output), &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expected["success"], result["success"])
			assert.Equal(t, tt.expected["providers_installed"], result["providers_installed"])
			assert.Equal(t, tt.expected["resource_count"], result["resource_count"])
			assert.Equal(t, tt.expected["modules_loaded"], result["modules_loaded"])
			assert.Equal(t, tt.expected["duration_seconds"], result["duration_seconds"])
			assert.Equal(t, tt.expected["has_errors"], result["has_errors"])
		})
	}
}

func TestJSONFormatter_FormatPreviewResult(t *testing.T) {
	formatter := NewJSONFormatter()

	tests := []struct {
		name     string
		result   PreviewResult
		expected map[string]interface{}
	}{
		{
			name: "no changes detected",
			result: PreviewResult{
				Success:      true,
				ChangesCount: 0,
				Changes:      []Change{},
				DriftResults: []DriftResult{},
				Duration:     time.Second,
			},
			expected: map[string]interface{}{
				"success":       true,
				"changes_count": float64(0),
				"changes":       []interface{}{},
				"drift_results": []interface{}{},
				"duration_seconds": float64(1),
				"has_drift":     false,
			},
		},
		{
			name: "changes and drift detected",
			result: PreviewResult{
				Success:      true,
				ChangesCount: 2,
				Changes: []Change{
					{
						Type:         "create",
						ResourceKind: "aws:s3:bucket",
						ResourceName: "new-bucket",
						Description:  "Create S3 bucket new-bucket",
					},
					{
						Type:         "update",
						ResourceKind: "aws:ec2:instance",
						ResourceName: "web-server",
						Description:  "Update EC2 instance web-server",
					},
				},
				DriftResults: []DriftResult{
					{
						ResourceName: "existing-bucket",
						HasDrift:     true,
						Changes:      []string{"versioning changed from false to true"},
					},
				},
				Duration: time.Millisecond * 2500,
			},
			expected: map[string]interface{}{
				"success":       true,
				"changes_count": float64(2),
				"changes": []interface{}{
					map[string]interface{}{
						"type":          "create",
						"resource_kind": "aws:s3:bucket",
						"resource_name": "new-bucket",
						"description":   "Create S3 bucket new-bucket",
					},
					map[string]interface{}{
						"type":          "update",
						"resource_kind": "aws:ec2:instance",
						"resource_name": "web-server",
						"description":   "Update EC2 instance web-server",
					},
				},
				"drift_results": []interface{}{
					map[string]interface{}{
						"resource_name": "existing-bucket",
						"has_drift":     true,
						"changes":       []interface{}{"versioning changed from false to true"},
					},
				},
				"duration_seconds": 2.5,
				"has_drift":        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.FormatPreviewResult(tt.result)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(output), &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expected["success"], result["success"])
			assert.Equal(t, tt.expected["changes_count"], result["changes_count"])
			assert.Equal(t, tt.expected["duration_seconds"], result["duration_seconds"])
			assert.Equal(t, tt.expected["has_drift"], result["has_drift"])
		})
	}
}

func TestJSONFormatter_FormatCommitResult(t *testing.T) {
	formatter := NewJSONFormatter()

	result := CommitResult{
		Success:        true,
		ResourcesApplied: 3,
		ExecutionLevels: []ExecutionLevel{
			{
				Level: 1,
				Resources: []string{"aws:s3:bucket.logs", "aws:rds:instance.db"},
				Duration:  time.Second * 30,
			},
			{
				Level: 2,
				Resources: []string{"aws:ec2:instance.web-1"},
				Duration:  time.Second * 45,
			},
		},
		TotalDuration: time.Second * 75,
	}

	output, err := formatter.FormatCommitResult(result)
	require.NoError(t, err)

	var jsonResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &jsonResult)
	require.NoError(t, err)

	assert.Equal(t, true, jsonResult["success"])
	assert.Equal(t, float64(3), jsonResult["resources_applied"])
	assert.Equal(t, float64(75), jsonResult["total_duration_seconds"])

	levels := jsonResult["execution_levels"].([]interface{})
	assert.Len(t, levels, 2)

	level1 := levels[0].(map[string]interface{})
	assert.Equal(t, float64(1), level1["level"])
	assert.Equal(t, float64(30), level1["duration_seconds"])
}

func TestMarkdownFormatter_FormatPreviewResult(t *testing.T) {
	formatter := NewMarkdownFormatter()

	result := PreviewResult{
		Success:      true,
		ChangesCount: 2,
		Changes: []Change{
			{
				Type:         "create",
				ResourceKind: "aws:s3:bucket",
				ResourceName: "new-bucket",
				Description:  "Create S3 bucket new-bucket",
			},
			{
				Type:         "update",
				ResourceKind: "aws:ec2:instance",
				ResourceName: "web-server",
				Description:  "Update EC2 instance web-server",
			},
		},
		DriftResults: []DriftResult{
			{
				ResourceName: "existing-bucket",
				HasDrift:     true,
				Changes:      []string{"versioning changed from false to true"},
			},
		},
		Duration: time.Second * 2,
	}

	output, err := formatter.FormatPreviewResult(result)
	require.NoError(t, err)

	// Check that output contains expected markdown elements
	assert.Contains(t, output, "# Infrastructure Preview")
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "**Changes detected:** 2")
	assert.Contains(t, output, "## Planned Changes")
	assert.Contains(t, output, "- ✅ **Create** `aws:s3:bucket.new-bucket`")
	assert.Contains(t, output, "- 🔄 **Update** `aws:ec2:instance.web-server`")
	assert.Contains(t, output, "## Drift Detection")
	assert.Contains(t, output, "- 🔄 **existing-bucket** (drift detected)")
	assert.Contains(t, output, "  - versioning changed from false to true")
}

func TestMarkdownFormatter_FormatCommitResult(t *testing.T) {
	formatter := NewMarkdownFormatter()

	result := CommitResult{
		Success:        true,
		ResourcesApplied: 3,
		ExecutionLevels: []ExecutionLevel{
			{
				Level: 1,
				Resources: []string{"aws:s3:bucket.logs", "aws:rds:instance.db"},
				Duration:  time.Second * 30,
			},
			{
				Level: 2,
				Resources: []string{"aws:ec2:instance.web-1"},
				Duration:  time.Second * 45,
			},
		},
		TotalDuration: time.Second * 75,
	}

	output, err := formatter.FormatCommitResult(result)
	require.NoError(t, err)

	// Check that output contains expected markdown elements
	assert.Contains(t, output, "# Infrastructure Commit")
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "**Status:** ✅ Success")
	assert.Contains(t, output, "**Resources applied:** 3")
	assert.Contains(t, output, "**Total duration:** 1m15s")
	assert.Contains(t, output, "## Execution Details")
	assert.Contains(t, output, "### Level 1 (30")
	assert.Contains(t, output, "- aws:s3:bucket.logs")
	assert.Contains(t, output, "- aws:rds:instance.db")
	assert.Contains(t, output, "### Level 2 (45")
	assert.Contains(t, output, "- aws:ec2:instance.web-1")
}
