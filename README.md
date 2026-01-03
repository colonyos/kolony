# Kolony

Kolony is a Kubernetes operator for managing ColonyOS resources declaratively. It enables GitOps workflows for ColonyOS by providing Custom Resource Definitions (CRDs) that sync with the ColonyOS API.

## Overview

Kolony bridges Kubernetes and ColonyOS, allowing you to:

- Define BlueprintDefinitions and Blueprints as Kubernetes resources
- Submit jobs to ColonyOS executors via ColonyProcess resources
- Track process state and output in Kubernetes status fields
- Use GitOps tools (ArgoCD, Flux) to manage ColonyOS infrastructure

## Custom Resources

| CRD | Description |
|-----|-------------|
| `BlueprintDefinition` | Defines a schema for a type of blueprint (similar to CRDs for custom resources) |
| `Blueprint` | An instance of a BlueprintDefinition representing desired state |
| `ColonyProcess` | A job-like resource that submits functions to ColonyOS and tracks execution |

## Prerequisites

- Kubernetes cluster v1.24+
- Helm v3.0+
- Access to a ColonyOS server
- ColonyOS executor private key

## Installation

### Using Helm

```bash
# Add the kolony namespace
kubectl create namespace kolony

# Install the operator
helm install kolony ./helm/kolony --namespace kolony
```

### From Pre-built Image

The operator image is available at `colonyos/kolony:latest`:

```bash
helm install kolony ./helm/kolony \
  --namespace kolony \
  --set image.repository=colonyos/kolony \
  --set image.tag=latest
```

## Configuration

### ColonyOS Credentials

The operator reads ColonyOS credentials from a Kubernetes Secret named `colonyos-credentials` in each namespace where you create Kolony resources.

Create the secret in your target namespace:

```bash
kubectl create secret generic colonyos-credentials \
  --namespace=<your-namespace> \
  --from-literal=serverHost="<colonyos-server-host>" \
  --from-literal=serverPort="<colonyos-server-port>" \
  --from-literal=tls="<true-or-false>" \
  --from-literal=colonyName="<your-colony-name>" \
  --from-literal=colonyPrvKey="<your-colony-private-key>" \
  --from-literal=executorPrvKey="<your-executor-private-key>"
```

Or apply a YAML file:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: colonyos-credentials
  namespace: <your-namespace>
type: Opaque
stringData:
  serverHost: "<colonyos-server-host>"
  serverPort: "<colonyos-server-port>"
  tls: "<true-or-false>"
  colonyName: "<your-colony-name>"
  colonyPrvKey: "<your-colony-private-key>"
  executorPrvKey: "<your-executor-private-key>"
```

**Note:** Two private keys are required:
- `colonyPrvKey`: Colony-level key for BlueprintDefinition operations
- `executorPrvKey`: Executor-level key for Blueprint and ColonyProcess operations

**Important:** Never commit credentials to version control. Use sealed-secrets or external secret management in production.

## Examples

The `examples/` directory contains ready-to-use YAML files:

| Example | Description |
|---------|-------------|
| `credentials-secret.yaml` | Template for ColonyOS credentials |
| `blueprintdefinition-executor.yaml` | Define executor deployment schema |
| `blueprintdefinition-gpu-cluster.yaml` | Define GPU cluster schema |
| `blueprint-llm-executor.yaml` | Deploy LLM inference executors |
| `blueprint-etl-workers.yaml` | Deploy ETL processing workers |
| `colonyprocess-simple.yaml` | Simple hello world job |
| `colonyprocess-ml-training.yaml` | ML training with GPU requirements |
| `colonyprocess-data-export.yaml` | Data export job |
| `colonyprocess-batch-render.yaml` | 3D rendering job |

### Quick Start with Examples

```bash
# 1. Create credentials (edit with your values first!)
cp examples/credentials-secret.yaml my-credentials.yaml
# Edit my-credentials.yaml with your ColonyOS credentials
kubectl apply -f my-credentials.yaml

# 2. Create a BlueprintDefinition
kubectl apply -f examples/blueprintdefinition-executor.yaml

# 3. Create a Blueprint
kubectl apply -f examples/blueprint-llm-executor.yaml

# 4. Submit a process
kubectl apply -f examples/colonyprocess-simple.yaml

# 5. Watch status
kubectl get colonyprocess -w
```

### Checking Status

```bash
# List all resources
kubectl get blueprintdefinitions
kubectl get blueprints
kubectl get colonyprocesses

# Get detailed status
kubectl describe colonyprocess <name>

# View process output
kubectl get colonyprocess <name> -o jsonpath='{.status.output}'
```

### Status Fields

ColonyProcess status includes:
- `processId`: ColonyOS process ID
- `state`: Pending, Waiting, Running, Success, or Failed
- `assignedExecutor`: Which executor is running the job
- `output`: Process output array
- `errors`: Any error messages

See [examples/README.md](examples/README.md) for more details

## Helm Values

Key configuration options in `values.yaml`:

```yaml
image:
  repository: colonyos/kolony
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 1

resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi

leaderElection:
  enabled: true

metrics:
  enabled: true
  port: 8080

health:
  port: 8081
```

## Uninstallation

```bash
# Remove the operator
helm uninstall kolony --namespace kolony

# Remove the namespace
kubectl delete namespace kolony

# Remove CRDs (this will delete all Kolony resources!)
kubectl delete crd blueprintdefinitions.colony.colonyos.io
kubectl delete crd blueprints.colony.colonyos.io
kubectl delete crd colonyprocesses.colony.colonyos.io
```

## Development

### Building from Source

```bash
# Build the binary
make build

# Build the container
docker build -t colonyos/kolony:latest .

# Push to registry
docker push colonyos/kolony:latest
```

### Running Locally

```bash
# Install CRDs
make install

# Run the controller locally
make run
```

### Running Tests

```bash
make test
```

## Architecture

See [docs/Design.md](docs/Design.md) for detailed architecture documentation including:

- CRD specifications
- Controller reconciliation loops
- Namespace-to-colony mapping
- GitOps integration patterns
- Process lifecycle management

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
