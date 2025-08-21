# Runestone Documentation

Welcome to the comprehensive documentation for Runestone, the next-generation Infrastructure-as-Code platform.

## 📚 Documentation Overview

This documentation is automatically generated from the latest version of Runestone and provides complete coverage of all features, commands, and configuration options.

### 🚀 [Getting Started](getting-started.md)
**New to Runestone?** Start here for a step-by-step guide to:
- Installing Runestone
- Creating your first infrastructure configuration
- Understanding key concepts like stateless design and policy-as-code
- Running your first deployment

### 📖 [API Reference](api-reference.md)
**Complete CLI command reference** covering:
- All available commands (bootstrap, preview, commit, align, dismantle, docs)
- Command flags and options
- Exit codes and error handling
- Environment variables

### ⚙️ [Configuration Reference](configuration-reference.md)
**Comprehensive YAML configuration guide** including:
- Complete configuration file structure
- All supported AWS resources (S3, EC2, RDS)
- Expression language syntax and examples
- Drift policies and dependency management
- Best practices and security considerations

### 💡 [Examples](examples.md)
**Real-world configuration examples** featuring:
- Simple web applications
- Multi-tier applications with databases
- Multi-environment configurations
- Policy compliance demonstrations
- Advanced features and patterns

## 🎯 Key Features Covered

### ✅ Production-Ready Infrastructure Management
- **Stateless Execution**: No state files, direct cloud provider API integration
- **Real-time Drift Detection**: Continuous monitoring with auto-healing capabilities
- **DAG-based Orchestration**: Intelligent dependency resolution and parallel execution
- **Policy-as-Code**: Built-in security and governance policy enforcement

### ✅ AWS Provider Support
- **S3 Buckets**: Complete lifecycle management with versioning and tagging
- **EC2 Instances**: Full instance management with state tracking
- **RDS Instances**: Database lifecycle with backup and configuration management
- **Resource Validation**: Comprehensive validation for all resource types

### ✅ Advanced Configuration Features
- **Expression Language**: Variables, conditionals, and loops in YAML
- **Module System**: Reusable infrastructure components (foundation implemented)
- **Multi-Environment**: Environment-specific configurations and scaling
- **Dependency Management**: Explicit resource dependencies with DAG execution

## 🔄 Documentation Updates

This documentation is automatically regenerated with each build to ensure it stays current with the latest features and changes. To regenerate the documentation:

```bash
# Generate documentation only
make docs

# Full development build with documentation
make dev

# Release build with tests and documentation
make release
```


## 🆘 Getting Help

### Documentation Issues
If you find issues with this documentation:
- 📧 Email: support@ataiva.com
- 🐛 Issues: [GitHub Issues](https://github.com/ataiva-software/runestone/issues)

### Quick Reference
- **Bootstrap**: runestone bootstrap - Install providers and validate configuration
- **Preview**: runestone preview - See what changes will be made
- **Commit**: runestone commit - Apply infrastructure changes
- **Align**: runestone align --once - Check and fix drift
- **Docs**: runestone docs - Generate this documentation

## 📋 Documentation Structure

```
docs/
├── README.md                    # This overview (you are here)
├── getting-started.md           # Step-by-step tutorial
├── api-reference.md             # Complete CLI reference
├── configuration-reference.md   # YAML configuration guide
└── examples.md                  # Real-world examples
```


## 🏗️ What's Next?

After reading through this documentation, you'll be ready to:

1. **Deploy Production Infrastructure**: Use Runestone for real AWS infrastructure management
2. **Implement Policy-as-Code**: Leverage built-in policies for security and governance
3. **Scale Multi-Environment**: Manage dev, staging, and production environments
4. **Contribute**: Help improve Runestone with feedback and contributions

---

**Built with ❤️ by [Ataiva Software](https://ataiva.com)**

*Last updated: Automatically generated with each build*