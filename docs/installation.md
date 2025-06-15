# Installation Guide

## Prerequisites

Before installing the n8n-operator, ensure you have:

- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- Cluster admin privileges or appropriate RBAC permissions
- For Gateway API support: Gateway API CRDs installed in the cluster

## Installation Methods

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
  name: operator-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: ghcr.io/jakub-k-slys/operator-catalog:v0.0.1
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
  source: operator-catalog
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

## Development Installation

For developers who want to build and deploy the operator from source:

### Prerequisites for Development

- go version v1.22.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Building and Pushing the Image

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/n8n-operator:tag
```

**NOTE:** This image must be published in your specified registry. Ensure you have proper permissions to pull the image from your working environment.

### Installing CRDs

Install the Custom Resource Definitions (CRDs) into your cluster:

```sh
make install
```

### Deploying the Controller

Deploy the controller to the cluster with your image:

```sh
make deploy IMG=<some-registry>/n8n-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin privileges or be logged in as admin.

## Project Distribution

### Building the Installer

1. Build the installer for your image:

```sh
make build-installer IMG=<some-registry>/n8n-operator:tag
```

This generates an 'install.yaml' file in the dist directory containing all resources built with Kustomize.

### Using the Installer

Users can install the project using:

```sh
kubectl apply -f https://raw.githubusercontent.com/jakub-k-slys/n8n-operator/main/dist/install.yaml
```

## Uninstallation

### Remove Instances

Delete the Custom Resources:

```sh
kubectl delete -k config/samples/
```

### Remove CRDs

Delete the Custom Resource Definitions:

```sh
make uninstall
```

### Remove Controller

Undeploy the controller:

```sh
make undeploy
