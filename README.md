# Drift

[![Go Report Card](https://goreportcard.com/badge/github.com/ataiva-software/drift)](https://goreportcard.com/report/github.com/ataiva-software/drift)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test](https://github.com/ataiva-software/drift/actions/workflows/test.yml/badge.svg)](https://github.com/ataiva-software/drift/actions/workflows/test.yml)

**Declarative, drift-aware infrastructure — stateless, multi-cloud ready, and human-centric.**

Drift is Ataiva's next-generation Infrastructure-as-Code platform. It solves the common pain points of existing IaC tools — brittle state files, drift surprises, and complex multi-cloud orchestration — by offering a stateless, DAG-driven execution engine with real-time reconciliation and human-friendly CLI workflows.

## Key Features

- **Stateless Execution**: No centralized state files - infrastructure state is inferred from cloud provider APIs  **WORKING**
- **Real-time Drift Detection**: Continuous reconciliation with optional auto-healing  **WORKING**
- **DAG-based Orchestration**: Parallel execution with intelligent dependency resolution  **WORKING**
- **Human-centric CLI**: Clean, readable output with comprehensive change summaries  **WORKING**
- **Expression Language**: Support for loops, conditionals, and variables in YAML  **WORKING**
- **Policy-as-Code**: Built-in security and governance policy enforcement  **WORKING**
- **Module System**: Reusable infrastructure components with local module support  **WORKING**
- **Multi-cloud Ready**: Extensible provider system (AWS production-ready, Kubernetes planned)

## Production Ready

Drift is **production-ready** for AWS infrastructure management with:

###  **Core Infrastructure Engine**

- Complete stateless execution with AWS API integration
- Real-time drift detection and auto-healing
- DAG-based parallel execution with dependency resolution
- Comprehensive error handling and retry logic

### **AWS Provider (Production Ready)**

- **S3 Buckets**: Full lifecycle management with versioning and tagging
- **EC2 Instances**: Complete instance management with state tracking
- **VPC Networking**: VPC, subnet, internet gateway, and security group management
- **Lambda Functions**: Serverless function deployment and management
- **Database Services**: RDS instances and DynamoDB tables with full configuration
- **API Gateway**: REST API management for serverless architectures
- **IAM Resources**: Complete IAM user, role, and policy management with tagging
- **Resource Validation**: Comprehensive validation for all resource types

### **Policy-as-Code (Production Ready)**

- Built-in security policies (S3 versioning, resource tagging)
- Governance policies (environment tagging, cost optimization)
- Severity-based enforcement (errors, warnings, info)
- Real-time policy evaluation during bootstrap

### **Enterprise Features**

- Module system for reusable infrastructure components
- Expression language with conditionals and loops
- Comprehensive CLI with human-readable output
- Full test coverage with integration testing

### **Getting Started**

Ready to use Drift in production? Check out our comprehensive documentation:

- **[ Complete Documentation](docs/)** - Auto-generated, always up-to-date
- **[ Getting Started Guide](docs/getting-started.md)** - Step-by-step tutorial
- **[ API Reference](docs/api-reference.md)** - Complete CLI command reference
- **[ Configuration Reference](docs/configuration-reference.md)** - YAML configuration guide
- **[ Examples](docs/examples.md)** - Real-world use cases and patterns

Or jump right in with the [Quick Start](#-quick-start) below.

## Installation

### Download Binary (Recommended)

Download the latest release for your platform:

- **Linux (x64)**: [drift-linux-amd64](https://github.com/ataiva-software/drift/releases/latest/download/drift-linux-amd64)
- **Linux (ARM64)**: [drift-linux-arm64](https://github.com/ataiva-software/drift/releases/latest/download/drift-linux-arm64)
- **macOS (Intel)**: [drift-darwin-amd64](https://github.com/ataiva-software/drift/releases/latest/download/drift-darwin-amd64)
- **macOS (Apple Silicon)**: [drift-darwin-arm64](https://github.com/ataiva-software/drift/releases/latest/download/drift-darwin-arm64)
- **Windows (x64)**: [drift-windows-amd64.exe](https://github.com/ataiva-software/drift/releases/latest/download/drift-windows-amd64.exe)

After downloading, make the binary executable and move it to your PATH:

```bash
# Linux/macOS
chmod +x drift-*
sudo mv drift-* /usr/local/bin/drift

# Verify installation
drift --version
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ataiva-software/drift.git
cd drift

# Build the binary
go build -o drift .

# Install globally (optional)
go install .
```

## Quick Start

### 1. Create a configuration file (`infra.yaml`)

```yaml
project: my-app
environment: dev
variables:
  region: us-east-1
  tags:
    owner: platform-team
    purpose: demo
    Environment: dev  # Required by policy

providers:
  aws:
    region: "${region}"
    profile: default

resources:
  # S3 bucket with versioning (policy compliant)
  - kind: aws:s3:bucket
    name: my-app-logs
    properties:
      versioning: true  # Required by policy
      tags:
        owner: platform-team
        purpose: demo
        Environment: dev  # Required by policy
    driftPolicy:
      autoHeal: true
      notifyOnly: false

  # RDS database
  - kind: aws:rds:instance
    name: my-app-db
    properties:
      db_instance_class: db.t3.micro
      engine: mysql
      engine_version: "8.0"
      db_name: myapp
      master_username: admin
      master_user_password: "ChangeMe123!"
      allocated_storage: 20
      backup_retention_period: 1
      tags:
        owner: platform-team
        purpose: demo
        Environment: dev  # Required by policy
    driftPolicy:
      autoHeal: true
      notifyOnly: false

  # Multiple EC2 instances using count
  - kind: aws:ec2:instance
    name: web-${index}
    count: 2
    properties:
      instance_type: t3.micro
      ami: ami-0abcdef1234567890
      tags:
        Name: "web-${index}"
        Environment: "${environment}"
        owner: "${tags.owner}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false
    depends_on:
      - "aws:rds:instance.my-app-db"

  # IAM user for application access
  - kind: aws:iam:user
    name: app-service-user
    properties:
      path: "/applications/"
      tags:
        owner: platform-team
        purpose: application-access
        Environment: dev  # Required by policy
    driftPolicy:
      autoHeal: true
      notifyOnly: false
```

### 2. Bootstrap your environment

```bash
drift bootstrap
```

Output:

```
 Bootstrapping Drift environment...
 Installing provider aws...
 Validating configuration...
 Configuration validated successfully
 Found 5 resource instances
 Evaluating policies...
 No policy violations found
 Bootstrap complete!
```

### 3. Preview changes

```bash
drift preview
```

Output:

```
 Inspecting live infrastructure...

Changes detected:

+ 5 new resources will be created

Detailed changes:
+ Create aws:s3:bucket.my-app-logs (aws:s3:bucket)
+ Create aws:rds:instance.my-app-db (aws:rds:instance)
+ Create aws:ec2:instance.web-0 (aws:ec2:instance)
+ Create aws:ec2:instance.web-1 (aws:ec2:instance)
+ Create aws:iam:user.app-service-user (aws:iam:user)

Next: run 'drift commit' to apply these changes.
```

### 4. Apply changes

```bash
drift commit
```

Output:

```
 Committing infrastructure changes...

--- Execution Level 1 ---
+ Creating aws:s3:bucket.my-app-logs
✓ Completed aws:s3:bucket.my-app-logs
+ Creating aws:rds:instance.my-app-db
✓ Completed aws:rds:instance.my-app-db
+ Creating aws:iam:user.app-service-user
✓ Completed aws:iam:user.app-service-user

--- Execution Level 2 ---
+ Creating aws:ec2:instance.web-0
+ Creating aws:ec2:instance.web-1
✓ Completed aws:ec2:instance.web-0
✓ Completed aws:ec2:instance.web-1

--- Execution Complete ---
 Commit complete (duration: 3m45s)

Changes applied:
+ Created aws:s3:bucket.my-app-logs
+ Created aws:rds:instance.my-app-db
+ Created aws:iam:user.app-service-user
+ Created aws:ec2:instance.web-0
+ Created aws:ec2:instance.web-1
```

### 5. Monitor and align drift

```bash
drift align --once
```

Output:

```
 Aligning desired state with reality... (14:30:15)
 Infrastructure aligned (no drift detected)
```

### 6. Policy Enforcement in Action

Drift includes built-in policies that automatically evaluate your infrastructure:

```bash
# Example with policy violations
drift bootstrap --config examples/policy-demo.yaml
```

Output:

```
 Bootstrapping Drift environment...
 Installing provider aws...
 Validating configuration...
 Configuration validated successfully
 Found 4 resource instances
 Evaluating policies...
  Found 3 policy violations:
    3 warnings
    - aws:s3:bucket.bad-bucket: S3 bucket should have versioning enabled for data protection
    - aws:s3:bucket.bad-bucket: Resource must have an Environment tag for proper resource management
    - aws:rds:instance.demo-db: Resource must have an Environment tag for proper resource management
 Bootstrap complete!
```

**Built-in Policies Include:**

- S3 versioning enforcement
- Resource tagging requirements
- Cost optimization rules
- Security best practices

## CLI Commands

| Command | Description |
|---------|-------------|
| `bootstrap` | Install providers, pull modules, and validate configuration |
| `preview` | Preview changes and detect drift (dry-run) |
| `commit` | Apply infrastructure changes |
| `align` | Continuously reconcile drift |
| `dismantle` | Destroy infrastructure resources |

### Command Options

```bash
# Bootstrap
drift bootstrap --config infra.yaml

# Preview with JSON output
drift preview --output json

# Commit with DAG visualization
drift commit --graph --auto-approve

# Continuous alignment
drift align --interval 10m

# One-time alignment
drift align --once

# Destroy infrastructure
drift dismantle --auto-approve
```

### Output Formats

All commands support multiple output formats for CI/CD integration:

```bash
# Human-readable output (default)
drift bootstrap --output human

# JSON output for automation
drift bootstrap --output json

# Markdown output for documentation
drift bootstrap --output markdown
```

**CI/CD Integration**: See [examples/ci-cd-integration.md](examples/ci-cd-integration.md) for complete GitHub Actions, GitLab CI, and Jenkins pipeline examples.

## Configuration Reference

### Basic Structure

```yaml
project: string              # Project name
environment: string          # Environment (dev, staging, prod)
variables:                   # Global variables
  key: value
providers:                   # Cloud providers
  aws:
    region: string
    profile: string
modules:                     # Reusable modules (future)
  name:
    source: string
    version: string
    inputs: {}
resources:                   # Infrastructure resources
  - kind: string
    name: string
    properties: {}
    driftPolicy: {}
```

### Expression Language

Runestone supports expressions in YAML values using `${}` syntax:

```yaml
# Variable substitution
region: "${region}"

# Ternary conditions
instance_type: "${environment == 'prod' ? 't3.large' : 't3.micro'}"

# Complex expressions
storage_size: "${environment == 'prod' ? 100 : 20}"
backup_enabled: "${environment == 'prod'}"
```

### Loops and Dynamic Resources

#### Count-based Resources

```yaml
resources:
  - kind: aws:ec2:instance
    name: web-${index}
    count: 3
    properties:
      instance_type: t3.micro
      tags:
        Name: "web-${index}"
```

#### For-each Resources

```yaml
variables:
  regions: [us-east-1, us-west-2, eu-west-1]

resources:
  - kind: aws:s3:bucket
    name: "logs-${region}"
    for_each: "${regions}"
    properties:
      versioning: true
      tags:
        region: "${region}"
```

### Drift Policies

```yaml
resources:
  - kind: aws:s3:bucket
    name: my-bucket
    properties:
      versioning: true
    driftPolicy:
      autoHeal: true      # Automatically fix drift
      notifyOnly: false   # Don't just notify, take action
```

## Supported Resources

### AWS Provider (13 Resource Types)

| Resource Type | Kind | Properties |
|---------------|------|------------|
| **Storage** |
| S3 Bucket | `aws:s3:bucket` | `versioning`, `tags` |
| **Compute** |
| EC2 Instance | `aws:ec2:instance` | `instance_type`, `ami`, `tags` |
| Lambda Function | `aws:lambda:function` | `runtime`, `handler`, `role`, `code_content`, `timeout`, `memory_size`, `tags` |
| **Networking** |
| VPC | `aws:ec2:vpc` | `cidr_block`, `tags` |
| Subnet | `aws:ec2:subnet` | `vpc_id`, `cidr_block`, `availability_zone`, `tags` |
| Internet Gateway | `aws:ec2:internet_gateway` | `tags` |
| Security Group | `aws:ec2:security_group` | `description`, `vpc_id`, `ingress`, `egress`, `tags` |
| **Database** |
| RDS Instance | `aws:rds:instance` | `db_instance_class`, `engine`, `engine_version`, `db_name`, `master_username`, `master_user_password`, `allocated_storage`, `backup_retention_period`, `tags` |
| DynamoDB Table | `aws:dynamodb:table` | `hash_key`, `range_key`, `attributes`, `tags` |
| **API & Integration** |
| API Gateway | `aws:apigateway:rest_api` | `description`, `tags` |
| **Security & Identity** |
| IAM User | `aws:iam:user` | `path`, `tags` |
| IAM Role | `aws:iam:role` | `assume_role_policy`, `path`, `description`, `tags` |
| IAM Policy | `aws:iam:policy` | `policy`, `path`, `description`, `tags` |

**Ready for Production**: All resources include full CRUD operations, drift detection, policy compliance, and comprehensive validation.

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/config
go test ./internal/providers/aws
go test ./internal/drift
go test ./internal/executor
```

## Development

### Project Structure

```
drift/
├── cmd/                    # CLI commands
│   ├── bootstrap.go
│   ├── preview.go
│   ├── commit.go
│   ├── align.go
│   ├── dismantle.go
│   └── docs.go            # Documentation generation
├── internal/
│   ├── config/            # Configuration parsing
│   ├── providers/         # Cloud provider implementations
│   │   └── aws/
│   ├── executor/          # DAG execution engine
│   ├── drift/             # Drift detection
│   └── docs/              # Documentation generation
├── examples/              # Example configurations
├── docs/                  # Generated documentation
├── .github/workflows/     # GitHub Actions
├── Makefile              # Build automation
└── README.md
```

### Build System

Drift uses a Makefile for build automation with automatic documentation generation:

```bash
# Development build with docs
make dev

# Full release build
make release VERSION=v1.0.0

# Run tests
make test

# Generate documentation only
make docs

# Clean build artifacts
make clean

# Install globally
make install
```

### GitHub Actions

The project includes two GitHub Actions workflows:

#### Test Workflow (`.github/workflows/test.yml`)

- **Triggers**: Every push to `main` and pull requests
- **Actions**: Run tests, build binary, validate examples
- **Purpose**: Continuous integration and quality assurance

#### Release Workflow (`.github/workflows/release.yml`)

- **Triggers**:
  - Manual dispatch with version input
  - Push of version tags (e.g., `v1.0.0`)
- **Actions**:
  - Run full test suite
  - Build binaries for multiple platforms (Linux, macOS, Windows)
  - Generate documentation
  - Create GitHub release with binaries and release notes
- **Purpose**: Automated releases with cross-platform binaries

#### Creating a Release

**Option 1: Manual Trigger**

1. Go to GitHub Actions → Release workflow
2. Click "Run workflow"
3. Enter version (e.g., `v1.0.0`)
4. Workflow creates tag and release automatically

**Option 2: Tag Push**

```bash
git tag v1.0.0
git push origin v1.0.0
```

Both methods create a GitHub release with:

- Cross-platform binaries
- Auto-generated release notes
- Complete documentation

### Documentation Generation

Documentation is automatically generated during build using the `drift docs` command:

- **Getting Started Guide** - Step-by-step tutorial
- **API Reference** - Complete CLI command reference
- **Configuration Reference** - YAML configuration guide
- **Examples** - Practical use cases and patterns

```bash
# Generate docs manually
./drift docs --output docs

# Generated files:
# docs/getting-started.md
# docs/api-reference.md
# docs/configuration-reference.md
# docs/examples.md
# docs/README.md (overview)
```

### Adding a New Provider

1. Implement the `Provider` interface in `internal/providers/`
2. Register the provider in CLI commands
3. Add resource validation and operations
4. Write comprehensive tests

### Adding a New Resource Type

1. Add resource kind to provider's `GetSupportedResourceTypes()`
2. Implement CRUD operations in provider
3. Add validation logic
4. Write tests for the new resource type

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📧 Email: <support@ataiva.com>
- 🐛 Issues: [GitHub Issues](https://github.com/ataiva-software/drift/issues)
-  Documentation: [docs/](docs/)

##  Roadmap

### Phase 1 (MVP) - Core Infrastructure Engine  **COMPLETED**

#### Configuration & Parsing 

- [x] YAML configuration parsing  **WORKING**
- [x] Expression language with variables, conditionals, loops  **WORKING**
- [x] Resource expansion (count, for_each)  **WORKING**
- [x] Configuration validation  **WORKING**

#### CLI Commands 

- [x] `bootstrap` - Install providers and validate configuration  **WORKING**
- [x] `preview` - Preview changes and detect drift  **WORKING** (requires valid AWS credentials)
- [x] `commit` - Apply infrastructure changes  **WORKING** (requires valid AWS credentials)
- [x] `align` - Continuously reconcile drift  **WORKING** (requires valid AWS credentials)
- [x] `dismantle` - Destroy infrastructure resources  **WORKING** (requires valid AWS credentials)
- [x] `docs` - Generate comprehensive documentation  **WORKING**

#### AWS Provider 

- [x] Provider initialization and authentication  **WORKING**
- [x] S3 bucket support (create, update, delete, state retrieval)  **WORKING**
- [x] EC2 instance support (create, state retrieval, delete)  **WORKING**
- [x] Resource tagging  **WORKING**
- [x] Error handling and retries  **WORKING**

#### Core Engine 

- [x] DAG-based execution engine  **WORKING**
- [x] Dependency resolution  **WORKING**
- [x] Parallel execution  **WORKING**
- [x] State management (stateless design)  **WORKING**
- [x] Drift detection algorithm  **WORKING**
- [x] Auto-healing capabilities  **WORKING**

#### Testing & Documentation 

- [x] Comprehensive test suite  **WORKING** (all tests passing)
- [x] Integration tests with AWS  **WORKING** (skips when no credentials)
- [x] CLI help documentation  **WORKING**
- [x] Example configurations  **WORKING**
- [x] Automatic documentation generation  **WORKING**
- [x] Getting Started guide  **WORKING**
- [x] API Reference  **WORKING**
- [x] Configuration Reference  **WORKING**
- [x] Examples documentation  **WORKING**

#### Build System 

- [x] Makefile with automated builds  **WORKING**
- [x] Automatic documentation generation on build  **WORKING**
- [x] Test automation  **WORKING**
- [x] Development and release builds  **WORKING**

### Phase 2 (Team Scale)  **COMPLETED**

#### Enhanced AWS Resources 

- [x] RDS instance support (create, update, delete, state retrieval)  **WORKING**
- [x] IAM User support (create, update, delete, state retrieval)  **WORKING**
- [x] IAM Role support (create, update, delete, state retrieval)  **WORKING**
- [x] IAM Policy support (create, update, delete, state retrieval)  **WORKING**
- [x] VPC support (create, update, delete, state retrieval)  **WORKING**
- [x] Subnet support (create, update, delete, state retrieval)  **WORKING**
- [x] Internet Gateway support (create, update, delete, state retrieval)  **WORKING**
- [x] Enhanced resource validation and error handling  **WORKING**
- [x] Comprehensive test coverage for all AWS resources  **WORKING**

#### Policy-as-Code Integration 

- [x] Policy engine with rule evaluation  **WORKING**
- [x] Built-in security and governance policies  **WORKING**
- [x] Policy violation detection and reporting  **WORKING**
- [x] Integration with bootstrap command  **WORKING**
- [x] Severity-based policy enforcement  **WORKING**

#### Module System Foundation 

- [x] Module registry and management  **WORKING**
- [x] Local module loading support  **WORKING**
- [x] Module validation and expansion framework  **WORKING**
- [x] Comprehensive test coverage  **WORKING**

#### Additional Phase 2 Features (Future)

- [ ] Kubernetes provider
- [x] Enhanced AWS resources (Lambda, CloudFormation)  **Lambda WORKING**
- [x] JSON/Markdown output for CI/CD  **WORKING**
- [x] CI/CD integration examples  **WORKING**

### Phase 3 (Next-Gen InfraOps)

- [ ] Multi-cloud orchestration (GCP, Azure)
- [ ] Self-healing infrastructure
- [ ] Observability and metrics
- [ ] GraphQL/REST API
- [ ] Cost analysis and optimization
- [ ] Performance optimization

### Phase 4 (Vision)

- [ ] Universal infrastructure control plane
- [ ] Plugin ecosystem
- [ ] GUI dashboard
- [ ] Advanced policy engine
- [ ] AI-powered infrastructure optimization

---

**Built with  by [Ataiva Software](https://ataiva.com)**
