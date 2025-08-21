package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateLambdaFunction(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid Lambda function",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"runtime":      "python3.9",
					"handler":      "index.handler",
					"role":         "arn:aws:iam::123456789012:role/lambda-role",
					"code_content": "def handler(event, context): return 'Hello World'",
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Lambda function with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.",
				Kind: "aws:lambda:function",
				Name: "",
				Properties: map[string]interface{}{
					"runtime": "python3.9",
					"handler": "index.handler",
					"role":    "arn:aws:iam::123456789012:role/lambda-role",
				},
			},
			wantErr: true,
		},
		{
			name: "Lambda function missing runtime",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"handler": "index.handler",
					"role":    "arn:aws:iam::123456789012:role/lambda-role",
				},
			},
			wantErr: true,
		},
		{
			name: "Lambda function missing handler",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"runtime": "python3.9",
					"role":    "arn:aws:iam::123456789012:role/lambda-role",
				},
			},
			wantErr: true,
		},
		{
			name: "Lambda function missing role",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"runtime": "python3.9",
					"handler": "index.handler",
				},
			},
			wantErr: true,
		},
		{
			name: "Lambda function with invalid runtime",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"runtime": "invalid-runtime",
					"handler": "index.handler",
					"role":    "arn:aws:iam::123456789012:role/lambda-role",
				},
			},
			wantErr: true,
		},
		{
			name: "Lambda function with invalid role ARN",
			instance: config.ResourceInstance{
				ID:   "aws:lambda:function.test-function",
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"runtime": "python3.9",
					"handler": "index.handler",
					"role":    "invalid-role-arn",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateResource(tt.instance)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
