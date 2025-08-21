package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/ataiva-software/runestone/internal/config"
)

func (p *Provider) validateAPIGateway(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("API Gateway name cannot be empty")
	}
	return nil
}

func (p *Provider) getAPIGatewayState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := apigateway.NewFromConfig(p.awsConfig)

	result, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list REST APIs: %w", err)
	}

	for _, api := range result.Items {
		if api.Name != nil && *api.Name == instance.Name {
			return map[string]interface{}{
				"id":          *api.Id,
				"name":        *api.Name,
				"description": aws.ToString(api.Description),
			}, nil
		}
	}

	return nil, nil
}

func (p *Provider) createAPIGateway(ctx context.Context, instance config.ResourceInstance) error {
	client := apigateway.NewFromConfig(p.awsConfig)

	input := &apigateway.CreateRestApiInput{
		Name: aws.String(instance.Name),
	}

	if descVal, exists := instance.Properties["description"]; exists {
		if desc, ok := descVal.(string); ok {
			input.Description = aws.String(desc)
		}
	}

	_, err := client.CreateRestApi(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create API Gateway %s: %w", instance.Name, err)
	}

	return nil
}

func (p *Provider) updateAPIGateway(ctx context.Context, instance config.ResourceInstance) error {
	return nil // Basic implementation
}

func (p *Provider) deleteAPIGateway(ctx context.Context, instance config.ResourceInstance) error {
	client := apigateway.NewFromConfig(p.awsConfig)

	state, err := p.getAPIGatewayState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	apiId := state["id"].(string)

	_, err = client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{
		RestApiId: aws.String(apiId),
	})
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete API Gateway %s: %w", instance.Name, err)
	}

	return nil
}
