# Configuration Reference

**Generated on: 2025-08-21 17:13:55 UTC**

This document provides a complete reference for Runestone configuration files.

## Configuration File Structure

```yaml
project: string              # Project name (required)
environment: string          # Environment name (required)
variables:                   # Global variables (optional)
  key: value
providers:                   # Cloud providers (required)
  provider_name:
    # Provider-specific configuration
modules:                     # Reusable modules (optional)
  module_name:
    source: string
    version: string
    inputs: {}
resources:                   # Infrastructure resources (required)
  - kind: string
    name: string
    # Resource-specific configuration
```

## Top-Level Fields

### `project` (required)
The name of your project. Used for resource naming and organization.

```yaml
project: my-awesome-app
```

### `environment` (required)
The environment name (e.g., dev, staging, prod). Available as `${environment}` variable.

```yaml
environment: production
```

### `variables` (optional)
Global variables that can be referenced throughout the configuration.

```yaml
variables:
  region: us-west-2
  instance_type: t3.medium
  tags:
    owner: platform-team
    cost-center: engineering
  allowed_cidrs:
    - "10.0.0.0/8"
    - "172.16.0.0/12"
```

## Providers

### AWS Provider

```yaml
providers:
  aws:
    region: string           # AWS region (required)
    profile: string          # AWS profile name (optional)
```

**Example:**
```yaml
providers:
  aws:
    region: "${region}"
    profile: production
```

## Resources

### Common Resource Fields

All resources support these fields:

```yaml
resources:
  - kind: string             # Resource type (required)
    name: string             # Resource name (required)
    count: int               # Number of instances (optional)
    for_each: array          # Iterate over array (optional)
    properties: {}           # Resource properties (optional)
    driftPolicy: {}          # Drift handling policy (optional)
    depends_on: []           # Dependencies (optional)
```

### AWS S3 Bucket

```yaml
- kind: aws:s3:bucket
  name: my-bucket-name
  properties:
    versioning: boolean      # Enable versioning (optional)
    tags: {}                 # Resource tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-fix drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:s3:bucket
  name: "${project}-logs-${environment}"
  properties:
    versioning: true
    tags:
      Environment: "${environment}"
      Purpose: application-logs
  driftPolicy:
    autoHeal: true
    notifyOnly: false
```

### AWS EC2 Instance

```yaml
- kind: aws:ec2:instance
  name: instance-name
  properties:
    instance_type: string    # EC2 instance type (required)
    ami: string              # AMI ID (required)
    tags: {}                 # Instance tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-fix drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:ec2:instance
  name: web-server-${index}
  count: 3
  properties:
    instance_type: "${environment == 'prod' ? 't3.large' : 't3.micro'}"
    ami: ami-0abcdef1234567890
    tags:
      Name: "web-server-${index}"
      Environment: "${environment}"
      Role: web-server
  driftPolicy:
    autoHeal: true
    notifyOnly: false
```

### AWS VPC

```yaml
- kind: aws:ec2:vpc
  name: vpc-name
  properties:
    cidr_block: string       # VPC CIDR block (required)
    tags: {}                 # VPC tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-heal drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:ec2:vpc
  name: app-vpc
  properties:
    cidr_block: "10.0.0.0/16"
    tags:
      Environment: "${environment}"
      Purpose: application-network
```

### AWS Subnet

```yaml
- kind: aws:ec2:subnet
  name: subnet-name
  properties:
    vpc_id: string           # VPC ID (required)
    cidr_block: string       # Subnet CIDR block (required)
    availability_zone: string # AZ (optional)
    tags: {}                 # Subnet tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-heal drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:ec2:subnet
  name: public-subnet-1a
  properties:
    vpc_id: "vpc-12345678"
    cidr_block: "10.0.1.0/24"
    availability_zone: "us-east-1a"
    tags:
      Environment: "${environment}"
      Tier: public
  depends_on:
    - "aws:ec2:vpc.app-vpc"
```

### AWS Internet Gateway

```yaml
- kind: aws:ec2:internet_gateway
  name: igw-name
  properties:
    tags: {}                 # IGW tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-heal drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:ec2:internet_gateway
  name: app-igw
  properties:
    tags:
      Environment: "${environment}"
      Purpose: internet-access
```

### AWS Lambda Function

```yaml
- kind: aws:lambda:function
  name: function-name
  properties:
    runtime: string          # Lambda runtime (required)
    handler: string          # Function handler (required)
    role: string             # IAM role ARN (required)
    code_content: string     # Function code (optional)
    description: string      # Function description (optional)
    timeout: integer         # Timeout in seconds (optional)
    memory_size: integer     # Memory in MB (optional)
    tags: {}                 # Function tags (optional)
  driftPolicy:
    autoHeal: boolean        # Auto-heal drift (default: false)
    notifyOnly: boolean      # Only notify on drift (default: true)
```

**Example:**
```yaml
- kind: aws:lambda:function
  name: data-processor
  properties:
    runtime: "python3.9"
    handler: "index.handler"
    role: "arn:aws:iam::123456789012:role/lambda-role"
    description: "Data processing function"
    timeout: 300
    memory_size: 512
    code_content: |
      def handler(event, context):
          return {'statusCode': 200, 'body': 'Hello World'}
    tags:
      Environment: "${environment}"
      Purpose: data-processing
  depends_on:
    - "aws:iam:role.lambda-role"
```

## Expression Language

Runestone supports expressions using `${}` syntax:

### Variable References
```yaml
region: "${region}"
name: "${project}-${environment}"
```

### Conditional Expressions
```yaml
instance_type: "${environment == 'prod' ? 't3.large' : 't3.micro'}"
backup_enabled: "${environment == 'prod'}"
storage_size: "${environment == 'prod' ? 100 : 20}"
```

### Loop Variables
When using `count` or `for_each`, special variables are available:

- `${index}` - Current index (0-based) for count
- `${item}` - Current item for for_each

```yaml
# Using count
- kind: aws:ec2:instance
  name: web-${index}
  count: 3

# Using for_each
- kind: aws:s3:bucket
  name: logs-${region}
  for_each: "${regions}"
```

## Drift Policies

Control how Runestone handles configuration drift:

```yaml
driftPolicy:
  autoHeal: boolean          # Automatically fix drift
  notifyOnly: boolean        # Only report drift, don't fix
```

**Behavior:**
- `autoHeal: true, notifyOnly: false` - Automatically fix drift
- `autoHeal: false, notifyOnly: true` - Report drift only
- `autoHeal: false, notifyOnly: false` - Report and prompt for action

## Dependencies

Specify resource dependencies using `depends_on`:

```yaml
resources:
  - kind: aws:s3:bucket
    name: app-bucket
    # ... properties

  - kind: aws:ec2:instance
    name: app-server
    depends_on:
      - aws:s3:bucket.app-bucket
    # ... properties
```

## Complete Example

```yaml
project: ecommerce-platform
environment: production
variables:
  region: us-east-1
  instance_count: 3
  tags:
    owner: platform-team
    cost-center: engineering
    project: ecommerce

providers:
  aws:
    region: "${region}"
    profile: production

resources:
  # Application logs bucket
  - kind: aws:s3:bucket
    name: "${project}-logs-${environment}"
    properties:
      versioning: true
      tags: "${tags}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false

  # Web servers
  - kind: aws:ec2:instance
    name: web-${index}
    count: "${instance_count}"
    properties:
      instance_type: "${environment == 'prod' ? 't3.large' : 't3.micro'}"
      ami: ami-0abcdef1234567890
      tags:
        Name: "web-${index}"
        Role: web-server
        Environment: "${environment}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false
    depends_on:
      - "aws:s3:bucket.${project}-logs-${environment}"
```
