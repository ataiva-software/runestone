package docs

import "time"

const apiReferenceTemplate = `# API Reference

**Generated on: {{.GeneratedAt}}**

This document provides a comprehensive reference for all Runestone CLI commands and configuration options.

## CLI Commands

### ` + "`runestone bootstrap`" + `

Initializes your Runestone environment by installing providers and validating configuration.

` + "```bash" + `
runestone bootstrap [flags]
` + "```" + `

**Flags:**
- ` + "`-c, --config string`" + ` - Path to configuration file (default: "infra.yaml")
- ` + "`-h, --help`" + ` - Help for bootstrap

**Example:**
` + "```bash" + `
runestone bootstrap --config my-infra.yaml
` + "```" + `

### ` + "`runestone preview`" + `

Shows what changes would be made without applying them.

` + "```bash" + `
runestone preview [flags]
` + "```" + `

**Flags:**
- ` + "`-c, --config string`" + ` - Path to configuration file (default: "infra.yaml")
- ` + "`--json`" + ` - Output results in JSON format
- ` + "`-h, --help`" + ` - Help for preview

**Example:**
` + "```bash" + `
runestone preview --json > changes.json
` + "```" + `

### ` + "`runestone commit`" + `

Applies infrastructure changes.

` + "```bash" + `
runestone commit [flags]
` + "```" + `

**Flags:**
- ` + "`-c, --config string`" + ` - Path to configuration file (default: "infra.yaml")
- ` + "`--auto-approve`" + ` - Skip interactive approval
- ` + "`--graph`" + ` - Show DAG visualization during execution
- ` + "`-h, --help`" + ` - Help for commit

**Example:**
` + "```bash" + `
runestone commit --auto-approve --graph
` + "```" + `

### ` + "`runestone align`" + `

Monitors and fixes infrastructure drift.

` + "```bash" + `
runestone align [flags]
` + "```" + `

**Flags:**
- ` + "`-c, --config string`" + ` - Path to configuration file (default: "infra.yaml")
- ` + "`--once`" + ` - Run alignment once instead of continuously
- ` + "`--interval duration`" + ` - Interval between checks (default: 5m0s)
- ` + "`-h, --help`" + ` - Help for align

**Example:**
` + "```bash" + `
# One-time check
runestone align --once

# Continuous monitoring every 10 minutes
runestone align --interval 10m
` + "```" + `

### ` + "`runestone dismantle`" + `

Destroys infrastructure resources.

` + "```bash" + `
runestone dismantle [flags]
` + "```" + `

**Flags:**
- ` + "`-c, --config string`" + ` - Path to configuration file (default: "infra.yaml")
- ` + "`--auto-approve`" + ` - Skip interactive approval
- ` + "`--force`" + ` - Force deletion even with dependencies
- ` + "`-h, --help`" + ` - Help for dismantle

**Example:**
` + "```bash" + `
runestone dismantle --auto-approve
` + "```" + `

## Exit Codes

- ` + "`0`" + ` - Success
- ` + "`1`" + ` - General error
- ` + "`2`" + ` - Configuration error
- ` + "`3`" + ` - Provider error
- ` + "`4`" + ` - Resource error

## Environment Variables

- ` + "`AWS_PROFILE`" + ` - AWS profile to use (overrides config)
- ` + "`AWS_REGION`" + ` - AWS region to use (overrides config)
- ` + "`RUNESTONE_LOG_LEVEL`" + ` - Log level (debug, info, warn, error)

## JSON Output Format

When using ` + "`--json`" + ` flag with preview command:

` + "```json" + `
{
  "summary": {
    "create": 2,
    "update": 0,
    "delete": 0,
    "total": 2
  },
  "changes": [
    {
      "type": "create",
      "resource_id": "aws:s3:bucket.my-app-logs",
      "resource_kind": "aws:s3:bucket",
      "resource_name": "my-app-logs",
      "properties": {
        "versioning": true,
        "tags": {
          "owner": "platform-team"
        }
      }
    }
  ],
  "drift": {}
}
` + "```" + `
`

func (g *Generator) generateAPIReference() error {
	data := struct {
		GeneratedAt string
	}{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	content, err := g.executeTemplate(apiReferenceTemplate, data)
	if err != nil {
		return err
	}

	return g.writeFile("api-reference.md", content)
}
