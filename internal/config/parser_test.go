package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected *Config
		wantErr  bool
	}{
		{
			name: "basic configuration",
			yaml: `
project: test-project
environment: dev
providers:
  aws:
    region: us-east-1
    profile: dev
resources:
  - kind: aws:s3:bucket
    name: test-bucket
    properties:
      versioning: true
`,
			expected: &Config{
				Project:     "test-project",
				Environment: "dev",
				Providers: map[string]Provider{
					"aws": {
						Region:  "us-east-1",
						Profile: "dev",
					},
				},
				Resources: []Resource{
					{
						Kind: "aws:s3:bucket",
						Name: "test-bucket",
						Properties: map[string]interface{}{
							"versioning": true,
						},
					},
				},
			},
		},
		{
			name: "configuration with variables and expressions",
			yaml: `
project: test-project
environment: prod
variables:
  region: us-west-2
  tags:
    owner: platform-team
providers:
  aws:
    region: "${region}"
resources:
  - kind: aws:s3:bucket
    name: test-bucket
    properties:
      versioning: true
      tags: "${tags}"
`,
			expected: &Config{
				Project:     "test-project",
				Environment: "prod",
				Variables: map[string]interface{}{
					"region": "us-west-2",
					"tags": map[string]interface{}{
						"owner": "platform-team",
					},
				},
				Providers: map[string]Provider{
					"aws": {
						Region: "us-west-2", // This should be resolved since it's a provider config
					},
				},
				Resources: []Resource{
					{
						Kind: "aws:s3:bucket",
						Name: "test-bucket",
						Properties: map[string]interface{}{
							"versioning": true,
							"tags":       "${tags}", // This should be deferred for resource expansion
						},
					},
				},
			},
		},
		{
			name: "configuration with ternary expressions",
			yaml: `
project: test-project
environment: prod
resources:
  - kind: aws:rds:instance
    name: db
    properties:
      storage_gb: "${environment == 'prod' ? 100 : 20}"
      backup_enabled: "${environment == 'prod'}"
`,
			expected: &Config{
				Project:     "test-project",
				Environment: "prod",
				Resources: []Resource{
					{
						Kind: "aws:rds:instance",
						Name: "db",
						Properties: map[string]interface{}{
							"storage_gb":     "${environment == 'prod' ? 100 : 20}", // Deferred
							"backup_enabled": "${environment == 'prod'}",            // Deferred
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			config, err := parser.Parse([]byte(tt.yaml))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.Project, config.Project)
			assert.Equal(t, tt.expected.Environment, config.Environment)
			assert.Equal(t, tt.expected.Providers, config.Providers)
			assert.Equal(t, len(tt.expected.Resources), len(config.Resources))

			for i, expectedResource := range tt.expected.Resources {
				assert.Equal(t, expectedResource.Kind, config.Resources[i].Kind)
				assert.Equal(t, expectedResource.Name, config.Resources[i].Name)
				assert.Equal(t, expectedResource.Properties, config.Resources[i].Properties)
			}
		})
	}
}

func TestParser_ExpandResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []Resource
		variables map[string]interface{}
		expected  []ResourceInstance
		wantErr   bool
	}{
		{
			name: "single resource without count or for_each",
			resources: []Resource{
				{
					Kind: "aws:s3:bucket",
					Name: "test-bucket",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
			expected: []ResourceInstance{
				{
					ID:   "aws:s3:bucket.test-bucket",
					Kind: "aws:s3:bucket",
					Name: "test-bucket",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
		},
		{
			name: "resource with count",
			resources: []Resource{
				{
					Kind:  "aws:ec2:instance",
					Name:  "web-${index}",
					Count: 3,
					Properties: map[string]interface{}{
						"instance_type": "t3.micro",
					},
				},
			},
			expected: []ResourceInstance{
				{
					ID:   "aws:ec2:instance.web-0",
					Kind: "aws:ec2:instance",
					Name: "web-0",
					Properties: map[string]interface{}{
						"instance_type": "t3.micro",
					},
				},
				{
					ID:   "aws:ec2:instance.web-1",
					Kind: "aws:ec2:instance",
					Name: "web-1",
					Properties: map[string]interface{}{
						"instance_type": "t3.micro",
					},
				},
				{
					ID:   "aws:ec2:instance.web-2",
					Kind: "aws:ec2:instance",
					Name: "web-2",
					Properties: map[string]interface{}{
						"instance_type": "t3.micro",
					},
				},
			},
		},
		{
			name: "resource with for_each",
			resources: []Resource{
				{
					Kind:    "aws:s3:bucket",
					Name:    "logs-${region}",
					ForEach: []interface{}{"us-east-1", "us-west-2"},
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
			expected: []ResourceInstance{
				{
					ID:   "aws:s3:bucket.logs-us-east-1",
					Kind: "aws:s3:bucket",
					Name: "logs-us-east-1",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
				{
					ID:   "aws:s3:bucket.logs-us-west-2",
					Kind: "aws:s3:bucket",
					Name: "logs-us-west-2",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
		},
		{
			name: "resource with for_each expression",
			resources: []Resource{
				{
					Kind:    "aws:s3:bucket",
					Name:    "logs-${region}",
					ForEach: "${regions}",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
			variables: map[string]interface{}{
				"regions": []interface{}{"us-east-1", "us-west-2"},
			},
			expected: []ResourceInstance{
				{
					ID:   "aws:s3:bucket.logs-us-east-1",
					Kind: "aws:s3:bucket",
					Name: "logs-us-east-1",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
				{
					ID:   "aws:s3:bucket.logs-us-west-2",
					Kind: "aws:s3:bucket",
					Name: "logs-us-west-2",
					Properties: map[string]interface{}{
						"versioning": true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			// Initialize variables map even if empty
			if tt.variables != nil {
				parser.variables = tt.variables
			} else {
				parser.variables = make(map[string]interface{})
			}

			instances, err := parser.ExpandResources(tt.resources)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(instances))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.ID, instances[i].ID)
				assert.Equal(t, expected.Kind, instances[i].Kind)
				assert.Equal(t, expected.Name, instances[i].Name)
				assert.Equal(t, expected.Properties, instances[i].Properties)
			}
		})
	}
}

func TestParser_evaluateExpression(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		variables map[string]interface{}
		expected  interface{}
		wantErr   bool
	}{
		{
			name:     "no expression",
			input:    "simple-string",
			expected: "simple-string",
		},
		{
			name:      "simple variable substitution",
			input:     "${region}",
			variables: map[string]interface{}{"region": "us-east-1"},
			expected:  "us-east-1",
		},
		{
			name:      "ternary expression",
			input:     "${env == 'prod' ? 'production' : 'development'}",
			variables: map[string]interface{}{"env": "prod"},
			expected:  "production",
		},
		{
			name:      "multiple expressions",
			input:     "${project}-${env}-bucket",
			variables: map[string]interface{}{"project": "myapp", "env": "dev"},
			expected:  "myapp-dev-bucket",
		},
		{
			name:      "numeric expression",
			input:     "${env == 'prod' ? 100 : 20}",
			variables: map[string]interface{}{"env": "prod"},
			expected:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			if tt.variables != nil {
				parser.variables = tt.variables
			}

			result, err := parser.evaluateExpression(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
