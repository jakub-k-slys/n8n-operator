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

## Installation

### Prerequisites

Before installing the n8n-operator, ensure you have:

- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- Cluster admin privileges or appropriate RBAC permissions
- For Gateway API support: Gateway API CRDs installed in the cluster

### Method 1: Using OLM Catalog (Recommended)

The easiest way to install the n8n-operator is through the Operator Lifecycle Manager (OLM) catalog.

#### Step 1: Install OLM (if not already installed)

```bash
curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0
```

#### Step 2: Install the n8n-operator from catalog

```bash
# Create the operator namespace
kubectl create namespace n8n-operator-system

# Install the catalog source
kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: n8n-operator-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: ghcr.io/jakub-k-slys/n8n-operator-catalog:v0.0.1
  displayName: N8n Operator Catalog
  publisher: jakub-k-slys
EOF

# Create subscription to install the operator
kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: n8n-operator
  namespace: n8n-operator-system
spec:
  channel: alpha
  name: n8n-operator
  source: n8n-operator-catalog
  sourceNamespace: olm
EOF
```

#### Step 3: Verify installation

```bash
kubectl get pods -n n8n-operator-system
kubectl get csv -n n8n-operator-system
```

### Method 2: Direct Installation

If you prefer not to use OLM, you can install the operator directly:

#### Step 1: Install CRDs

```bash
kubectl apply -f https://raw.githubusercontent.com/jakub-k-slys/n8n-operator/main/config/crd/bases/n8n.slys.dev_n8ns.yaml
```

#### Step 2: Install the operator

```bash
kubectl apply -f https://raw.githubusercontent.com/jakub-k-slys/n8n-operator/main/dist/install.yaml
```

### Method 3: From Source

For development or customization:

```bash
# Clone the repository
git clone https://github.com/jakub-k-slys/n8n-operator.git
cd n8n-operator

# Install CRDs
make install

# Deploy the operator
make deploy IMG=ghcr.io/jakub-k-slys/n8n-operator:v0.0.1
```

### Verification

After installation, verify the operator is running:

```bash
kubectl get pods -n n8n-operator-system
kubectl get crd | grep n8n
```

You should see the n8n-operator pod running and the `n8ns.n8n.slys.dev` CRD installed.

## Quick Start

Create an n8n instance with Ingress:
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
      ssl: false
  ingress:
    enable: true
    ingressClassName: "nginx"
```

Alternatively, using Gateway API HTTPRoute:
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
      ssl: false
  httpRoute:
    enable: true
    gatewayRef:
      name: "gateway"
      namespace: "default"
```

## Configuration Reference

### Complete Configuration Example

```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-complete
spec:
  # Database configuration (required)
  database:
    postgres:
      host: "postgres-host"
      port: 5432
      database: "n8n"
      user: "n8n-user"
      password: "password"
      ssl: true  # Optional: enable SSL connection
  
  # Ingress configuration (optional)
  # Note: Cannot be used together with httpRoute
  ingress:
    enable: true
    ingressClassName: "nginx"
    tls:
      - hosts:
          - "n8n.example.com"
        secretName: "n8n-tls"
  
  # HTTPRoute configuration (optional)
  # Note: Cannot be used together with ingress
  httpRoute:
    enable: false
    gatewayRef:
      name: "gateway"
      namespace: "default"
  
  # Persistent storage configuration (optional)
  persistentStorage:
    enable: true
    storageClassName: "standard"
    size: "10Gi"  # Defaults to "10Gi" if not specified
  
  # Metrics configuration (optional)
  metrics:
    enable: true
  
  # Hostname configuration (optional)
  hostname:
    enable: true
    url: "n8n.example.com"
```

### Database Configuration

The `database` field is required and currently supports PostgreSQL:

```yaml
database:
  postgres:
    host: "postgres-host"        # Required: PostgreSQL host
    port: 5432                   # Required: PostgreSQL port (1-65535)
    database: "n8n"             # Required: Database name
    user: "n8n-user"            # Required: Database user
    password: "password"         # Required: Database password
    ssl: false                   # Optional: Enable SSL connection
```

### Traffic Routing Configuration

**Important**: Ingress and HTTPRoute cannot both be enabled simultaneously.

#### Kubernetes Ingress

```yaml
ingress:
  enable: true                   # Required when using ingress
  ingressClassName: "nginx"      # Optional: IngressClass to use
  tls:                          # Optional: TLS configuration
    - hosts:
        - "n8n.example.com"
      secretName: "n8n-tls"
```

#### Gateway API HTTPRoute

```yaml
httpRoute:
  enable: true                   # Required when using HTTPRoute
  gatewayRef:                   # Required when HTTPRoute is enabled
    name: "gateway"             # Required: Gateway name
    namespace: "default"        # Optional: Gateway namespace
```

### Storage Configuration

```yaml
persistentStorage:
  enable: true                   # Required when using persistent storage
  storageClassName: "standard"   # Optional: StorageClass to use
  size: "10Gi"                  # Optional: Volume size (defaults to "10Gi")
```

### Metrics Configuration

```yaml
metrics:
  enable: true                   # Required when enabling metrics
```

### Hostname Configuration

```yaml
hostname:
  enable: true                   # Required when using hostname
  url: "n8n.example.com"        # Required when hostname is enabled
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
