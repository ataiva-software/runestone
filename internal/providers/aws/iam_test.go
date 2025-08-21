package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateIAMUser(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid IAM user",
			instance: config.ResourceInstance{
				ID:   "aws:iam:user.test-user",
				Kind: "aws:iam:user",
				Name: "test-user",
				Properties: map[string]interface{}{
					"path": "/",
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "IAM user with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:iam:user.",
				Kind: "aws:iam:user",
				Name: "",
				Properties: map[string]interface{}{
					"path": "/",
				},
			},
			wantErr: true,
		},
		{
			name: "IAM user with invalid path",
			instance: config.ResourceInstance{
				ID:   "aws:iam:user.test-user",
				Kind: "aws:iam:user",
				Name: "test-user",
				Properties: map[string]interface{}{
					"path": "invalid-path", // Should start with /
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

func TestValidateIAMRole(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid IAM role",
			instance: config.ResourceInstance{
				ID:   "aws:iam:role.test-role",
				Kind: "aws:iam:role",
				Name: "test-role",
				Properties: map[string]interface{}{
					"assume_role_policy": `{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Effect": "Allow",
								"Principal": {
									"Service": "ec2.amazonaws.com"
								},
								"Action": "sts:AssumeRole"
							}
						]
					}`,
					"path": "/",
				},
			},
			wantErr: false,
		},
		{
			name: "IAM role missing assume_role_policy",
			instance: config.ResourceInstance{
				ID:   "aws:iam:role.test-role",
				Kind: "aws:iam:role",
				Name: "test-role",
				Properties: map[string]interface{}{
					"path": "/",
				},
			},
			wantErr: true,
		},
		{
			name: "IAM role with invalid assume_role_policy",
			instance: config.ResourceInstance{
				ID:   "aws:iam:role.test-role",
				Kind: "aws:iam:role",
				Name: "test-role",
				Properties: map[string]interface{}{
					"assume_role_policy": "invalid-json",
					"path":               "/",
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

func TestValidateIAMPolicy(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid IAM policy",
			instance: config.ResourceInstance{
				ID:   "aws:iam:policy.test-policy",
				Kind: "aws:iam:policy",
				Name: "test-policy",
				Properties: map[string]interface{}{
					"policy": `{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Effect": "Allow",
								"Action": "s3:GetObject",
								"Resource": "*"
							}
						]
					}`,
					"path":        "/",
					"description": "Test policy for S3 access",
				},
			},
			wantErr: false,
		},
		{
			name: "IAM policy missing policy document",
			instance: config.ResourceInstance{
				ID:   "aws:iam:policy.test-policy",
				Kind: "aws:iam:policy",
				Name: "test-policy",
				Properties: map[string]interface{}{
					"path": "/",
				},
			},
			wantErr: true,
		},
		{
			name: "IAM policy with invalid policy document",
			instance: config.ResourceInstance{
				ID:   "aws:iam:policy.test-policy",
				Kind: "aws:iam:policy",
				Name: "test-policy",
				Properties: map[string]interface{}{
					"policy": "invalid-json",
					"path":   "/",
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
