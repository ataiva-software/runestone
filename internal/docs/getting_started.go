package docs

import "time"

const gettingStartedTemplate = `# Getting Started with Runestone

**Generated on: {{.GeneratedAt}}**

Runestone is a declarative, drift-aware Infrastructure-as-Code platform that provides stateless, multi-cloud ready infrastructure management with human-centric CLI workflows.

## 🚀 Quick Start

### 1. Installation

#### From Source
` + "```bash" + `
git clone https://github.com/ataiva-software/runestone.git
cd runestone
go build -o runestone .
` + "```" + `

#### Install Globally
` + "```bash" + `
go install github.com/ataiva-software/runestone@latest
` + "```" + `

### 2. Create Your First Configuration

Create a file called ` + "`infra.yaml`" + `:

` + "```yaml" + `
project: my-first-project
environment: dev
variables:
  region: us-east-1
  tags:
    owner: platform-team
    purpose: demo

providers:
  aws:
    region: "${region}"
    profile: default

resources:
  # S3 bucket for application logs
  - kind: aws:s3:bucket
    name: my-app-logs-${environment}
    properties:
      versioning: true
      tags: "${tags}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false

  # Multiple web servers using count
  - kind: aws:ec2:instance
    name: web-${index}
    count: 2
    properties:
      instance_type: t3.micro
      ami: ami-0abcdef1234567890
      tags:
        Name: "web-${index}"
        Environment: "${environment}"
    driftPolicy:
      autoHeal: true
      notifyOnly: false
` + "```" + `

### 3. Bootstrap Your Environment

` + "```bash" + `
runestone bootstrap
` + "```" + `

This command:
- Validates your configuration
- Installs required providers
- Expands resources (2 EC2 instances + 1 S3 bucket = 3 total resources)

### 4. Preview Changes

` + "```bash" + `
runestone preview
` + "```" + `

This shows you what changes will be made without actually applying them.

### 5. Apply Changes

` + "```bash" + `
runestone commit
` + "```" + `

This applies the changes to your infrastructure.

### 6. Monitor for Drift

` + "```bash" + `
# One-time drift check
runestone align --once

# Continuous monitoring (runs every 5 minutes)
runestone align
` + "```" + `

### 7. Clean Up

` + "```bash" + `
runestone dismantle --auto-approve
` + "```" + `

## 🔧 Key Concepts

### Stateless Design
Runestone doesn't use state files. Instead, it queries your cloud providers directly to understand the current state of your infrastructure.

### Expression Language
Use ` + "`${variable}`" + ` syntax for dynamic values:
- ` + "`${region}`" + ` - Variable substitution
- ` + "`${environment == 'prod' ? 't3.large' : 't3.micro'}`" + ` - Conditional expressions
- ` + "`${index}`" + ` - Loop index for count-based resources

### Drift Detection
Runestone continuously monitors your infrastructure and can automatically fix drift when ` + "`autoHeal: true`" + ` is set.

### DAG Execution
Resources are executed in dependency order using a Directed Acyclic Graph (DAG) for optimal parallelization.

## 📖 Next Steps

- Read the [Configuration Reference](configuration-reference.md)
- Check out [Examples](examples.md)
- Explore the [API Reference](api-reference.md)

## 🆘 Getting Help

- 📧 Email: support@ataiva.com
- 🐛 Issues: [GitHub Issues](https://github.com/ataiva-software/runestone/issues)
- 📖 Documentation: Generated docs in ` + "`docs/`" + ` directory
`

func (g *Generator) generateGettingStarted() error {
	data := struct {
		GeneratedAt string
	}{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	content, err := g.executeTemplate(gettingStartedTemplate, data)
	if err != nil {
		return err
	}

	return g.writeFile("getting-started.md", content)
}
