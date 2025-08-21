package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/ataiva-software/runestone/internal/config"
)

// IAM User name validation regex (AWS requirements)
var iamUserNameRegex = regexp.MustCompile(`^[\w+=,.@-]+$`)

// getIAMUserState retrieves the current state of an IAM user
func (p *Provider) getIAMUserState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := iam.NewFromConfig(p.awsConfig)

	input := &iam.GetUserInput{
		UserName: aws.String(instance.Name),
	}

	result, err := client.GetUser(ctx, input)
	if err != nil {
		// Check if user doesn't exist
		if isResourceNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get IAM user %s: %w", instance.Name, err)
	}

	// Get user tags
	tagsInput := &iam.ListUserTagsInput{
		UserName: aws.String(instance.Name),
	}
	tagsResult, err := client.ListUserTags(ctx, tagsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM user tags for %s: %w", instance.Name, err)
	}

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range tagsResult.Tags {
		tags[*tag.Key] = *tag.Value
	}

	state := map[string]interface{}{
		"user_name":   *result.User.UserName,
		"path":        *result.User.Path,
		"user_id":     *result.User.UserId,
		"arn":         *result.User.Arn,
		"create_date": result.User.CreateDate.Format("2006-01-02T15:04:05Z"),
		"tags":        tags,
	}

	return state, nil
}

// createIAMUser creates a new IAM user
func (p *Provider) createIAMUser(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Get path from properties (default to "/")
	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	input := &iam.CreateUserInput{
		UserName: aws.String(instance.Name),
		Path:     aws.String(path),
	}

	_, err := client.CreateUser(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create IAM user %s: %w", instance.Name, err)
	}

	// Add tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}

			if len(tags) > 0 {
				tagInput := &iam.TagUserInput{
					UserName: aws.String(instance.Name),
					Tags:     tags,
				}
				_, err = client.TagUser(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to tag IAM user %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// updateIAMUser updates an existing IAM user
func (p *Provider) updateIAMUser(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Update tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			// First, get existing tags to determine what to remove
			existingTagsInput := &iam.ListUserTagsInput{
				UserName: aws.String(instance.Name),
			}
			existingTagsResult, err := client.ListUserTags(ctx, existingTagsInput)
			if err != nil {
				return fmt.Errorf("failed to get existing tags for IAM user %s: %w", instance.Name, err)
			}

			// Remove tags that are no longer needed
			var tagsToRemove []string
			for _, existingTag := range existingTagsResult.Tags {
				if _, exists := tagsMap[*existingTag.Key]; !exists {
					tagsToRemove = append(tagsToRemove, *existingTag.Key)
				}
			}

			if len(tagsToRemove) > 0 {
				untagInput := &iam.UntagUserInput{
					UserName: aws.String(instance.Name),
					TagKeys:  tagsToRemove,
				}
				_, err = client.UntagUser(ctx, untagInput)
				if err != nil {
					return fmt.Errorf("failed to remove tags from IAM user %s: %w", instance.Name, err)
				}
			}

			// Add/update tags
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}

			if len(tags) > 0 {
				tagInput := &iam.TagUserInput{
					UserName: aws.String(instance.Name),
					Tags:     tags,
				}
				_, err = client.TagUser(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for IAM user %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteIAMUser deletes an IAM user
func (p *Provider) deleteIAMUser(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	input := &iam.DeleteUserInput{
		UserName: aws.String(instance.Name),
	}

	_, err := client.DeleteUser(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil // User already deleted
		}
		return fmt.Errorf("failed to delete IAM user %s: %w", instance.Name, err)
	}

	return nil
}

// validateIAMUser validates IAM user configuration
func (p *Provider) validateIAMUser(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("IAM user name cannot be empty")
	}

	// Validate user name format
	if !iamUserNameRegex.MatchString(instance.Name) {
		return fmt.Errorf("invalid user name '%s': must contain only alphanumeric characters and +=,.@-", instance.Name)
	}

	// Validate user name length
	if len(instance.Name) > 64 {
		return fmt.Errorf("user name '%s' is too long (max 64 characters)", instance.Name)
	}

	// Validate path if specified
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			if !strings.HasPrefix(pathStr, "/") {
				return fmt.Errorf("path must start with /")
			}
			if !strings.HasSuffix(pathStr, "/") {
				return fmt.Errorf("path must end with /")
			}
			if len(pathStr) > 512 {
				return fmt.Errorf("path is too long (max 512 characters)")
			}
		}
	}

	return nil
}

// getIAMRoleState retrieves the current state of an IAM role
func (p *Provider) getIAMRoleState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := iam.NewFromConfig(p.awsConfig)

	input := &iam.GetRoleInput{
		RoleName: aws.String(instance.Name),
	}

	result, err := client.GetRole(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get IAM role %s: %w", instance.Name, err)
	}

	// Get role tags
	tagsInput := &iam.ListRoleTagsInput{
		RoleName: aws.String(instance.Name),
	}
	tagsResult, err := client.ListRoleTags(ctx, tagsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM role tags for %s: %w", instance.Name, err)
	}

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range tagsResult.Tags {
		tags[*tag.Key] = *tag.Value
	}

	state := map[string]interface{}{
		"role_name":           *result.Role.RoleName,
		"path":                *result.Role.Path,
		"role_id":             *result.Role.RoleId,
		"arn":                 *result.Role.Arn,
		"assume_role_policy":  *result.Role.AssumeRolePolicyDocument,
		"create_date":         result.Role.CreateDate.Format("2006-01-02T15:04:05Z"),
		"tags":                tags,
	}

	if result.Role.Description != nil {
		state["description"] = *result.Role.Description
	}

	return state, nil
}

// createIAMRole creates a new IAM role
func (p *Provider) createIAMRole(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Get assume role policy from properties
	assumeRolePolicyVal, exists := instance.Properties["assume_role_policy"]
	if !exists {
		return fmt.Errorf("assume_role_policy is required for IAM role")
	}

	assumeRolePolicy, ok := assumeRolePolicyVal.(string)
	if !ok {
		return fmt.Errorf("assume_role_policy must be a string")
	}

	// Get path from properties (default to "/")
	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	input := &iam.CreateRoleInput{
		RoleName:                 aws.String(instance.Name),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
		Path:                     aws.String(path),
	}

	// Add description if specified
	if descVal, exists := instance.Properties["description"]; exists {
		if descStr, ok := descVal.(string); ok {
			input.Description = aws.String(descStr)
		}
	}

	// Add tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}
			input.Tags = tags
		}
	}

	_, err := client.CreateRole(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create IAM role %s: %w", instance.Name, err)
	}

	return nil
}

// deleteIAMRole deletes an IAM role
func (p *Provider) deleteIAMRole(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	input := &iam.DeleteRoleInput{
		RoleName: aws.String(instance.Name),
	}

	_, err := client.DeleteRole(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil // Role already deleted
		}
		return fmt.Errorf("failed to delete IAM role %s: %w", instance.Name, err)
	}

	return nil
}

// validateIAMRole validates IAM role configuration
func (p *Provider) validateIAMRole(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("IAM role name cannot be empty")
	}

	// Validate role name format
	if !iamUserNameRegex.MatchString(instance.Name) {
		return fmt.Errorf("invalid role name '%s': must contain only alphanumeric characters and +=,.@-", instance.Name)
	}

	// Validate role name length
	if len(instance.Name) > 64 {
		return fmt.Errorf("role name '%s' is too long (max 64 characters)", instance.Name)
	}

	// Validate assume role policy
	assumeRolePolicyVal, exists := instance.Properties["assume_role_policy"]
	if !exists {
		return fmt.Errorf("assume_role_policy is required for IAM role")
	}

	assumeRolePolicy, ok := assumeRolePolicyVal.(string)
	if !ok {
		return fmt.Errorf("assume_role_policy must be a string")
	}

	// Validate JSON format
	var policyDoc interface{}
	if err := json.Unmarshal([]byte(assumeRolePolicy), &policyDoc); err != nil {
		return fmt.Errorf("invalid assume_role_policy JSON: %w", err)
	}

	// Validate path if specified
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			if !strings.HasPrefix(pathStr, "/") {
				return fmt.Errorf("path must start with /")
			}
			if !strings.HasSuffix(pathStr, "/") {
				return fmt.Errorf("path must end with /")
			}
			if len(pathStr) > 512 {
				return fmt.Errorf("path is too long (max 512 characters)")
			}
		}
	}

	return nil
}

// getIAMPolicyState retrieves the current state of an IAM policy
func (p *Provider) getIAMPolicyState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := iam.NewFromConfig(p.awsConfig)

	// For customer managed policies, we need to construct the ARN
	// Format: arn:aws:iam::account-id:policy/path/policy-name
	accountID, err := p.getAccountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}

	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	policyArn := fmt.Sprintf("arn:aws:iam::%s:policy%s%s", accountID, path, instance.Name)

	input := &iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	}

	result, err := client.GetPolicy(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get IAM policy %s: %w", instance.Name, err)
	}

	// Get policy version to retrieve the policy document
	versionInput := &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyArn),
		VersionId: result.Policy.DefaultVersionId,
	}

	versionResult, err := client.GetPolicyVersion(ctx, versionInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM policy version for %s: %w", instance.Name, err)
	}

	// Get policy tags
	tagsInput := &iam.ListPolicyTagsInput{
		PolicyArn: aws.String(policyArn),
	}
	tagsResult, err := client.ListPolicyTags(ctx, tagsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM policy tags for %s: %w", instance.Name, err)
	}

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range tagsResult.Tags {
		tags[*tag.Key] = *tag.Value
	}

	state := map[string]interface{}{
		"policy_name": *result.Policy.PolicyName,
		"path":        *result.Policy.Path,
		"policy_id":   *result.Policy.PolicyId,
		"arn":         *result.Policy.Arn,
		"policy":      *versionResult.PolicyVersion.Document,
		"create_date": result.Policy.CreateDate.Format("2006-01-02T15:04:05Z"),
		"tags":        tags,
	}

	if result.Policy.Description != nil {
		state["description"] = *result.Policy.Description
	}

	return state, nil
}

// validateIAMPolicy validates IAM policy configuration
func (p *Provider) validateIAMPolicy(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("IAM policy name cannot be empty")
	}

	// Validate policy name format
	if !iamUserNameRegex.MatchString(instance.Name) {
		return fmt.Errorf("invalid policy name '%s': must contain only alphanumeric characters and +=,.@-", instance.Name)
	}

	// Validate policy name length
	if len(instance.Name) > 128 {
		return fmt.Errorf("policy name '%s' is too long (max 128 characters)", instance.Name)
	}

	// Validate policy document
	policyVal, exists := instance.Properties["policy"]
	if !exists {
		return fmt.Errorf("policy document is required for IAM policy")
	}

	policy, ok := policyVal.(string)
	if !ok {
		return fmt.Errorf("policy document must be a string")
	}

	// Validate JSON format
	var policyDoc interface{}
	if err := json.Unmarshal([]byte(policy), &policyDoc); err != nil {
		return fmt.Errorf("invalid policy JSON: %w", err)
	}

	// Validate path if specified
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			if !strings.HasPrefix(pathStr, "/") {
				return fmt.Errorf("path must start with /")
			}
			if !strings.HasSuffix(pathStr, "/") {
				return fmt.Errorf("path must end with /")
			}
			if len(pathStr) > 512 {
				return fmt.Errorf("path is too long (max 512 characters)")
			}
		}
	}

	return nil
}

// createIAMPolicy creates a new IAM policy
func (p *Provider) createIAMPolicy(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Get policy document from properties
	policyVal, exists := instance.Properties["policy"]
	if !exists {
		return fmt.Errorf("policy document is required for IAM policy")
	}

	policy, ok := policyVal.(string)
	if !ok {
		return fmt.Errorf("policy document must be a string")
	}

	// Get path from properties (default to "/")
	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	input := &iam.CreatePolicyInput{
		PolicyName:     aws.String(instance.Name),
		PolicyDocument: aws.String(policy),
		Path:           aws.String(path),
	}

	// Add description if specified
	if descVal, exists := instance.Properties["description"]; exists {
		if descStr, ok := descVal.(string); ok {
			input.Description = aws.String(descStr)
		}
	}

	// Add tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}
			input.Tags = tags
		}
	}

	_, err := client.CreatePolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create IAM policy %s: %w", instance.Name, err)
	}

	return nil
}

// updateIAMRole updates an existing IAM role
func (p *Provider) updateIAMRole(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Update assume role policy if specified
	if assumeRolePolicyVal, exists := instance.Properties["assume_role_policy"]; exists {
		if assumeRolePolicy, ok := assumeRolePolicyVal.(string); ok {
			updateInput := &iam.UpdateAssumeRolePolicyInput{
				RoleName:       aws.String(instance.Name),
				PolicyDocument: aws.String(assumeRolePolicy),
			}
			_, err := client.UpdateAssumeRolePolicy(ctx, updateInput)
			if err != nil {
				return fmt.Errorf("failed to update assume role policy for %s: %w", instance.Name, err)
			}
		}
	}

	// Update description if specified
	if descVal, exists := instance.Properties["description"]; exists {
		if descStr, ok := descVal.(string); ok {
			updateDescInput := &iam.UpdateRoleDescriptionInput{
				RoleName:    aws.String(instance.Name),
				Description: aws.String(descStr),
			}
			_, err := client.UpdateRoleDescription(ctx, updateDescInput)
			if err != nil {
				return fmt.Errorf("failed to update description for IAM role %s: %w", instance.Name, err)
			}
		}
	}

	// Update tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			// First, get existing tags to determine what to remove
			existingTagsInput := &iam.ListRoleTagsInput{
				RoleName: aws.String(instance.Name),
			}
			existingTagsResult, err := client.ListRoleTags(ctx, existingTagsInput)
			if err != nil {
				return fmt.Errorf("failed to get existing tags for IAM role %s: %w", instance.Name, err)
			}

			// Remove tags that are no longer needed
			var tagsToRemove []string
			for _, existingTag := range existingTagsResult.Tags {
				if _, exists := tagsMap[*existingTag.Key]; !exists {
					tagsToRemove = append(tagsToRemove, *existingTag.Key)
				}
			}

			if len(tagsToRemove) > 0 {
				untagInput := &iam.UntagRoleInput{
					RoleName: aws.String(instance.Name),
					TagKeys:  tagsToRemove,
				}
				_, err = client.UntagRole(ctx, untagInput)
				if err != nil {
					return fmt.Errorf("failed to remove tags from IAM role %s: %w", instance.Name, err)
				}
			}

			// Add/update tags
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}

			if len(tags) > 0 {
				tagInput := &iam.TagRoleInput{
					RoleName: aws.String(instance.Name),
					Tags:     tags,
				}
				_, err = client.TagRole(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for IAM role %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// updateIAMPolicy updates an existing IAM policy
func (p *Provider) updateIAMPolicy(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Get the policy ARN
	accountID, err := p.getAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account ID: %w", err)
	}

	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	policyArn := fmt.Sprintf("arn:aws:iam::%s:policy%s%s", accountID, path, instance.Name)

	// Update policy document if specified
	if policyVal, exists := instance.Properties["policy"]; exists {
		if policy, ok := policyVal.(string); ok {
			// Create a new policy version
			createVersionInput := &iam.CreatePolicyVersionInput{
				PolicyArn:      aws.String(policyArn),
				PolicyDocument: aws.String(policy),
				SetAsDefault:   true,
			}
			_, err := client.CreatePolicyVersion(ctx, createVersionInput)
			if err != nil {
				return fmt.Errorf("failed to update policy document for %s: %w", instance.Name, err)
			}
		}
	}

	// Update tags if specified
	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			// First, get existing tags to determine what to remove
			existingTagsInput := &iam.ListPolicyTagsInput{
				PolicyArn: aws.String(policyArn),
			}
			existingTagsResult, err := client.ListPolicyTags(ctx, existingTagsInput)
			if err != nil {
				return fmt.Errorf("failed to get existing tags for IAM policy %s: %w", instance.Name, err)
			}

			// Remove tags that are no longer needed
			var tagsToRemove []string
			for _, existingTag := range existingTagsResult.Tags {
				if _, exists := tagsMap[*existingTag.Key]; !exists {
					tagsToRemove = append(tagsToRemove, *existingTag.Key)
				}
			}

			if len(tagsToRemove) > 0 {
				untagInput := &iam.UntagPolicyInput{
					PolicyArn: aws.String(policyArn),
					TagKeys:   tagsToRemove,
				}
				_, err = client.UntagPolicy(ctx, untagInput)
				if err != nil {
					return fmt.Errorf("failed to remove tags from IAM policy %s: %w", instance.Name, err)
				}
			}

			// Add/update tags
			tags := make([]types.Tag, 0, len(tagsMap))
			for key, value := range tagsMap {
				if valueStr, ok := value.(string); ok {
					tags = append(tags, types.Tag{
						Key:   aws.String(key),
						Value: aws.String(valueStr),
					})
				}
			}

			if len(tags) > 0 {
				tagInput := &iam.TagPolicyInput{
					PolicyArn: aws.String(policyArn),
					Tags:      tags,
				}
				_, err = client.TagPolicy(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for IAM policy %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteIAMPolicy deletes an IAM policy
func (p *Provider) deleteIAMPolicy(ctx context.Context, instance config.ResourceInstance) error {
	client := iam.NewFromConfig(p.awsConfig)

	// Get the policy ARN
	accountID, err := p.getAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account ID: %w", err)
	}

	path := "/"
	if pathVal, exists := instance.Properties["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	policyArn := fmt.Sprintf("arn:aws:iam::%s:policy%s%s", accountID, path, instance.Name)

	input := &iam.DeletePolicyInput{
		PolicyArn: aws.String(policyArn),
	}

	_, err = client.DeletePolicy(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil // Policy already deleted
		}
		return fmt.Errorf("failed to delete IAM policy %s: %w", instance.Name, err)
	}

	return nil
}

// getAccountID retrieves the AWS account ID
func (p *Provider) getAccountID(ctx context.Context) (string, error) {
	// Use STS to get caller identity
	stsClient := p.stsClient
	if stsClient == nil {
		return "", fmt.Errorf("STS client not initialized")
	}

	result, err := stsClient.GetCallerIdentity(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	return *result.Account, nil
}
