package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/ataiva-software/runestone/internal/config"
)

// Provider implements the AWS provider
type Provider struct {
	awsConfig aws.Config
	s3Client  *s3.Client
	ec2Client *ec2.Client
	rdsClient *rds.Client
	iamClient *iam.Client
	stsClient *sts.Client
	region    string
}

// retryConfig defines retry behavior
type retryConfig struct {
	maxRetries int
	baseDelay  time.Duration
}

// defaultRetryConfig returns the default retry configuration
func defaultRetryConfig() retryConfig {
	return retryConfig{
		maxRetries: 3,
		baseDelay:  time.Second,
	}
}

// isResourceNotFound checks if an error indicates a resource was not found
func isResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	// Check for common AWS "not found" error patterns
	return strings.Contains(errStr, "NoSuchBucket") ||
		strings.Contains(errStr, "NoSuchUser") ||
		strings.Contains(errStr, "NoSuchRole") ||
		strings.Contains(errStr, "NoSuchEntity") ||
		strings.Contains(errStr, "NoSuchPolicy") ||
		strings.Contains(errStr, "InvalidInstanceID.NotFound") ||
		strings.Contains(errStr, "DBInstanceNotFound") ||
		strings.Contains(errStr, "NotFound") ||
		strings.Contains(errStr, "does not exist")
}

// retryWithBackoff executes a function with exponential backoff retry
func (p *Provider) retryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	config := defaultRetryConfig()
	
	for attempt := 0; attempt <= config.maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			return fmt.Errorf("%s failed (non-retryable): %w", operation, err)
		}

		if attempt == config.maxRetries {
			return fmt.Errorf("%s failed after %d attempts: %w", operation, config.maxRetries+1, err)
		}

		// Calculate delay with exponential backoff
		delay := config.baseDelay * time.Duration(1<<attempt)
		fmt.Printf("  Retrying %s in %v (attempt %d/%d)...\n", operation, delay, attempt+2, config.maxRetries+1)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return nil // Should never reach here
}

// isNonRetryableError determines if an error should not be retried
func isNonRetryableError(err error) bool {
	errStr := err.Error()
	
	// Don't retry authentication errors
	if strings.Contains(errStr, "AuthFailure") || strings.Contains(errStr, "InvalidUserID.NotFound") {
		return true
	}
	
	// Don't retry validation errors
	if strings.Contains(errStr, "ValidationException") || strings.Contains(errStr, "InvalidParameterValue") {
		return true
	}
	
	// Don't retry resource already exists errors
	if strings.Contains(errStr, "BucketAlreadyExists") || strings.Contains(errStr, "BucketAlreadyOwnedByYou") {
		return true
	}
	
	return false
}

// RDS Instance operations

func (p *Provider) createRDSInstance(ctx context.Context, instance config.ResourceInstance) error {
	dbInstanceIdentifier := instance.Name

	// Required parameters
	dbInstanceClass, ok := instance.Properties["db_instance_class"].(string)
	if !ok {
		return fmt.Errorf("db_instance_class is required for RDS instance")
	}

	engine, ok := instance.Properties["engine"].(string)
	if !ok {
		return fmt.Errorf("engine is required for RDS instance")
	}

	masterUsername, ok := instance.Properties["master_username"].(string)
	if !ok {
		return fmt.Errorf("master_username is required for RDS instance")
	}

	masterUserPassword, ok := instance.Properties["master_user_password"].(string)
	if !ok {
		return fmt.Errorf("master_user_password is required for RDS instance")
	}

	// Optional parameters with defaults
	allocatedStorage := int32(20)
	if storage, ok := instance.Properties["allocated_storage"].(int); ok {
		allocatedStorage = int32(storage)
	}

	input := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
		DBInstanceClass:      aws.String(dbInstanceClass),
		Engine:               aws.String(engine),
		MasterUsername:       aws.String(masterUsername),
		MasterUserPassword:   aws.String(masterUserPassword),
		AllocatedStorage:     aws.Int32(allocatedStorage),
	}

	// Optional parameters
	if dbName, ok := instance.Properties["db_name"].(string); ok {
		input.DBName = aws.String(dbName)
	}

	if engineVersion, ok := instance.Properties["engine_version"].(string); ok {
		input.EngineVersion = aws.String(engineVersion)
	}

	if backupRetentionPeriod, ok := instance.Properties["backup_retention_period"].(int); ok {
		input.BackupRetentionPeriod = aws.Int32(int32(backupRetentionPeriod))
	}

	// Add tags if specified
	if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok && len(tags) > 0 {
		var tagList []rdstypes.Tag
		for key, value := range tags {
			tagList = append(tagList, rdstypes.Tag{
				Key:   aws.String(key),
				Value: aws.String(fmt.Sprintf("%v", value)),
			})
		}
		input.Tags = tagList
	}

	// Create RDS instance with retry
	err := p.retryWithBackoff(ctx, fmt.Sprintf("create RDS instance %s", dbInstanceIdentifier), func() error {
		_, err := p.rdsClient.CreateDBInstance(ctx, input)
		return err
	})

	return err
}

func (p *Provider) updateRDSInstance(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error {
	dbInstanceIdentifier := instance.Name

	input := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
		ApplyImmediately:     aws.Bool(true),
	}

	// Check for changes that can be modified
	if dbInstanceClass, ok := instance.Properties["db_instance_class"].(string); ok {
		if currentClass, exists := currentState["db_instance_class"]; !exists || currentClass != dbInstanceClass {
			input.DBInstanceClass = aws.String(dbInstanceClass)
		}
	}

	if allocatedStorage, ok := instance.Properties["allocated_storage"].(int); ok {
		if currentStorage, exists := currentState["allocated_storage"]; !exists || currentStorage != allocatedStorage {
			input.AllocatedStorage = aws.Int32(int32(allocatedStorage))
		}
	}

	if backupRetentionPeriod, ok := instance.Properties["backup_retention_period"].(int); ok {
		if currentRetention, exists := currentState["backup_retention_period"]; !exists || currentRetention != backupRetentionPeriod {
			input.BackupRetentionPeriod = aws.Int32(int32(backupRetentionPeriod))
		}
	}

	// Update RDS instance with retry
	err := p.retryWithBackoff(ctx, fmt.Sprintf("update RDS instance %s", dbInstanceIdentifier), func() error {
		_, err := p.rdsClient.ModifyDBInstance(ctx, input)
		return err
	})

	return err
}

func (p *Provider) deleteRDSInstance(ctx context.Context, instance config.ResourceInstance) error {
	dbInstanceIdentifier := instance.Name

	input := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   aws.String(dbInstanceIdentifier),
		SkipFinalSnapshot:      aws.Bool(true),
		DeleteAutomatedBackups: aws.Bool(true),
	}

	// Delete RDS instance with retry
	err := p.retryWithBackoff(ctx, fmt.Sprintf("delete RDS instance %s", dbInstanceIdentifier), func() error {
		_, err := p.rdsClient.DeleteDBInstance(ctx, input)
		return err
	})

	return err
}

func (p *Provider) getRDSInstanceState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	dbInstanceIdentifier := instance.Name

	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
	}

	var result *rds.DescribeDBInstancesOutput
	err := p.retryWithBackoff(ctx, fmt.Sprintf("describe RDS instance %s", dbInstanceIdentifier), func() error {
		var err error
		result, err = p.rdsClient.DescribeDBInstances(ctx, input)
		return err
	})

	if err != nil {
		// Check if instance doesn't exist
		if strings.Contains(err.Error(), "DBInstanceNotFound") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to describe RDS instance: %w", err)
	}

	if len(result.DBInstances) == 0 {
		return nil, nil
	}

	dbInstance := result.DBInstances[0]
	state := map[string]interface{}{
		"db_instance_identifier": aws.ToString(dbInstance.DBInstanceIdentifier),
		"db_instance_class":      aws.ToString(dbInstance.DBInstanceClass),
		"engine":                 aws.ToString(dbInstance.Engine),
		"engine_version":         aws.ToString(dbInstance.EngineVersion),
		"db_instance_status":     aws.ToString(dbInstance.DBInstanceStatus),
		"allocated_storage":      aws.ToInt32(dbInstance.AllocatedStorage),
		"master_username":        aws.ToString(dbInstance.MasterUsername),
	}

	if dbInstance.DBName != nil {
		state["db_name"] = aws.ToString(dbInstance.DBName)
	}

	if dbInstance.BackupRetentionPeriod != nil {
		state["backup_retention_period"] = aws.ToInt32(dbInstance.BackupRetentionPeriod)
	}

	// Add tags
	if len(dbInstance.TagList) > 0 {
		tags := make(map[string]interface{})
		for _, tag := range dbInstance.TagList {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		state["tags"] = tags
	}

	return state, nil
}

func (p *Provider) validateRDSInstance(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("RDS instance name cannot be empty")
	}

	// Validate required properties
	if _, ok := instance.Properties["db_instance_class"]; !ok {
		return fmt.Errorf("db_instance_class is required for RDS instance")
	}

	if _, ok := instance.Properties["engine"]; !ok {
		return fmt.Errorf("engine is required for RDS instance")
	}

	if _, ok := instance.Properties["master_username"]; !ok {
		return fmt.Errorf("master_username is required for RDS instance")
	}

	if _, ok := instance.Properties["master_user_password"]; !ok {
		return fmt.Errorf("master_user_password is required for RDS instance")
	}

	// Validate engine type
	if engine, ok := instance.Properties["engine"].(string); ok {
		validEngines := []string{"mysql", "postgres", "mariadb", "oracle-ee", "oracle-se2", "sqlserver-ex", "sqlserver-web", "sqlserver-se", "sqlserver-ee"}
		valid := false
		for _, validEngine := range validEngines {
			if engine == validEngine {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid engine type: %s", engine)
		}
	}

	return nil
}

// NewProvider creates a new AWS provider
func NewProvider() *Provider {
	return &Provider{}
}

// Initialize sets up the AWS provider with configuration
func (p *Provider) Initialize(ctx context.Context, providerConfig map[string]interface{}) error {
	// Extract region and profile from config
	region, _ := providerConfig["region"].(string)
	if region == "" {
		region = "us-east-1" // default region
	}
	p.region = region

	profile, _ := providerConfig["profile"].(string)

	// Load AWS configuration with timeout - but don't make any network calls
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(region))
	if profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(profile))
	}

	// Create a context with timeout for AWS config loading
	configCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(configCtx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config (region: %s, profile: %s): %w", region, profile, err)
	}

	p.awsConfig = cfg
	p.s3Client = s3.NewFromConfig(cfg)
	p.ec2Client = ec2.NewFromConfig(cfg)
	p.rdsClient = rds.NewFromConfig(cfg)
	p.iamClient = iam.NewFromConfig(cfg)
	p.stsClient = sts.NewFromConfig(cfg)

	return nil
}

// Create creates a new AWS resource
func (p *Provider) Create(ctx context.Context, instance config.ResourceInstance) error {
	switch instance.Kind {
	case "aws:s3:bucket":
		return p.createS3Bucket(ctx, instance)
	case "aws:ec2:instance":
		return p.createEC2Instance(ctx, instance)
	case "aws:ec2:vpc":
		return p.createVPC(ctx, instance)
	case "aws:ec2:subnet":
		return p.createSubnet(ctx, instance)
	case "aws:ec2:internet_gateway":
		return p.createInternetGateway(ctx, instance)
	case "aws:ec2:security_group":
		return p.createSecurityGroup(ctx, instance)
	case "aws:lambda:function":
		return p.createLambdaFunction(ctx, instance)
	case "aws:dynamodb:table":
		return p.createDynamoDBTable(ctx, instance)
	case "aws:apigateway:rest_api":
		return p.createAPIGateway(ctx, instance)
	case "aws:rds:instance":
		return p.createRDSInstance(ctx, instance)
	case "aws:iam:user":
		return p.createIAMUser(ctx, instance)
	case "aws:iam:role":
		return p.createIAMRole(ctx, instance)
	case "aws:iam:policy":
		return p.createIAMPolicy(ctx, instance)
	default:
		return fmt.Errorf("unsupported resource type: %s", instance.Kind)
	}
}

// Update updates an existing AWS resource
func (p *Provider) Update(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error {
	switch instance.Kind {
	case "aws:s3:bucket":
		return p.updateS3Bucket(ctx, instance, currentState)
	case "aws:ec2:instance":
		return p.updateEC2Instance(ctx, instance, currentState)
	case "aws:ec2:vpc":
		return p.updateVPC(ctx, instance)
	case "aws:ec2:subnet":
		return p.updateSubnet(ctx, instance)
	case "aws:ec2:internet_gateway":
		return p.updateInternetGateway(ctx, instance)
	case "aws:ec2:security_group":
		return p.updateSecurityGroup(ctx, instance)
	case "aws:lambda:function":
		return p.updateLambdaFunction(ctx, instance)
	case "aws:dynamodb:table":
		return p.updateDynamoDBTable(ctx, instance)
	case "aws:apigateway:rest_api":
		return p.updateAPIGateway(ctx, instance)
	case "aws:rds:instance":
		return p.updateRDSInstance(ctx, instance, currentState)
	case "aws:iam:user":
		return p.updateIAMUser(ctx, instance)
	case "aws:iam:role":
		return p.updateIAMRole(ctx, instance)
	case "aws:iam:policy":
		return p.updateIAMPolicy(ctx, instance)
	default:
		return fmt.Errorf("unsupported resource type: %s", instance.Kind)
	}
}

// Delete deletes an AWS resource
func (p *Provider) Delete(ctx context.Context, instance config.ResourceInstance) error {
	switch instance.Kind {
	case "aws:s3:bucket":
		return p.deleteS3Bucket(ctx, instance)
	case "aws:ec2:instance":
		return p.deleteEC2Instance(ctx, instance)
	case "aws:ec2:vpc":
		return p.deleteVPC(ctx, instance)
	case "aws:ec2:subnet":
		return p.deleteSubnet(ctx, instance)
	case "aws:ec2:internet_gateway":
		return p.deleteInternetGateway(ctx, instance)
	case "aws:ec2:security_group":
		return p.deleteSecurityGroup(ctx, instance)
	case "aws:lambda:function":
		return p.deleteLambdaFunction(ctx, instance)
	case "aws:dynamodb:table":
		return p.deleteDynamoDBTable(ctx, instance)
	case "aws:apigateway:rest_api":
		return p.deleteAPIGateway(ctx, instance)
	case "aws:rds:instance":
		return p.deleteRDSInstance(ctx, instance)
	case "aws:iam:user":
		return p.deleteIAMUser(ctx, instance)
	case "aws:iam:role":
		return p.deleteIAMRole(ctx, instance)
	case "aws:iam:policy":
		return p.deleteIAMPolicy(ctx, instance)
	default:
		return fmt.Errorf("unsupported resource type: %s", instance.Kind)
	}
}

// GetCurrentState retrieves the current state of an AWS resource
func (p *Provider) GetCurrentState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	switch instance.Kind {
	case "aws:s3:bucket":
		return p.getS3BucketState(ctx, instance)
	case "aws:ec2:instance":
		return p.getEC2InstanceState(ctx, instance)
	case "aws:ec2:vpc":
		return p.getVPCState(ctx, instance)
	case "aws:ec2:subnet":
		return p.getSubnetState(ctx, instance)
	case "aws:ec2:internet_gateway":
		return p.getInternetGatewayState(ctx, instance)
	case "aws:ec2:security_group":
		return p.getSecurityGroupState(ctx, instance)
	case "aws:lambda:function":
		return p.getLambdaFunctionState(ctx, instance)
	case "aws:dynamodb:table":
		return p.getDynamoDBTableState(ctx, instance)
	case "aws:apigateway:rest_api":
		return p.getAPIGatewayState(ctx, instance)
	case "aws:rds:instance":
		return p.getRDSInstanceState(ctx, instance)
	case "aws:iam:user":
		return p.getIAMUserState(ctx, instance)
	case "aws:iam:role":
		return p.getIAMRoleState(ctx, instance)
	case "aws:iam:policy":
		return p.getIAMPolicyState(ctx, instance)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", instance.Kind)
	}
}

// ValidateResource validates an AWS resource configuration
func (p *Provider) ValidateResource(instance config.ResourceInstance) error {
	switch instance.Kind {
	case "aws:s3:bucket":
		return p.validateS3Bucket(instance)
	case "aws:ec2:instance":
		return p.validateEC2Instance(instance)
	case "aws:ec2:vpc":
		return p.validateVPC(instance)
	case "aws:ec2:subnet":
		return p.validateSubnet(instance)
	case "aws:ec2:internet_gateway":
		return p.validateInternetGateway(instance)
	case "aws:ec2:security_group":
		return p.validateSecurityGroup(instance)
	case "aws:lambda:function":
		return p.validateLambdaFunction(instance)
	case "aws:dynamodb:table":
		return p.validateDynamoDBTable(instance)
	case "aws:apigateway:rest_api":
		return p.validateAPIGateway(instance)
	case "aws:rds:instance":
		return p.validateRDSInstance(instance)
	case "aws:iam:user":
		return p.validateIAMUser(instance)
	case "aws:iam:role":
		return p.validateIAMRole(instance)
	case "aws:iam:policy":
		return p.validateIAMPolicy(instance)
	default:
		return fmt.Errorf("unsupported resource type: %s", instance.Kind)
	}
}

// GetSupportedResourceTypes returns the AWS resource types supported
func (p *Provider) GetSupportedResourceTypes() []string {
	return []string{
		"aws:s3:bucket",
		"aws:ec2:instance",
		"aws:ec2:vpc",
		"aws:ec2:subnet",
		"aws:ec2:internet_gateway",
		"aws:ec2:security_group",
		"aws:lambda:function",
		"aws:dynamodb:table",
		"aws:apigateway:rest_api",
		"aws:rds:instance",
		"aws:iam:user",
		"aws:iam:role",
		"aws:iam:policy",
	}
}

// S3 Bucket operations

func (p *Provider) createS3Bucket(ctx context.Context, instance config.ResourceInstance) error {
	bucketName := instance.Name

	// Create bucket with retry
	err := p.retryWithBackoff(ctx, fmt.Sprintf("create S3 bucket %s", bucketName), func() error {
		_, err := p.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
		return err
	})
	if err != nil {
		return err
	}

	// Configure versioning if specified
	if versioning, ok := instance.Properties["versioning"].(bool); ok && versioning {
		err = p.retryWithBackoff(ctx, fmt.Sprintf("enable versioning for S3 bucket %s", bucketName), func() error {
			_, err := p.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
				Bucket: aws.String(bucketName),
				VersioningConfiguration: &s3types.VersioningConfiguration{
					Status: s3types.BucketVersioningStatusEnabled,
				},
			})
			return err
		})
		if err != nil {
			return err
		}
	}

	// Apply tags if specified
	if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok && len(tags) > 0 {
		tagSet := make([]s3types.Tag, 0, len(tags))
		for key, value := range tags {
			tagSet = append(tagSet, s3types.Tag{
				Key:   aws.String(key),
				Value: aws.String(fmt.Sprintf("%v", value)),
			})
		}

		err = p.retryWithBackoff(ctx, fmt.Sprintf("apply tags to S3 bucket %s", bucketName), func() error {
			_, err := p.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
				Bucket: aws.String(bucketName),
				Tagging: &s3types.Tagging{
					TagSet: tagSet,
				},
			})
			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) updateS3Bucket(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error {
	bucketName := instance.Name

	// Update versioning if changed
	if versioning, ok := instance.Properties["versioning"].(bool); ok {
		currentVersioning, _ := currentState["versioning"].(bool)
		if versioning != currentVersioning {
			status := s3types.BucketVersioningStatusSuspended
			if versioning {
				status = s3types.BucketVersioningStatusEnabled
			}

			_, err := p.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
				Bucket: aws.String(bucketName),
				VersioningConfiguration: &s3types.VersioningConfiguration{
					Status: status,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update versioning for S3 bucket %s: %w", bucketName, err)
			}
		}
	}

	// Update tags if changed
	if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok {
		tagSet := make([]s3types.Tag, 0, len(tags))
		for key, value := range tags {
			tagSet = append(tagSet, s3types.Tag{
				Key:   aws.String(key),
				Value: aws.String(fmt.Sprintf("%v", value)),
			})
		}

		_, err := p.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
			Bucket: aws.String(bucketName),
			Tagging: &s3types.Tagging{
				TagSet: tagSet,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update tags for S3 bucket %s: %w", bucketName, err)
		}
	}

	return nil
}

func (p *Provider) deleteS3Bucket(ctx context.Context, instance config.ResourceInstance) error {
	bucketName := instance.Name

	_, err := p.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 bucket %s: %w", bucketName, err)
	}

	return nil
}

func (p *Provider) getS3BucketState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	bucketName := instance.Name
	state := make(map[string]interface{})

	// Check if bucket exists
	_, err := p.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Bucket doesn't exist
		return nil, nil
	}

	state["name"] = bucketName

	// Get versioning status
	versioningOutput, err := p.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		state["versioning"] = versioningOutput.Status == s3types.BucketVersioningStatusEnabled
	}

	// Get tags
	taggingOutput, err := p.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && len(taggingOutput.TagSet) > 0 {
		tags := make(map[string]interface{})
		for _, tag := range taggingOutput.TagSet {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
		state["tags"] = tags
	}

	return state, nil
}

func (p *Provider) validateS3Bucket(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("S3 bucket name is required")
	}

	// Validate bucket name format (simplified)
	if len(instance.Name) < 3 || len(instance.Name) > 63 {
		return fmt.Errorf("S3 bucket name must be between 3 and 63 characters")
	}

	if strings.Contains(instance.Name, "_") {
		return fmt.Errorf("S3 bucket name cannot contain underscores")
	}

	return nil
}

// EC2 Instance operations (simplified implementation)

func (p *Provider) createEC2Instance(ctx context.Context, instance config.ResourceInstance) error {
	instanceType, ok := instance.Properties["instance_type"].(string)
	if !ok {
		return fmt.Errorf("instance_type is required for EC2 instance")
	}

	ami, ok := instance.Properties["ami"].(string)
	if !ok {
		return fmt.Errorf("ami is required for EC2 instance")
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	// Add tags if specified
	if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok && len(tags) > 0 {
		tagSpecs := make([]types.TagSpecification, 1)
		tagSpecs[0] = types.TagSpecification{
			ResourceType: types.ResourceTypeInstance,
			Tags:         make([]types.Tag, 0, len(tags)),
		}

		for key, value := range tags {
			tagSpecs[0].Tags = append(tagSpecs[0].Tags, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(fmt.Sprintf("%v", value)),
			})
		}

		input.TagSpecifications = tagSpecs
	}

	_, err := p.ec2Client.RunInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create EC2 instance %s: %w", instance.Name, err)
	}

	return nil
}

func (p *Provider) updateEC2Instance(ctx context.Context, instance config.ResourceInstance, currentState map[string]interface{}) error {
	// EC2 instance updates are limited - for now, just handle tags
	instanceID, ok := currentState["instance_id"].(string)
	if !ok {
		return fmt.Errorf("instance_id not found in current state")
	}

	if tags, ok := instance.Properties["tags"].(map[string]interface{}); ok {
		tagList := make([]types.Tag, 0, len(tags))
		for key, value := range tags {
			tagList = append(tagList, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(fmt.Sprintf("%v", value)),
			})
		}

		_, err := p.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{instanceID},
			Tags:      tagList,
		})
		if err != nil {
			return fmt.Errorf("failed to update tags for EC2 instance %s: %w", instanceID, err)
		}
	}

	return nil
}

func (p *Provider) deleteEC2Instance(ctx context.Context, instance config.ResourceInstance) error {
	// First, get the current state to find the instance ID
	state, err := p.getEC2InstanceState(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to get instance state for deletion: %w", err)
	}

	// If instance doesn't exist, consider it already deleted
	if state == nil {
		return nil
	}

	instanceID, ok := state["instance_id"].(string)
	if !ok {
		return fmt.Errorf("instance_id not found in state")
	}

	// Terminate the instance
	_, err = p.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate EC2 instance %s: %w", instanceID, err)
	}

	return nil
}

func (p *Provider) getEC2InstanceState(ctx context.Context, instance config.ResourceInstance) (map[string]interface{}, error) {
	instanceName := instance.Name
	
	// Find instances by Name tag
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{instanceName},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "pending", "stopping", "stopped"},
			},
		},
	}

	result, err := p.ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	// Check if any instances were found
	var foundInstance *types.Instance
	for _, reservation := range result.Reservations {
		for _, inst := range reservation.Instances {
			// Double-check the Name tag matches exactly
			for _, tag := range inst.Tags {
				if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil && *tag.Value == instanceName {
					foundInstance = &inst
					break
				}
			}
			if foundInstance != nil {
				break
			}
		}
		if foundInstance != nil {
			break
		}
	}

	// If no instance found, return nil (resource doesn't exist)
	if foundInstance == nil {
		return nil, nil
	}

	// Build state map
	state := make(map[string]interface{})
	state["instance_id"] = *foundInstance.InstanceId
	state["instance_type"] = string(foundInstance.InstanceType)
	state["ami"] = *foundInstance.ImageId
	state["state"] = string(foundInstance.State.Name)

	// Extract tags
	if len(foundInstance.Tags) > 0 {
		tags := make(map[string]interface{})
		for _, tag := range foundInstance.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
		state["tags"] = tags
	}

	// Add other useful information
	if foundInstance.PublicIpAddress != nil {
		state["public_ip"] = *foundInstance.PublicIpAddress
	}
	if foundInstance.PrivateIpAddress != nil {
		state["private_ip"] = *foundInstance.PrivateIpAddress
	}
	if foundInstance.LaunchTime != nil {
		state["launch_time"] = foundInstance.LaunchTime.Format("2006-01-02T15:04:05Z")
	}

	return state, nil
}

func (p *Provider) validateEC2Instance(instance config.ResourceInstance) error {
	if instance.Name == "" {
		return fmt.Errorf("EC2 instance name is required")
	}

	if _, ok := instance.Properties["instance_type"]; !ok {
		return fmt.Errorf("instance_type is required for EC2 instance")
	}

	if _, ok := instance.Properties["ami"]; !ok {
		return fmt.Errorf("ami is required for EC2 instance")
	}

	return nil
}
