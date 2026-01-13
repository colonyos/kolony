# Kolony

Kolony is a Kubernetes operator for managing ColonyOS resources declaratively. It enables GitOps workflows for ColonyOS by providing Custom Resource Definitions (CRDs) that sync with the ColonyOS API.

## Why Kolony?

### Kubernetes Reconciliation for Cyber-Physical Systems

Kubernetes has revolutionized how we manage cloud infrastructure through declarative configuration and continuous reconciliation. However, traditional Kubernetes is limited to managing resources that run *within* the cluster - containers, services, and cloud-native workloads.

**Kolony extends the Kubernetes reconciliation model to the physical world.**

Through ColonyOS executors, Kolony can reconcile state across systems that cannot run Kubernetes themselves:

- **IoT and Embedded Systems** - Sensors, PLCs, industrial controllers, and edge devices with limited compute resources
- **Legacy Infrastructure** - SCADA systems, proprietary hardware, and brownfield installations
- **Remote and Harsh Environments** - Mining sites, offshore platforms, Arctic installations with intermittent connectivity
- **Specialized Hardware** - GPUs, FPGAs, quantum computers, and domain-specific accelerators

### Unified Control Plane for Heterogeneous Infrastructure

Modern organizations operate across multiple compute paradigms:

| Environment | Traditional Management | With Kolony |
|-------------|----------------------|-------------|
| Cloud (AWS, GCP, Azure) | Terraform, CloudFormation | Kubernetes + Kolony |
| On-premise Kubernetes | kubectl, Helm | Kubernetes + Kolony |
| HPC Clusters | SLURM scripts, manual | Kubernetes + Kolony |
| Edge/IoT | Vendor-specific tools | Kubernetes + Kolony |
| Industrial OT | SCADA, proprietary | Kubernetes + Kolony |

Kolony provides a **single declarative interface** for all these environments, managed through familiar Kubernetes tooling.

### HPC Integration

High-Performance Computing clusters represent massive computational resources that traditionally operate in isolation from cloud-native infrastructure. Kolony bridges this gap:

- **Submit HPC jobs from Kubernetes** - Define SLURM/PBS jobs as ColonyProcess resources
- **Declarative cluster configuration** - Manage HPC software stacks, modules, and environments via Blueprints
- **Hybrid workflows** - Orchestrate pipelines that span Kubernetes and HPC seamlessly
- **Resource federation** - Treat HPC clusters as executor pools alongside cloud resources
- **Burst to HPC** - Automatically offload compute-intensive workloads when cluster capacity is available

### GitOps for Everything

By representing physical infrastructure as Kubernetes resources, Kolony enables:

- **Version-controlled infrastructure** - Git history for industrial control systems, not just cloud resources
- **Pull request workflows** - Review changes to PLC configurations like code
- **Audit trails** - Full traceability of who changed what, when
- **Rollback capabilities** - Revert physical infrastructure to previous states
- **Environment promotion** - Dev -> Staging -> Production for cyber-physical systems

### Bridge Between IT and OT

Operational Technology (OT) - the systems that control physical processes - has historically been siloed from Information Technology (IT). Kolony bridges this divide:

- **Common tooling** - Platform engineers and OT engineers use the same kubectl/GitOps workflows
- **Unified observability** - Correlate cloud metrics with industrial sensor data
- **Consistent security model** - Apply Kubernetes RBAC to physical infrastructure access
- **Integrated CI/CD** - Deploy firmware updates and control system changes through the same pipelines as application code

### Edge Computing with Resilience

For remote sites with unreliable connectivity, Kolony combined with ColonyOS edge executors provides:

- **Offline operation** - Edge executors continue operating during network outages
- **Store-and-forward** - Jobs queue locally and sync when connectivity returns
- **Local reconciliation** - Critical control loops run at the edge, not dependent on cloud
- **Eventual consistency** - Kubernetes desired state propagates to edge when possible

### Eclipse Arrowhead Integration

[Eclipse Arrowhead](https://arrowhead.eu/eclipse-arrowhead-2/) is a service-oriented framework for industrial automation and Industry 4.0, enabling interoperability across heterogeneous IoT/OT environments. It organizes systems into **Local Clouds** - closed, local industrial networks at the edge - each containing three mandatory core systems:

- **Service Registry** - Service discovery for industrial microsystems
- **Orchestrator** - Determines which service instances consumers should use
- **Authorization** - X.509 certificate and JWT-based access control

Kolony enables **Kubernetes-native management of Arrowhead infrastructure**:

| Arrowhead Concept | Kolony Blueprint | Description |
|-------------------|------------------|-------------|
| Local Cloud | `ArrowheadLocalCloud` | Deploy and configure an Arrowhead local cloud |
| Service Registry | `ArrowheadService` | Register/deregister services declaratively |
| Orchestration Rules | `ArrowheadOrchestration` | Define consumer-provider bindings as code |
| Authorization | `ArrowheadAuthorization` | Manage access policies via GitOps |
| Inter-Cloud | `ArrowheadGatekeeper` | Configure cross-cloud service exchange |

**Why this matters:**

- **GitOps for industrial automation** - Version-controlled Arrowhead configurations with PR-based review
- **Unified management** - Same kubectl/Helm workflows for cloud services and factory floor
- **Declarative service mesh** - Define the entire Arrowhead topology as Kubernetes resources
- **Hybrid architectures** - Orchestrate workloads across Kubernetes, Arrowhead local clouds, and HPC
- **Protocol bridging** - ColonyOS executors can translate between Arrowhead services and OPC-UA, Modbus, Z-Wave, IO-Link

Example: Deploying an Arrowhead service via Kolony:

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  name: temperature-sensor-service
  namespace: factory-floor
spec:
  kind: ArrowheadService
  data:
    localCloud: "production-line-1"
    service:
      name: "temperature-monitoring"
      uri: "/sensor/temperature"
      interface: "HTTP-SECURE-JSON"
      version: 1
    provider:
      systemName: "plc-sensor-gateway"
      address: "192.168.10.50"
      port: 8443
    metadata:
      unit: "celsius"
      location: "furnace-zone-a"
```

This enables industrial systems to be managed with the same rigor and tooling as cloud-native applications, bringing DevOps practices to the factory floor.

### Key Benefits

| Benefit | Description |
|---------|-------------|
| **Declarative** | Define desired state, let reconcilers handle the how |
| **Idempotent** | Apply the same configuration repeatedly without side effects |
| **Self-healing** | Automatic drift detection and correction |
| **Observable** | Standard Kubernetes status, events, and metrics |
| **Extensible** | Add new reconciler types for any domain |
| **Secure** | Leverage Kubernetes RBAC, network policies, and secrets |
| **Portable** | Same manifests work across any Kubernetes cluster |

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
| `colonyprocess-simple.yaml` | Simple hello world container |
| `colonyprocess-docker-executor.yaml` | Container with echo, date, hostname |

### Quick Start with Examples

```bash
# 1. Create credentials (edit with your values first!)
cp examples/credentials-secret.yaml my-credentials.yaml
# Edit my-credentials.yaml with your ColonyOS credentials
kubectl apply -f my-credentials.yaml

# 2. Submit a process (use 'create' with generateName)
kubectl create -f examples/colonyprocess-simple.yaml

# 3. Watch status
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

# View full status
kubectl get colonyprocess <name> -o yaml
```

### Status Fields

ColonyProcess status includes:
- `processId`: ColonyOS process ID
- `state`: Pending, Waiting, Running, Success, or Failed
- `assignedExecutor`: Which executor is running the job
- `output`: Explicit return values from the executor (not stdout)
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

MIT License. See [LICENSE](LICENSE) for details.
