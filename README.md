<div align="center">
  <img src="https://raw.githubusercontent.com/n8n-io/n8n/master/assets/n8n-logo.png" alt="n8n Logo" height="100">
  <br><br>
  <img src="https://raw.githubusercontent.com/operator-framework/operator-sdk/4407fd6723aef2063d1dde356abf59ca3bbc849f/website/static/operator_logo_sdk_color.svg" alt="Operator SDK Logo" height="80">
  <img src="https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg" alt="Kubernetes Logo" height="80">
</div>

# N8n Kubernetes Operator

A Kubernetes operator to manage N8N workflow automation instances.

[![Documentation Status](https://readthedocs.org/projects/n8n-operator/badge/?version=latest)](https://n8n-operator.readthedocs.io/en/latest/?badge=latest)

## Overview

The n8n-operator automates the deployment and management of n8n workflow automation instances on Kubernetes. It provides:

- Automated deployment of n8n instances with PostgreSQL database configuration
- Support for both Kubernetes Ingress and Gateway API HTTPRoute for traffic routing
- Secure defaults with non-root container execution
- Automated TLS configuration for secure access

## Quick Start

Create an n8n instance:
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
    hostname: "n8n.example.com"
```

## Documentation

For detailed information about installation, configuration, and usage, please visit our [documentation](https://n8n-operator.readthedocs.io/).

Key documentation sections:
- [Installation Guide](https://n8n-operator.readthedocs.io/en/latest/installation/)
- [Configuration Guide](https://n8n-operator.readthedocs.io/en/latest/configuration/)
- [API Reference](https://n8n-operator.readthedocs.io/en/latest/api/)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the terms of the [LICENSE](LICENSE) file included in the repository.
