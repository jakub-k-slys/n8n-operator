# Configuration Guide

## Traffic Routing Options

The n8n operator supports two methods for routing traffic to n8n instances:

### 1. Kubernetes Ingress

Standard Kubernetes Ingress configuration with the following features:
- Configurable ingress class
- Optional TLS configuration
- Hostname-based routing

Example configuration:
```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-sample
spec:
  ingress:
    enable: true
    hostname: "n8n.example.com"
    ingressClassName: "nginx"
    tls:
      - hosts:
          - "n8n.example.com"
        secretName: "n8n-tls"
```

### 2. Gateway API HTTPRoute

Modern Gateway API routing (v1) configuration offering:
- Support for Gateway API features
- Hostname-based routing
- Integration with Gateway API implementations

Example configuration:
```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-sample
spec:
  httpRoute:
    enable: true
    hostname: "n8n.example.com"
    gatewayRef:
      name: "gateway"
      namespace: "default"
```

**Note:** Only one routing method (Ingress or HTTPRoute) can be enabled at a time.

## Database Configuration

### PostgreSQL Integration

The operator supports PostgreSQL database configuration with the following options:
- Host and port configuration
- Database name
- User authentication
- SSL support

**Note:** Database configuration is optional. If no database configuration is provided, n8n will use its default SQLite database for development/testing purposes.

Example configuration:
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
```

## Persistent Storage

Configure persistent storage for n8n data with the following options:

- Enable/disable persistent storage
- Custom storage class selection
- Configurable storage size (defaults to 10Gi)
- Data persistence at `/home/node/.n8n`
- Automatic PVC creation and management

Example configuration:
```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-sample
spec:
  persistentStorage:
    enable: true
    storageClassName: "standard"
    size: "10Gi"  # Optional, defaults to "10Gi"
```

## Security Configuration

The n8n operator implements several security features:

1. Non-root Container Execution
   - Containers run as non-root by default
   - Enhanced security through principle of least privilege

2. TLS Configuration
   - Automated TLS certificate management
   - Secure HTTPS access to n8n instances

3. Database Security
   - Secure database connections
   - Optional SSL support for database communication

## Complete Configuration Example

Here's a complete example combining all major features:

```yaml
apiVersion: n8n.slys.dev/v1alpha1
kind: N8n
metadata:
  name: n8n-complete
spec:
  # Database Configuration
  database:
    postgres:
      host: "postgres-host"
      port: 5432
      database: "n8n"
      user: "n8n-user"
      password: "password"
      ssl: true

  # Ingress Configuration
  ingress:
    enable: true
    hostname: "n8n.example.com"
    ingressClassName: "nginx"
    tls:
      - hosts:
          - "n8n.example.com"
        secretName: "n8n-tls"

  # Persistent Storage Configuration
  persistentStorage:
    enable: true
    storageClassName: "standard"
    size: "20Gi"
