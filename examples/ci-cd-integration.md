# CI/CD Integration Examples

Runestone supports JSON and Markdown output formats that are perfect for CI/CD pipelines and automation.

## GitHub Actions Example

```yaml
name: Infrastructure Deployment
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  infrastructure:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
          
      - name: Build Runestone
        run: go build -o runestone .
        
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1
          
      - name: Bootstrap Infrastructure
        run: |
          ./runestone bootstrap --config infra.yaml --output json > bootstrap-result.json
          
      - name: Preview Changes
        id: preview
        run: |
          ./runestone preview --config infra.yaml --output json > preview-result.json
          echo "preview_output=$(cat preview-result.json)" >> $GITHUB_OUTPUT
          
      - name: Comment PR with Preview
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const preview = JSON.parse(fs.readFileSync('preview-result.json', 'utf8'));
            
            let comment = '##  Infrastructure Preview\n\n';
            
            if (preview.success) {
              comment += `**Changes detected:** ${preview.changes_count}\n`;
              comment += `**Duration:** ${preview.duration_seconds.toFixed(2)}s\n\n`;
              
              if (preview.changes_count > 0) {
                comment += '### Planned Changes\n';
                preview.changes.forEach(change => {
                  const icon = change.type === 'create' ? '' : 
                              change.type === 'update' ? '' : '❌';
                  comment += `- ${icon} **${change.type}** \`${change.resource_kind}.${change.resource_name}\`\n`;
                });
              } else {
                comment += ' No changes detected - infrastructure is up to date\n';
              }
              
              if (preview.has_drift) {
                comment += '\n### Drift Detected\n';
                preview.drift_results.forEach(drift => {
                  if (drift.has_drift) {
                    comment += `-  **${drift.resource_name}**\n`;
                    drift.changes.forEach(change => {
                      comment += `  - ${change}\n`;
                    });
                  }
                });
              }
            } else {
              comment += ` **Error:** ${preview.error}\n`;
            }
            
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
            
      - name: Apply Changes
        if: github.ref == 'refs/heads/main'
        run: |
          ./runestone commit --config infra.yaml --output json --auto-approve > commit-result.json
          
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: runestone-results
          path: |
            bootstrap-result.json
            preview-result.json
            commit-result.json
```

## GitLab CI Example

```yaml
stages:
  - validate
  - preview
  - deploy

variables:
  AWS_DEFAULT_REGION: us-east-1

before_script:
  - go build -o runestone .

validate:
  stage: validate
  script:
    - ./runestone bootstrap --config infra.yaml --output json
  artifacts:
    reports:
      junit: bootstrap-result.json
    paths:
      - bootstrap-result.json

preview:
  stage: preview
  script:
    - ./runestone preview --config infra.yaml --output markdown > preview.md
    - ./runestone preview --config infra.yaml --output json > preview.json
  artifacts:
    paths:
      - preview.md
      - preview.json
  only:
    - merge_requests

deploy:
  stage: deploy
  script:
    - ./runestone commit --config infra.yaml --output json --auto-approve
  artifacts:
    paths:
      - commit-result.json
  only:
    - main
```

## Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    environment {
        AWS_DEFAULT_REGION = 'us-east-1'
    }
    
    stages {
        stage('Build') {
            steps {
                sh 'go build -o runestone .'
            }
        }
        
        stage('Bootstrap') {
            steps {
                sh './runestone bootstrap --config infra.yaml --output json > bootstrap-result.json'
                archiveArtifacts artifacts: 'bootstrap-result.json'
            }
        }
        
        stage('Preview') {
            steps {
                script {
                    sh './runestone preview --config infra.yaml --output json > preview-result.json'
                    
                    def preview = readJSON file: 'preview-result.json'
                    
                    if (preview.success) {
                        echo "Changes detected: ${preview.changes_count}"
                        echo "Duration: ${preview.duration_seconds}s"
                        
                        if (preview.changes_count > 0) {
                            echo "Planned changes:"
                            preview.changes.each { change ->
                                echo "  - ${change.type}: ${change.resource_kind}.${change.resource_name}"
                            }
                        }
                    } else {
                        error "Preview failed: ${preview.error}"
                    }
                }
                archiveArtifacts artifacts: 'preview-result.json'
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                sh './runestone commit --config infra.yaml --output json --auto-approve > commit-result.json'
                archiveArtifacts artifacts: 'commit-result.json'
            }
        }
    }
    
    post {
        always {
            publishHTML([
                allowMissing: false,
                alwaysLinkToLastBuild: true,
                keepAll: true,
                reportDir: '.',
                reportFiles: '*.json',
                reportName: 'Runestone Results'
            ])
        }
    }
}
```

## Parsing JSON Output

### Python Example

```python
import json
import sys

def parse_runestone_output(filename):
    with open(filename, 'r') as f:
        result = json.load(f)
    
    if result['success']:
        print(f" Operation successful")
        print(f"Duration: {result['duration_seconds']:.2f}s")
        
        if 'changes_count' in result:
            print(f"Changes: {result['changes_count']}")
            
        if 'policy_violations' in result and result['policy_violations']:
            print(f"Policy violations: {len(result['policy_violations'])}")
            for violation in result['policy_violations']:
                print(f"  - {violation['severity']}: {violation['message']}")
    else:
        print(f" Operation failed: {result['error']}")
        sys.exit(1)

if __name__ == "__main__":
    parse_runestone_output(sys.argv[1])
```

### Bash Example

```bash
#!/bin/bash

# Run bootstrap and capture result
./runestone bootstrap --config infra.yaml --output json > bootstrap.json

# Check if bootstrap was successful
SUCCESS=$(jq -r '.success' bootstrap.json)
if [ "$SUCCESS" != "true" ]; then
    echo "Bootstrap failed:"
    jq -r '.error' bootstrap.json
    exit 1
fi

# Check for policy violations
VIOLATIONS=$(jq -r '.policy_violations | length' bootstrap.json)
if [ "$VIOLATIONS" -gt 0 ]; then
    echo "Found $VIOLATIONS policy violations:"
    jq -r '.policy_violations[] | "  - \(.severity): \(.message)"' bootstrap.json
fi

# Run preview
./runestone preview --config infra.yaml --output json > preview.json

# Check changes
CHANGES=$(jq -r '.changes_count' preview.json)
echo "Changes detected: $CHANGES"

if [ "$CHANGES" -gt 0 ]; then
    echo "Planned changes:"
    jq -r '.changes[] | "  - \(.type): \(.resource_kind).\(.resource_name)"' preview.json
fi
```

## Output Format Reference

### Bootstrap JSON Output

```json
{
  "success": true,
  "providers_installed": ["aws"],
  "resource_count": 5,
  "modules_loaded": 0,
  "policy_violations": [
    {
      "resource_name": "aws:s3:bucket.logs",
      "rule_name": "s3-versioning-enabled",
      "message": "S3 bucket should have versioning enabled",
      "severity": "warning"
    }
  ],
  "duration_seconds": 1.234,
  "has_errors": false
}
```

### Preview JSON Output

```json
{
  "success": true,
  "changes_count": 2,
  "changes": [
    {
      "type": "create",
      "resource_kind": "aws:s3:bucket",
      "resource_name": "logs",
      "description": "Create S3 bucket logs"
    }
  ],
  "drift_results": [
    {
      "resource_name": "aws:s3:bucket.existing",
      "has_drift": true,
      "changes": ["versioning changed from false to true"]
    }
  ],
  "duration_seconds": 2.567,
  "has_drift": true
}
```

### Commit JSON Output

```json
{
  "success": true,
  "resources_applied": 3,
  "execution_levels": [
    {
      "level": 1,
      "resources": ["aws:s3:bucket.logs"],
      "duration_seconds": 15.0
    },
    {
      "level": 2,
      "resources": ["aws:ec2:instance.web-1", "aws:ec2:instance.web-2"],
      "duration_seconds": 45.0
    }
  ],
  "total_duration_seconds": 60.0
}
```

## Benefits for CI/CD

1. **Structured Output**: JSON format enables easy parsing and integration with other tools
2. **Error Handling**: Clear success/failure indicators with detailed error messages
3. **Metrics**: Duration and resource counts for monitoring and reporting
4. **Policy Integration**: Built-in policy violation reporting for governance
5. **Drift Detection**: Automated drift detection and reporting
6. **Parallel Execution**: Execution level information for understanding deployment flow

## Best Practices

1. **Always check the `success` field** before proceeding with subsequent steps
2. **Parse policy violations** to enforce governance in your pipeline
3. **Use preview in PR workflows** to show planned changes before deployment
4. **Archive JSON outputs** as build artifacts for debugging and auditing
5. **Set up notifications** based on drift detection results
6. **Use auto-approve flag** in production deployments with proper safeguards
