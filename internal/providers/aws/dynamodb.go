package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ataiva-software/runestone/internal/config"
)

func (p *Provider) validateDynamoDBTable(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("DynamoDB table name cannot be empty")
	}

	if _, exists := instance.Properties["hash_key"]; !exists {
		return fmt.Errorf("hash_key is required for DynamoDB table")
	}

	return nil
}

func (p *Provider) getDynamoDBTableState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := dynamodb.NewFromConfig(p.awsConfig)

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(instance.Name),
	}

	result, err := client.DescribeTable(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to describe DynamoDB table %s: %w", instance.Name, err)
	}

	table := result.Table
	return map[string]interface{}{
		"table_name":   *table.TableName,
		"table_status": string(table.TableStatus),
		"table_arn":    *table.TableArn,
	}, nil
}

func (p *Provider) createDynamoDBTable(ctx context.Context, instance config.ResourceInstance) error {
	client := dynamodb.NewFromConfig(p.awsConfig)

	hashKey := instance.Properties["hash_key"].(string)

	// Default attribute definition
	attributes := []types.AttributeDefinition{
		{
			AttributeName: aws.String(hashKey),
			AttributeType: types.ScalarAttributeTypeS,
		},
	}

	// Override with custom attributes if provided
	if attrsVal, exists := instance.Properties["attributes"]; exists {
		if attrsList, ok := attrsVal.([]interface{}); ok {
			attributes = make([]types.AttributeDefinition, 0, len(attrsList))
			for _, attr := range attrsList {
				if attrMap, ok := attr.(map[string]interface{}); ok {
					name := attrMap["name"].(string)
					attrType := attrMap["type"].(string)
					attributes = append(attributes, types.AttributeDefinition{
						AttributeName: aws.String(name),
						AttributeType: types.ScalarAttributeType(attrType),
					})
				}
			}
		}
	}

	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(hashKey),
			KeyType:       types.KeyTypeHash,
		},
	}

	// Add range key if provided
	if rangeKeyVal, exists := instance.Properties["range_key"]; exists {
		if rangeKey, ok := rangeKeyVal.(string); ok {
			keySchema = append(keySchema, types.KeySchemaElement{
				AttributeName: aws.String(rangeKey),
				KeyType:       types.KeyTypeRange,
			})
		}
	}

	input := &dynamodb.CreateTableInput{
		TableName:            aws.String(instance.Name),
		AttributeDefinitions: attributes,
		KeySchema:            keySchema,
		BillingMode:          types.BillingModePayPerRequest,
	}

	_, err := client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create DynamoDB table %s: %w", instance.Name, err)
	}

	return nil
}

func (p *Provider) updateDynamoDBTable(ctx context.Context, instance config.ResourceInstance) error {
	// DynamoDB tables have limited update capabilities
	return nil
}

func (p *Provider) deleteDynamoDBTable(ctx context.Context, instance config.ResourceInstance) error {
	client := dynamodb.NewFromConfig(p.awsConfig)

	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(instance.Name),
	})
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete DynamoDB table %s: %w", instance.Name, err)
	}

	return nil
}
