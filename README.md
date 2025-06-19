<div align="center">
  <img src="https://raw.githubusercontent.com/n8n-io/n8n/master/assets/n8n-logo.png" alt="n8n Logo" height="100">
  <br><br>
  <img src="https://raw.githubusercontent.com/operator-framework/operator-sdk/4407fd6723aef2063d1dde356abf59ca3bbc849f/website/static/operator_logo_sdk_color.svg" alt="Operator SDK Logo" height="80">
  <img src="https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg" alt="Kubernetes Logo" height="80">
</div>

# N8n Kubernetes Operator

A Kubernetes operator that automates the deployment and management of n8n workflow automation instances on Kubernetes clusters.

[![Documentation Status](https://readthedocs.org/projects/n8n-operator/badge/?version=latest)](https://n8n-operator.readthedocs.io/en/latest/?badge=latest)

## Key Features

- **Automated Deployment**: Simplified n8n instance deployment with PostgreSQL database configuration
- **Traffic Routing**: Support for both Kubernetes Ingress and Gateway API HTTPRoute
- **Persistent Storage**: Automatic volume provisioning and configurable storage management
- **Security**: Non-root container execution with automated TLS configuration
- **Monitoring**: Prometheus metrics integration for operational visibility

## Quick Start

Create a basic n8n instance:

```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-sample
spec:
  database:
    postgres:
      host: "postgres-host"
      port: 5432
      database: "n8n"
      user: "n8n-user"
      password: "password"
  ingress:
    enable: true
    ingressClassName: "nginx"
```

## 📖 Documentation

**[Read the full documentation →](https://n8n-operator.readthedocs.io/)**

For comprehensive guides and detailed configuration options, visit our complete documentation:

- **[Installation Guide](https://n8n-operator.readthedocs.io/en/latest/installation/)** - Get started with installation and setup
- **[Configuration Guide](https://n8n-operator.readthedocs.io/en/latest/configuration/)** - Complete configuration reference
- **[API Reference](https://n8n-operator.readthedocs.io/en/latest/api/)** - Custom Resource Definitions and specifications

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.
