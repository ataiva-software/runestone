package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/ataiva-software/runestone/internal/config"
)

var validRuntimes = map[string]bool{
	"nodejs18.x":   true,
	"nodejs20.x":   true,
	"python3.8":    true,
	"python3.9":    true,
	"python3.10":   true,
	"python3.11":   true,
	"python3.12":   true,
	"java8":        true,
	"java11":       true,
	"java17":       true,
	"java21":       true,
	"dotnet6":      true,
	"dotnet8":      true,
	"go1.x":        true,
	"ruby3.2":      true,
	"provided.al2": true,
}

// validateLambdaFunction validates Lambda function configuration
func (p *Provider) validateLambdaFunction(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("Lambda function name cannot be empty")
	}

	// Validate runtime
	runtimeVal, exists := instance.Properties["runtime"]
	if !exists {
		return fmt.Errorf("runtime is required for Lambda function")
	}

	runtime, ok := runtimeVal.(string)
	if !ok {
		return fmt.Errorf("runtime must be a string")
	}

	if !validRuntimes[runtime] {
		return fmt.Errorf("invalid runtime '%s'", runtime)
	}

	// Validate handler
	handlerVal, exists := instance.Properties["handler"]
	if !exists {
		return fmt.Errorf("handler is required for Lambda function")
	}

	handler, ok := handlerVal.(string)
	if !ok {
		return fmt.Errorf("handler must be a string")
	}

	if handler == "" {
		return fmt.Errorf("handler cannot be empty")
	}

	// Validate role
	roleVal, exists := instance.Properties["role"]
	if !exists {
		return fmt.Errorf("role is required for Lambda function")
	}

	role, ok := roleVal.(string)
	if !ok {
		return fmt.Errorf("role must be a string")
	}

	if !strings.HasPrefix(role, "arn:aws:iam::") {
		return fmt.Errorf("invalid role ARN format: %s", role)
	}

	return nil
}

// getLambdaFunctionState retrieves the current state of a Lambda function
func (p *Provider) getLambdaFunctionState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := lambda.NewFromConfig(p.awsConfig)

	input := &lambda.GetFunctionInput{
		FunctionName: aws.String(instance.Name),
	}

	result, err := client.GetFunction(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil, nil // Function doesn't exist
		}
		return nil, fmt.Errorf("failed to describe Lambda function %s: %w", instance.Name, err)
	}

	function := result.Configuration

	// Convert tags to map
	tags := make(map[string]interface{})
	if result.Tags != nil {
		for key, value := range result.Tags {
			tags[key] = value
		}
	}

	state := map[string]interface{}{
		"function_name": *function.FunctionName,
		"function_arn":  *function.FunctionArn,
		"runtime":       string(function.Runtime),
		"handler":       *function.Handler,
		"role":          *function.Role,
		"state":         string(function.State),
		"tags":          tags,
	}

	if function.Description != nil {
		state["description"] = *function.Description
	}

	if function.Timeout != nil {
		state["timeout"] = *function.Timeout
	}

	if function.MemorySize != nil {
		state["memory_size"] = *function.MemorySize
	}

	return state, nil
}

// createLambdaFunction creates a new Lambda function
func (p *Provider) createLambdaFunction(ctx context.Context, instance config.ResourceInstance) error {
	client := lambda.NewFromConfig(p.awsConfig)

	runtime := instance.Properties["runtime"].(string)
	handler := instance.Properties["handler"].(string)
	role := instance.Properties["role"].(string)

	// Default code if not provided
	codeContent := "def handler(event, context):\n    return 'Hello from Lambda!'"
	if codeVal, exists := instance.Properties["code_content"]; exists {
		if code, ok := codeVal.(string); ok {
			codeContent = code
		}
	}

	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(instance.Name),
		Runtime:      types.Runtime(runtime),
		Handler:      aws.String(handler),
		Role:         aws.String(role),
		Code: &types.FunctionCode{
			ZipFile: []byte(codeContent),
		},
	}

	// Add optional properties
	if descVal, exists := instance.Properties["description"]; exists {
		if desc, ok := descVal.(string); ok {
			input.Description = aws.String(desc)
		}
	}

	if timeoutVal, exists := instance.Properties["timeout"]; exists {
		if timeout, ok := timeoutVal.(int); ok {
			input.Timeout = aws.Int32(int32(timeout))
		}
	}

	if memoryVal, exists := instance.Properties["memory_size"]; exists {
		if memory, ok := memoryVal.(int); ok {
			input.MemorySize = aws.Int32(int32(memory))
		}
	}

	// Add tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags := make(map[string]string)
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags[key] = valueStr
				}
			}
			if len(tags) > 0 {
				input.Tags = tags
			}
		}
	}

	_, err := client.CreateFunction(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create Lambda function %s: %w", instance.Name, err)
	}

	return nil
}

// updateLambdaFunction updates an existing Lambda function
func (p *Provider) updateLambdaFunction(ctx context.Context, instance config.ResourceInstance) error {
	client := lambda.NewFromConfig(p.awsConfig)

	// Update function configuration
	configInput := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(instance.Name),
	}

	if runtimeVal, exists := instance.Properties["runtime"]; exists {
		if runtime, ok := runtimeVal.(string); ok {
			configInput.Runtime = types.Runtime(runtime)
		}
	}

	if handlerVal, exists := instance.Properties["handler"]; exists {
		if handler, ok := handlerVal.(string); ok {
			configInput.Handler = aws.String(handler)
		}
	}

	if roleVal, exists := instance.Properties["role"]; exists {
		if role, ok := roleVal.(string); ok {
			configInput.Role = aws.String(role)
		}
	}

	if descVal, exists := instance.Properties["description"]; exists {
		if desc, ok := descVal.(string); ok {
			configInput.Description = aws.String(desc)
		}
	}

	if timeoutVal, exists := instance.Properties["timeout"]; exists {
		if timeout, ok := timeoutVal.(int); ok {
			configInput.Timeout = aws.Int32(int32(timeout))
		}
	}

	if memoryVal, exists := instance.Properties["memory_size"]; exists {
		if memory, ok := memoryVal.(int); ok {
			configInput.MemorySize = aws.Int32(int32(memory))
		}
	}

	_, err := client.UpdateFunctionConfiguration(ctx, configInput)
	if err != nil {
		return fmt.Errorf("failed to update Lambda function configuration %s: %w", instance.Name, err)
	}

	// Update code if provided
	if codeVal, exists := instance.Properties["code_content"]; exists {
		if code, ok := codeVal.(string); ok {
			codeInput := &lambda.UpdateFunctionCodeInput{
				FunctionName: aws.String(instance.Name),
				ZipFile:      []byte(code),
			}

			_, err := client.UpdateFunctionCode(ctx, codeInput)
			if err != nil {
				return fmt.Errorf("failed to update Lambda function code %s: %w", instance.Name, err)
			}
		}
	}

	// Update tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags := make(map[string]string)
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags[key] = valueStr
				}
			}

			if len(tags) > 0 {
				tagInput := &lambda.TagResourceInput{
					Resource: aws.String(fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", 
						p.awsConfig.Region, "123456789012", instance.Name)), // Placeholder ARN
					Tags: tags,
				}
				_, err := client.TagResource(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for Lambda function %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteLambdaFunction deletes a Lambda function
func (p *Provider) deleteLambdaFunction(ctx context.Context, instance config.ResourceInstance) error {
	client := lambda.NewFromConfig(p.awsConfig)

	input := &lambda.DeleteFunctionInput{
		FunctionName: aws.String(instance.Name),
	}

	_, err := client.DeleteFunction(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil // Function already deleted
		}
		return fmt.Errorf("failed to delete Lambda function %s: %w", instance.Name, err)
	}

	return nil
}
