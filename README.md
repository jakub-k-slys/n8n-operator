# n8n-operator

A Kubernetes operator to manage n8n workflow automation instances with support for both Ingress and Gateway API routing.

## Description

The n8n-operator automates the deployment and management of n8n workflow automation instances on Kubernetes. It provides:

- Automated deployment of n8n instances with PostgreSQL database configuration
- Support for both Kubernetes Ingress and Gateway API HTTPRoute for traffic routing
- Secure defaults with non-root container execution
- Automated TLS configuration for secure access

## Features

### Traffic Routing Options

The operator supports two methods for routing traffic to n8n instances:

1. **Kubernetes Ingress**
   - Standard Kubernetes Ingress resource
   - Configurable ingress class
   - Optional TLS configuration
   - Hostname-based routing

2. **Gateway API HTTPRoute**
   - Modern Gateway API routing (v1)
   - Support for Gateway API features
   - Hostname-based routing
   - Integration with Gateway API implementations

Note: Only one routing method (Ingress or HTTPRoute) can be enabled at a time.

### Database Configuration

- PostgreSQL database integration
- Configurable database connection parameters
- SSL support for database connections

### Persistent Storage

- Optional persistent storage for n8n data
- Configurable storage class and size
- Data persisted at `/home/node/.n8n`
- Automatic PVC creation and management

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- For Gateway API support: Gateway API CRDs installed in the cluster

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/n8n-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/n8n-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### Example Usage

Create an n8n instance with Ingress:

```yaml
apiVersion: cache.slys.dev/v1alpha1
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
    hostname: "n8n.example.com"
    ingressClassName: "nginx"
    tls:
      - hosts:
          - "n8n.example.com"
        secretName: "n8n-tls"
```

Or with Gateway API HTTPRoute:

```yaml
apiVersion: cache.slys.dev/v1alpha1
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
    hostname: "n8n.example.com"
    gatewayRef:
      name: "gateway"
      namespace: "default"
  persistentStorage:
    enable: true
    storageClassName: "standard"
    size: "1Gi"
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/n8n-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/n8n-operator/<tag or branch>/dist/install.yaml
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
