package aws

import (
	"context"
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestProvider_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		providerConfig map[string]interface{}
		wantErr        bool
	}{
		{
			name: "valid configuration with region and profile",
			providerConfig: map[string]interface{}{
				"region":  "us-east-1",
				"profile": "default", // Use default profile instead of test
			},
			wantErr: false,
		},
		{
			name: "valid configuration with region only",
			providerConfig: map[string]interface{}{
				"region": "us-west-2",
			},
			wantErr: false,
		},
		{
			name:           "empty configuration uses defaults",
			providerConfig: map[string]interface{}{},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewProvider()
			ctx := context.Background()

			err := provider.Initialize(ctx, tt.providerConfig)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider.s3Client)
				assert.NotNil(t, provider.ec2Client)
			}
		})
	}
}

func TestProvider_ValidateResource(t *testing.T) {
	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid S3 bucket",
			instance: config.ResourceInstance{
				Kind: "aws:s3:bucket",
				Name: "test-bucket",
				Properties: map[string]interface{}{
					"versioning": true,
				},
			},
			wantErr: false,
		},
		{
			name: "S3 bucket with invalid name (too short)",
			instance: config.ResourceInstance{
				Kind: "aws:s3:bucket",
				Name: "ab",
				Properties: map[string]interface{}{
					"versioning": true,
				},
			},
			wantErr: true,
		},
		{
			name: "S3 bucket with invalid name (contains underscore)",
			instance: config.ResourceInstance{
				Kind: "aws:s3:bucket",
				Name: "test_bucket",
				Properties: map[string]interface{}{
					"versioning": true,
				},
			},
			wantErr: true,
		},
		{
			name: "S3 bucket with empty name",
			instance: config.ResourceInstance{
				Kind: "aws:s3:bucket",
				Name: "",
				Properties: map[string]interface{}{
					"versioning": true,
				},
			},
			wantErr: true,
		},
		{
			name: "valid EC2 instance",
			instance: config.ResourceInstance{
				Kind: "aws:ec2:instance",
				Name: "test-instance",
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
					"ami":           "ami-12345",
				},
			},
			wantErr: false,
		},
		{
			name: "EC2 instance missing instance_type",
			instance: config.ResourceInstance{
				Kind: "aws:ec2:instance",
				Name: "test-instance",
				Properties: map[string]interface{}{
					"ami": "ami-12345",
				},
			},
			wantErr: true,
		},
		{
			name: "EC2 instance missing ami",
			instance: config.ResourceInstance{
				Kind: "aws:ec2:instance",
				Name: "test-instance",
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
				},
			},
			wantErr: true,
		},
		{
			name: "valid RDS instance",
			instance: config.ResourceInstance{
				Kind: "aws:rds:instance",
				Name: "test-db",
				Properties: map[string]interface{}{
					"db_instance_class":     "db.t3.micro",
					"engine":                "mysql",
					"master_username":       "admin",
					"master_user_password":  "password123",
					"allocated_storage":     20,
				},
			},
			wantErr: false,
		},
		{
			name: "RDS instance missing db_instance_class",
			instance: config.ResourceInstance{
				Kind: "aws:rds:instance",
				Name: "test-db",
				Properties: map[string]interface{}{
					"engine":                "mysql",
					"master_username":       "admin",
					"master_user_password":  "password123",
				},
			},
			wantErr: true,
		},
		{
			name: "RDS instance missing engine",
			instance: config.ResourceInstance{
				Kind: "aws:rds:instance",
				Name: "test-db",
				Properties: map[string]interface{}{
					"db_instance_class":     "db.t3.micro",
					"master_username":       "admin",
					"master_user_password":  "password123",
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported resource type",
			instance: config.ResourceInstance{
				Kind: "aws:lambda:function",
				Name: "test-function",
				Properties: map[string]interface{}{
					"engine": "mysql",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewProvider()
			err := provider.ValidateResource(tt.instance)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_GetSupportedResourceTypes(t *testing.T) {
	provider := NewProvider()
	types := provider.GetSupportedResourceTypes()

	assert.Contains(t, types, "aws:s3:bucket")
	assert.Contains(t, types, "aws:ec2:instance")
	assert.Contains(t, types, "aws:ec2:vpc")
	assert.Contains(t, types, "aws:ec2:subnet")
	assert.Contains(t, types, "aws:ec2:internet_gateway")
	assert.Contains(t, types, "aws:ec2:security_group")
	assert.Contains(t, types, "aws:lambda:function")
	assert.Contains(t, types, "aws:dynamodb:table")
	assert.Contains(t, types, "aws:apigateway:rest_api")
	assert.Contains(t, types, "aws:rds:instance")
	assert.Contains(t, types, "aws:iam:user")
	assert.Contains(t, types, "aws:iam:role")
	assert.Contains(t, types, "aws:iam:policy")
	assert.Len(t, types, 13) // Should have exactly 13 supported types
}

func TestProvider_validateS3Bucket(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid bucket name",
			instance: config.ResourceInstance{
				Name: "my-test-bucket",
			},
			wantErr: false,
		},
		{
			name: "bucket name too short",
			instance: config.ResourceInstance{
				Name: "ab",
			},
			wantErr: true,
		},
		{
			name: "bucket name too long",
			instance: config.ResourceInstance{
				Name: "this-is-a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-for-s3-buckets",
			},
			wantErr: true,
		},
		{
			name: "bucket name with underscore",
			instance: config.ResourceInstance{
				Name: "test_bucket",
			},
			wantErr: true,
		},
		{
			name: "empty bucket name",
			instance: config.ResourceInstance{
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateS3Bucket(tt.instance)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_validateEC2Instance(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid EC2 instance",
			instance: config.ResourceInstance{
				Name: "test-instance",
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
					"ami":           "ami-12345",
				},
			},
			wantErr: false,
		},
		{
			name: "missing instance_type",
			instance: config.ResourceInstance{
				Name: "test-instance",
				Properties: map[string]interface{}{
					"ami": "ami-12345",
				},
			},
			wantErr: true,
		},
		{
			name: "missing ami",
			instance: config.ResourceInstance{
				Name: "test-instance",
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
				},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			instance: config.ResourceInstance{
				Name: "",
				Properties: map[string]interface{}{
					"instance_type": "t3.micro",
					"ami":           "ami-12345",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateEC2Instance(tt.instance)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
