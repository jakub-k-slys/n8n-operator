# n8n Operator Documentation

Welcome to the n8n Operator documentation. This operator automates the deployment and management of n8n workflow automation instances on Kubernetes clusters.

## Table of Contents

1. [Installation Guide](installation.md)
   * Prerequisites
   * Installation Steps
   * Project Distribution
   * Uninstallation

2. [Configuration Guide](configuration.md)
   - Traffic Routing Options
   - Database Configuration
   - Persistent Storage
   - Security Configuration
   - Complete Configuration Example

3. [API Reference](api.rst)
   - Custom Resource Definitions
   - API Specifications

## Features

### Traffic Routing
- Support for Kubernetes Ingress
- Support for Gateway API HTTPRoute
- Configurable hostnames and TLS

### Database Integration
- PostgreSQL database support
- Configurable connection parameters
- SSL support for secure connections

### Storage Management
- Persistent storage configuration
- Automatic volume provisioning
- Configurable storage size

### Security
- Non-root container execution
- Automated TLS configuration
- Secure database connections

## Support

If you encounter any issues or need assistance:
1. Check the documentation sections above
2. Look for similar issues in the GitHub repository
3. Open a new issue if needed

## Contributing

We welcome contributions! Please see our [Contributing Guide](../CONTRIBUTING.md) for details on how to:
- Submit bug reports
- Request features
- Submit pull requests
- Improve documentation
