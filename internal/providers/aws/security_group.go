package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ataiva-software/runestone/internal/config"
)

func (p *Provider) validateSecurityGroup(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("security group name cannot be empty")
	}

	descVal, exists := instance.Properties["description"]
	if !exists {
		return fmt.Errorf("description is required for security group")
	}

	if desc, ok := descVal.(string); !ok || desc == "" {
		return fmt.Errorf("description must be a non-empty string")
	}

	return nil
}

func (p *Provider) getSecurityGroupState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := ec2.NewFromConfig(p.awsConfig)

	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{instance.Name},
			},
		},
	}

	result, err := client.DescribeSecurityGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe security group %s: %w", instance.Name, err)
	}

	if len(result.SecurityGroups) == 0 {
		return nil, nil
	}

	sg := result.SecurityGroups[0]
	tags := make(map[string]interface{})
	for _, tag := range sg.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return map[string]interface{}{
		"group_id":    *sg.GroupId,
		"group_name":  *sg.GroupName,
		"description": *sg.Description,
		"vpc_id":      *sg.VpcId,
		"tags":        tags,
	}, nil
}

func (p *Provider) createSecurityGroup(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	description := instance.Properties["description"].(string)
	vpcId := ""
	if vpcVal, exists := instance.Properties["vpc_id"]; exists {
		if vpc, ok := vpcVal.(string); ok {
			vpcId = vpc
		}
	}

	input := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(instance.Name),
		Description: aws.String(description),
	}

	if vpcId != "" {
		input.VpcId = aws.String(vpcId)
	}

	result, err := client.CreateSecurityGroup(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create security group %s: %w", instance.Name, err)
	}

	groupId := *result.GroupId

	// Add tags
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
				_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
					Resources: []string{groupId},
					Tags:      tags,
				})
				if err != nil {
					return fmt.Errorf("failed to tag security group %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

func (p *Provider) updateSecurityGroup(ctx context.Context, instance config.ResourceInstance) error {
	// Security groups can only have their tags updated
	state, err := p.getSecurityGroupState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("security group %s not found", instance.Name)
	}

	groupId := state["group_id"].(string)

	if tagsVal, exists := instance.Properties["tags"]; exists {
		if tagsMap, ok := tagsVal.(map[string]interface{}); ok {
			client := ec2.NewFromConfig(p.awsConfig)
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
				_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
					Resources: []string{groupId},
					Tags:      tags,
				})
				if err != nil {
					return fmt.Errorf("failed to update tags for security group %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

func (p *Provider) deleteSecurityGroup(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	state, err := p.getSecurityGroupState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	groupId := state["group_id"].(string)

	_, err = client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(groupId),
	})
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete security group %s: %w", instance.Name, err)
	}

	return nil
}
