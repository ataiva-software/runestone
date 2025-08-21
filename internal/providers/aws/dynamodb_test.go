package aws

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateDynamoDBTable(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		instance config.ResourceInstance
		wantErr  bool
	}{
		{
			name: "valid DynamoDB table",
			instance: config.ResourceInstance{
				ID:   "aws:dynamodb:table.test-table",
				Kind: "aws:dynamodb:table",
				Name: "test-table",
				Properties: map[string]interface{}{
					"hash_key": "id",
					"attributes": []interface{}{
						map[string]interface{}{
							"name": "id",
							"type": "S",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "DynamoDB table with empty name",
			instance: config.ResourceInstance{
				ID:   "aws:dynamodb:table.",
				Kind: "aws:dynamodb:table",
				Name: "",
				Properties: map[string]interface{}{
					"hash_key": "id",
				},
			},
			wantErr: true,
		},
		{
			name: "DynamoDB table missing hash_key",
			instance: config.ResourceInstance{
				ID:   "aws:dynamodb:table.test-table",
				Kind: "aws:dynamodb:table",
				Name: "test-table",
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
