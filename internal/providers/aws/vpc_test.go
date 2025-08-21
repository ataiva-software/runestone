package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateVPC(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid VPC",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:vpc.test-vpc",
				Kind: "aws:ec2:vpc",
				Name: "test-vpc",
				Properties: map[string]interface{}{
					"cidr_block": "10.0.0.0/16",
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "VPC with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:vpc.",
				Kind: "aws:ec2:vpc",
				Name: "",
				Properties: map[string]interface{}{
					"cidr_block": "10.0.0.0/16",
				},
			},
			wantErr: true,
		},
		{
			name: "VPC missing CIDR block",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:vpc.test-vpc",
				Kind: "aws:ec2:vpc",
				Name: "test-vpc",
				Properties: map[string]interface{}{
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "VPC with invalid CIDR block",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:vpc.test-vpc",
				Kind: "aws:ec2:vpc",
				Name: "test-vpc",
				Properties: map[string]interface{}{
					"cidr_block": "invalid-cidr",
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

func TestValidateSubnet(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid subnet",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:subnet.test-subnet",
				Kind: "aws:ec2:subnet",
				Name: "test-subnet",
				Properties: map[string]interface{}{
					"vpc_id":            "vpc-12345678",
					"cidr_block":        "10.0.1.0/24",
					"availability_zone": "us-east-1a",
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "subnet missing VPC ID",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:subnet.test-subnet",
				Kind: "aws:ec2:subnet",
				Name: "test-subnet",
				Properties: map[string]interface{}{
					"cidr_block":        "10.0.1.0/24",
					"availability_zone": "us-east-1a",
				},
			},
			wantErr: true,
		},
		{
			name: "subnet missing CIDR block",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:subnet.test-subnet",
				Kind: "aws:ec2:subnet",
				Name: "test-subnet",
				Properties: map[string]interface{}{
					"vpc_id":            "vpc-12345678",
					"availability_zone": "us-east-1a",
				},
			},
			wantErr: true,
		},
		{
			name: "subnet with invalid CIDR block",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:subnet.test-subnet",
				Kind: "aws:ec2:subnet",
				Name: "test-subnet",
				Properties: map[string]interface{}{
					"vpc_id":            "vpc-12345678",
					"cidr_block":        "invalid-cidr",
					"availability_zone": "us-east-1a",
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

func TestValidateInternetGateway(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid internet gateway",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:internet_gateway.test-igw",
				Kind: "aws:ec2:internet_gateway",
				Name: "test-igw",
				Properties: map[string]interface{}{
					"tags": map[string]interface{}{
						"Environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "internet gateway with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:internet_gateway.",
				Kind: "aws:ec2:internet_gateway",
				Name: "",
				Properties: map[string]interface{}{},
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
