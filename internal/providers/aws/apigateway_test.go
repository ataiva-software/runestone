package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateAPIGateway(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid API Gateway",
			instance: config.ResourceInstance{
				ID:   "aws:apigateway:rest_api.test-api",
				Kind: "aws:apigateway:rest_api",
				Name: "test-api",
				Properties: map[string]interface{}{
					"description": "Test API",
				},
			},
			wantErr: false,
		},
		{
			name: "API Gateway with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:apigateway:rest_api.",
				Kind: "aws:apigateway:rest_api",
				Name: "",
				Properties: map[string]interface{}{
					"description": "Test API",
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
