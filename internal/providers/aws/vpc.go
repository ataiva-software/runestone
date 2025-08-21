package aws

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ataiva-software/runestone/internal/config"
)

// validateVPC validates VPC configuration
func (p *Provider) validateVPC(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("VPC name cannot be empty")
	}

	// Validate CIDR block
	cidrVal, exists := instance.Properties["cidr_block"]
	if !exists {
		return fmt.Errorf("cidr_block is required for VPC")
	}

	cidr, ok := cidrVal.(string)
	if !ok {
		return fmt.Errorf("cidr_block must be a string")
	}

	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("invalid CIDR block '%s': %w", cidr, err)
	}

	return nil
}

// getVPCState retrieves the current state of a VPC
func (p *Provider) getVPCState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := ec2.NewFromConfig(p.awsConfig)

	// Find VPC by name tag
	input := &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{instance.Name},
			},
		},
	}

	result, err := client.DescribeVpcs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC %s: %w", instance.Name, err)
	}

	if len(result.Vpcs) == 0 {
		return nil, nil // VPC doesn't exist
	}

	vpc := result.Vpcs[0]

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range vpc.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	state := map[string]interface{}{
		"vpc_id":     *vpc.VpcId,
		"cidr_block": *vpc.CidrBlock,
		"state":      string(vpc.State),
		"tags":       tags,
	}

	return state, nil
}

// createVPC creates a new VPC
func (p *Provider) createVPC(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	cidr := instance.Properties["cidr_block"].(string)

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(cidr),
	}

	result, err := client.CreateVpc(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create VPC %s: %w", instance.Name, err)
	}

	vpcId := *result.Vpc.VpcId

	// Add Name tag
	nameTagInput := &ec2.CreateTagsInput{
		Resources: []string{vpcId},
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instance.Name),
			},
		},
	}

	_, err = client.CreateTags(ctx, nameTagInput)
	if err != nil {
		return fmt.Errorf("failed to tag VPC %s: %w", instance.Name, err)
	}

	// Add additional tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{vpcId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to add tags to VPC %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteVPC deletes a VPC
func (p *Provider) deleteVPC(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	// Get VPC ID by name
	state, err := p.getVPCState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return nil // VPC already deleted
	}

	vpcId := state["vpc_id"].(string)

	input := &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcId),
	}

	_, err = client.DeleteVpc(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete VPC %s: %w", instance.Name, err)
	}

	return nil
}

// validateSubnet validates subnet configuration
func (p *Provider) validateSubnet(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("subnet name cannot be empty")
	}

	// Validate VPC ID
	vpcIdVal, exists := instance.Properties["vpc_id"]
	if !exists {
		return fmt.Errorf("vpc_id is required for subnet")
	}

	vpcId, ok := vpcIdVal.(string)
	if !ok {
		return fmt.Errorf("vpc_id must be a string")
	}

	if !strings.HasPrefix(vpcId, "vpc-") {
		return fmt.Errorf("invalid vpc_id format: %s", vpcId)
	}

	// Validate CIDR block
	cidrVal, exists := instance.Properties["cidr_block"]
	if !exists {
		return fmt.Errorf("cidr_block is required for subnet")
	}

	cidr, ok := cidrVal.(string)
	if !ok {
		return fmt.Errorf("cidr_block must be a string")
	}

	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("invalid CIDR block '%s': %w", cidr, err)
	}

	return nil
}

// getSubnetState retrieves the current state of a subnet
func (p *Provider) getSubnetState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := ec2.NewFromConfig(p.awsConfig)

	// Find subnet by name tag
	input := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{instance.Name},
			},
		},
	}

	result, err := client.DescribeSubnets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnet %s: %w", instance.Name, err)
	}

	if len(result.Subnets) == 0 {
		return nil, nil // Subnet doesn't exist
	}

	subnet := result.Subnets[0]

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range subnet.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	state := map[string]interface{}{
		"subnet_id":         *subnet.SubnetId,
		"vpc_id":            *subnet.VpcId,
		"cidr_block":        *subnet.CidrBlock,
		"availability_zone": *subnet.AvailabilityZone,
		"state":             string(subnet.State),
		"tags":              tags,
	}

	return state, nil
}

// createSubnet creates a new subnet
func (p *Provider) createSubnet(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	vpcId := instance.Properties["vpc_id"].(string)
	cidr := instance.Properties["cidr_block"].(string)

	input := &ec2.CreateSubnetInput{
		VpcId:     aws.String(vpcId),
		CidrBlock: aws.String(cidr),
	}

	// Add availability zone if specified
	if azVal, exists := instance.Properties["availability_zone"]; exists {
		if az, ok := azVal.(string); ok {
			input.AvailabilityZone = aws.String(az)
		}
	}

	result, err := client.CreateSubnet(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create subnet %s: %w", instance.Name, err)
	}

	subnetId := *result.Subnet.SubnetId

	// Add Name tag
	nameTagInput := &ec2.CreateTagsInput{
		Resources: []string{subnetId},
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instance.Name),
			},
		},
	}

	_, err = client.CreateTags(ctx, nameTagInput)
	if err != nil {
		return fmt.Errorf("failed to tag subnet %s: %w", instance.Name, err)
	}

	// Add additional tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{subnetId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to add tags to subnet %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteSubnet deletes a subnet
func (p *Provider) deleteSubnet(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	// Get subnet ID by name
	state, err := p.getSubnetState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return nil // Subnet already deleted
	}

	subnetId := state["subnet_id"].(string)

	input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(subnetId),
	}

	_, err = client.DeleteSubnet(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete subnet %s: %w", instance.Name, err)
	}

	return nil
}

// validateInternetGateway validates internet gateway configuration
func (p *Provider) validateInternetGateway(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("internet gateway name cannot be empty")
	}
	return nil
}

// getInternetGatewayState retrieves the current state of an internet gateway
func (p *Provider) getInternetGatewayState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	client := ec2.NewFromConfig(p.awsConfig)

	// Find internet gateway by name tag
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{instance.Name},
			},
		},
	}

	result, err := client.DescribeInternetGateways(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe internet gateway %s: %w", instance.Name, err)
	}

	if len(result.InternetGateways) == 0 {
		return nil, nil // Internet gateway doesn't exist
	}

	igw := result.InternetGateways[0]

	// Convert tags to map
	tags := make(map[string]interface{})
	for _, tag := range igw.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	state := map[string]interface{}{
		"internet_gateway_id": *igw.InternetGatewayId,
		"tags":                tags,
	}

	return state, nil
}

// createInternetGateway creates a new internet gateway
func (p *Provider) createInternetGateway(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	input := &ec2.CreateInternetGatewayInput{}

	result, err := client.CreateInternetGateway(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create internet gateway %s: %w", instance.Name, err)
	}

	igwId := *result.InternetGateway.InternetGatewayId

	// Add Name tag
	nameTagInput := &ec2.CreateTagsInput{
		Resources: []string{igwId},
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instance.Name),
			},
		},
	}

	_, err = client.CreateTags(ctx, nameTagInput)
	if err != nil {
		return fmt.Errorf("failed to tag internet gateway %s: %w", instance.Name, err)
	}

	// Add additional tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{igwId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to add tags to internet gateway %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// deleteInternetGateway deletes an internet gateway
func (p *Provider) deleteInternetGateway(ctx context.Context, instance config.ResourceInstance) error {
	client := ec2.NewFromConfig(p.awsConfig)

	// Get internet gateway ID by name
	state, err := p.getInternetGatewayState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return nil // Internet gateway already deleted
	}

	igwId := state["internet_gateway_id"].(string)

	input := &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwId),
	}

	_, err = client.DeleteInternetGateway(ctx, input)
	if err != nil {
		if isResourceNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete internet gateway %s: %w", instance.Name, err)
	}

	return nil
}

// updateVPC updates an existing VPC (mainly tags)
func (p *Provider) updateVPC(ctx context.Context, instance config.ResourceInstance) error {
	// VPCs can only have their tags updated
	state, err := p.getVPCState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("VPC %s not found", instance.Name)
	}

	vpcId := state["vpc_id"].(string)

	// Update tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{vpcId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for VPC %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// updateSubnet updates an existing subnet (mainly tags)
func (p *Provider) updateSubnet(ctx context.Context, instance config.ResourceInstance) error {
	// Subnets can only have their tags updated
	state, err := p.getSubnetState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("subnet %s not found", instance.Name)
	}

	subnetId := state["subnet_id"].(string)

	// Update tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{subnetId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for subnet %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}

// updateInternetGateway updates an existing internet gateway (mainly tags)
func (p *Provider) updateInternetGateway(ctx context.Context, instance config.ResourceInstance) error {
	// Internet gateways can only have their tags updated
	state, err := p.getInternetGatewayState(ctx, instance)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("internet gateway %s not found", instance.Name)
	}

	igwId := state["internet_gateway_id"].(string)

	// Update tags if specified
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
				tagInput := &ec2.CreateTagsInput{
					Resources: []string{igwId},
					Tags:      tags,
				}
				_, err = client.CreateTags(ctx, tagInput)
				if err != nil {
					return fmt.Errorf("failed to update tags for internet gateway %s: %w", instance.Name, err)
				}
			}
		}
	}

	return nil
}
