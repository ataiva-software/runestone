# Examples

**Generated on: 2025-08-21 16:44:46 UTC**

This document provides practical examples of Runestone configurations for common use cases.

## Basic Web Application

A simple web application with load balancer and database.

```yaml
project: webapp
environment: production
variables:
  region: us-east-1
  tags:
    owner: devops-team
    project: webapp

providers:
  aws:
    region: "${region}"

resources:
  # Application logs
  - kind: aws:s3:bucket
    name: "${project}-logs-${environment}"
    properties:
      versioning: true
      tags: "${tags}"
    driftPolicy:
      autoHeal: true

  # Web servers
  - kind: aws:ec2:instance
    name: web-${index}
    count: 2
    properties:
      instance_type: t3.medium
      ami: ami-0abcdef1234567890
      tags:
        Name: "web-${index}"
        Role: web-server
    driftPolicy:
      autoHeal: true
```

## VPC Networking

Complete VPC setup with public and private subnets.

```yaml
project: vpc-app
environment: production
variables:
  region: us-east-1
  vpc_cidr: "10.0.0.0/16"
  tags:
    owner: platform-team
    Environment: production

providers:
  aws:
    region: "${region}"

resources:
  # Main VPC
  - kind: aws:ec2:vpc
    name: app-vpc
    properties:
      cidr_block: "${vpc_cidr}"
      tags: "${tags}"
    driftPolicy:
      autoHeal: true

  # Internet Gateway
  - kind: aws:ec2:internet_gateway
    name: app-igw
    properties:
      tags: "${tags}"
    driftPolicy:
      autoHeal: true

  # Public subnet for web tier
  - kind: aws:ec2:subnet
    name: public-subnet-1a
    properties:
      vpc_id: "vpc-placeholder"  # Resolved at runtime
      cidr_block: "10.0.1.0/24"
      availability_zone: "us-east-1a"
      tags:
        Name: public-subnet-1a
        Tier: public
    depends_on:
      - "aws:ec2:vpc.app-vpc"
    driftPolicy:
      autoHeal: true

  # Private subnet for app tier
  - kind: aws:ec2:subnet
    name: private-subnet-1a
    properties:
      vpc_id: "vpc-placeholder"  # Resolved at runtime
      cidr_block: "10.0.2.0/24"
      availability_zone: "us-east-1a"
      tags:
        Name: private-subnet-1a
        Tier: private
    depends_on:
      - "aws:ec2:vpc.app-vpc"
    driftPolicy:
      autoHeal: true

  # Application data bucket
  - kind: aws:s3:bucket
    name: "${project}-data-${environment}"
    properties:
      versioning: true
      tags: "${tags}"
    driftPolicy:
      autoHeal: true
```

## Serverless Application

Complete serverless setup with Lambda functions and IAM roles.

```yaml
project: serverless-app
environment: production
variables:
  region: us-east-1
  tags:
    owner: serverless-team
    Environment: production

providers:
  aws:
    region: "${region}"

resources:
  # Lambda execution role
  - kind: aws:iam:role
    name: lambda-execution-role
    properties:
      assume_role_policy: |
        {
          "Version": "2012-10-17",
          "Statement": [
            {
              "Effect": "Allow",
              "Principal": {"Service": "lambda.amazonaws.com"},
              "Action": "sts:AssumeRole"
            }
          ]
        }
      path: "/service-roles/"
      tags: "${tags}"
    driftPolicy:
      autoHeal: true

  # API Lambda function
  - kind: aws:lambda:function
    name: api-handler
    properties:
      runtime: "python3.9"
      handler: "api.handler"
      role: "arn:aws:iam::123456789012:role/service-roles/lambda-execution-role"
      timeout: 30
      memory_size: 256
      code_content: |
        import json
        def handler(event, context):
            return {
                'statusCode': 200,
                'body': json.dumps({'message': 'Hello from API!'})
            }
      tags: "${tags}"
    depends_on:
      - "aws:iam:role.lambda-execution-role"
    driftPolicy:
      autoHeal: true

  # Data storage bucket
  - kind: aws:s3:bucket
    name: "${project}-data-${environment}"
    properties:
      versioning: true
      tags: "${tags}"
    driftPolicy:
      autoHeal: true
```

## Multi-Environment Setup

Configuration that adapts based on environment.

```yaml
project: api-service
environment: "${ENV:-dev}"  # Use ENV var or default to dev
variables:
  region: us-west-2
  # Environment-specific settings
  instance_type: "${environment == 'prod' ? 't3.large' : 't3.micro'}"
  instance_count: "${environment == 'prod' ? 5 : 2}"
  backup_enabled: "${environment == 'prod'}"
  
  tags:
    owner: backend-team
    environment: "${environment}"

providers:
  aws:
    region: "${region}"

resources:
  # Data storage
  - kind: aws:s3:bucket
    name: "${project}-data-${environment}"
    properties:
      versioning: "${backup_enabled}"
      tags: "${tags}"
    driftPolicy:
      autoHeal: "${environment == 'prod'}"
      notifyOnly: "${environment != 'prod'}"

  # API servers
  - kind: aws:ec2:instance
    name: api-${index}
    count: "${instance_count}"
    properties:
      instance_type: "${instance_type}"
      ami: ami-0abcdef1234567890
      tags:
        Name: "api-${index}"
        Role: api-server
        Environment: "${environment}"
    driftPolicy:
      autoHeal: "${environment == 'prod'}"
```

## Multi-Region Deployment

Deploy resources across multiple regions.

```yaml
project: global-app
environment: production
variables:
  regions:
    - us-east-1
    - us-west-2
    - eu-west-1
  
  tags:
    owner: platform-team
    project: global-app

providers:
  aws:
    region: us-east-1  # Primary region

resources:
  # Regional buckets
  - kind: aws:s3:bucket
    name: "${project}-${region}-${environment}"
    for_each: "${regions}"
    properties:
      versioning: true
      tags:
        Region: "${region}"
        Primary: "${region == 'us-east-1' ? 'true' : 'false'}"
    driftPolicy:
      autoHeal: true

  # Regional compute
  - kind: aws:ec2:instance
    name: "${region}-app-${index}"
    for_each: "${regions}"
    count: 2
    properties:
      instance_type: t3.medium
      ami: ami-0abcdef1234567890
      tags:
        Name: "${region}-app-${index}"
        Region: "${region}"
    driftPolicy:
      autoHeal: true
```

## Development vs Production

Different configurations for different environments.

```yaml
# dev.yaml
project: myapp
environment: dev
variables:
  region: us-east-1
  tags:
    owner: dev-team
    cost-center: development

providers:
  aws:
    region: "${region}"

resources:
  - kind: aws:s3:bucket
    name: "${project}-dev-bucket"
    properties:
      versioning: false  # Save costs in dev
      tags: "${tags}"
    driftPolicy:
      autoHeal: false
      notifyOnly: true

  - kind: aws:ec2:instance
    name: dev-server
    properties:
      instance_type: t3.micro  # Smallest instance
      ami: ami-0abcdef1234567890
      tags:
        Name: dev-server
        Environment: dev
    driftPolicy:
      autoHeal: false
```

```yaml
# prod.yaml
project: myapp
environment: prod
variables:
  region: us-east-1
  tags:
    owner: platform-team
    cost-center: production

providers:
  aws:
    region: "${region}"

resources:
  - kind: aws:s3:bucket
    name: "${project}-prod-bucket"
    properties:
      versioning: true  # Enable versioning in prod
      tags: "${tags}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false

  - kind: aws:ec2:instance
    name: prod-server-${index}
    count: 3  # Multiple instances for HA
    properties:
      instance_type: t3.large  # Larger instances
      ami: ami-0abcdef1234567890
      tags:
        Name: "prod-server-${index}"
        Environment: prod
    driftPolicy:
      autoHeal: true
      notifyOnly: false
```

## Complex Dependencies

Resources with complex dependency relationships.

```yaml
project: microservices
environment: production
variables:
  region: us-east-1

providers:
  aws:
    region: "${region}"

resources:
  # Shared storage
  - kind: aws:s3:bucket
    name: shared-storage
    properties:
      versioning: true

  # Database storage
  - kind: aws:s3:bucket
    name: database-backups
    properties:
      versioning: true

  # API Gateway
  - kind: aws:ec2:instance
    name: api-gateway
    properties:
      instance_type: t3.medium
      ami: ami-0abcdef1234567890
      tags:
        Name: api-gateway
        Role: gateway
    depends_on:
      - aws:s3:bucket.shared-storage

  # Microservices
  - kind: aws:ec2:instance
    name: service-${index}
    count: 3
    properties:
      instance_type: t3.small
      ami: ami-0abcdef1234567890
      tags:
        Name: "service-${index}"
        Role: microservice
    depends_on:
      - aws:ec2:instance.api-gateway
      - aws:s3:bucket.shared-storage
      - aws:s3:bucket.database-backups
```

## Usage Examples

### Deploy to Development
```bash
# Use dev configuration
runestone bootstrap --config dev.yaml
runestone preview --config dev.yaml
runestone commit --config dev.yaml
```

### Deploy to Production
```bash
# Use prod configuration with extra safety
runestone bootstrap --config prod.yaml
runestone preview --config prod.yaml --json > changes.json
# Review changes.json
runestone commit --config prod.yaml --graph
```

### Environment-Specific Variables
```bash
# Set environment via environment variable
export ENV=staging
runestone bootstrap --config multi-env.yaml

# Or use different config files
runestone bootstrap --config staging.yaml
```

### Continuous Monitoring
```bash
# Monitor production for drift every 5 minutes
runestone align --config prod.yaml --interval 5m

# One-time drift check
runestone align --config prod.yaml --once
```
