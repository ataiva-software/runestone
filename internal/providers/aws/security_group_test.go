package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateSecurityGroup(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid security group",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:security_group.test-sg",
				Kind: "aws:ec2:security_group",
				Name: "test-sg",
				Properties: map[string]interface{}{
					"description": "Test security group",
					"vpc_id":      "vpc-12345678",
					"ingress": []interface{}{
						map[string]interface{}{
							"from_port":   80,
							"to_port":     80,
							"protocol":    "tcp",
							"cidr_blocks": []interface{}{"0.0.0.0/0"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "security group with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:security_group.",
				Kind: "aws:ec2:security_group",
				Name: "",
				Properties: map[string]interface{}{
					"description": "Test security group",
					"vpc_id":      "vpc-12345678",
				},
			},
			wantErr: true,
		},
		{
			name: "security group missing description",
			instance: config.ResourceInstance{
				ID:   "aws:ec2:security_group.test-sg",
				Kind: "aws:ec2:security_group",
				Name: "test-sg",
				Properties: map[string]interface{}{
					"vpc_id": "vpc-12345678",
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
