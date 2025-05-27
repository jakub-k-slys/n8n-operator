# Installation Guide

## Prerequisites

Before installing the n8n operator, ensure you have:

- go version v1.22.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- For Gateway API support: Gateway API CRDs installed in the cluster

## Installation Steps

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
kubectl apply -f https://raw.githubusercontent.com/<org>/n8n-operator/<tag or branch>/dist/install.yaml
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
