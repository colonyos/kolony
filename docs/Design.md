# Kolony Design Document

Kolony is a Kubernetes operator that enables declarative management of ColonyOS resources using Custom Resource Definitions (CRDs). This document describes the architecture, design decisions, and implementation details.

## Overview

```mermaid
flowchart TB
    subgraph Kubernetes
        CRDs[Custom Resources]
        Controller[Kolony Controller]
        Secrets[Credentials Secret]
    end

    subgraph ColonyOS
        API[Colonies API]
        Blueprints[(Blueprints)]
        Processes[(Processes)]
        Executors[Executors]
    end

    subgraph GitOps
        Git[Git Repository]
        ArgoCD[ArgoCD / Flux]
    end

    Git -->|sync| ArgoCD
    ArgoCD -->|apply| CRDs
    Controller -->|watch| CRDs
    Controller -->|read| Secrets
    Controller -->|API calls| API
    API --> Blueprints
    API --> Processes
    Executors -->|execute| Processes
```

## Custom Resource Definitions

Kolony defines three primary CRDs that map to ColonyOS concepts:

```mermaid
classDiagram
    class BlueprintDefinition {
        +string kind
        +Metadata metadata
        +DefinitionSpec spec
        +DefinitionStatus status
    }

    class Blueprint {
        +string kind
        +Metadata metadata
        +BlueprintSpec spec
        +BlueprintStatus status
    }

    class ColonyProcess {
        +string funcName
        +Metadata metadata
        +ProcessSpec spec
        +ProcessStatus status
    }

    BlueprintDefinition <|-- Blueprint : defines schema
    Blueprint ..> ColonyProcess : triggers reconciliation
```

### BlueprintDefinition

Defines a schema for a type of blueprint, similar to how a CRD defines a schema for custom resources.

```yaml
apiVersion: colonyos.io/v1
kind: BlueprintDefinition
metadata:
  name: gpu-executor-def
  namespace: ml-team
spec:
  kind: GPUExecutor
  names:
    singular: gpuexecutor
    plural: gpuexecutors
  handler:
    executorType: docker-reconciler
    functionName: reconcile
    reconcileInterval: 60
status:
  definitionId: abc123...
  registered: true
  lastSyncTime: "2026-01-03T10:00:00Z"
```

### Blueprint

An instance of a BlueprintDefinition, representing desired state for a resource.

```yaml
apiVersion: colonyos.io/v1
kind: Blueprint
metadata:
  name: llm-cluster
  namespace: ml-team
spec:
  kind: GPUExecutor
  replicas: 10
  image: colonyos/llm-executor:v2
  resources:
    gpu: 1
    memory: 32Gi
status:
  blueprintId: def456...
  generation: 5
  synced: true
  readyReplicas: 10
  lastReconcileTime: "2026-01-03T10:05:00Z"
```

### ColonyProcess

A job-like resource that submits a function to ColonyOS and tracks its execution.

```yaml
apiVersion: colonyos.io/v1
kind: ColonyProcess
metadata:
  name: train-model-run-42
  namespace: ml-team
spec:
  funcName: train
  executorType: gpu-executor
  args:
    - "--epochs=100"
  kwargs:
    model: gpt-neo
    dataset: s3://bucket/data
  maxWaitTime: 3600
  maxExecTime: 86400
status:
  processId: proc789...
  state: RUNNING
  assignedExecutor: gpu-node-3
  startTime: "2026-01-03T10:00:00Z"
  completionTime: null
  output: []
```

## Controller Architecture

```mermaid
flowchart LR
    subgraph Controller
        Reconciler[Reconciler]
        Client[ColonyOS Client]
        Cache[Informer Cache]
    end

    subgraph K8s API
        Watch[Watch API]
        CRUD[CRUD API]
    end

    Watch -->|events| Cache
    Cache -->|work items| Reconciler
    Reconciler -->|read/write| CRUD
    Reconciler -->|sync| Client
```

### Reconciliation Loop

Each controller follows the standard Kubernetes reconciliation pattern:

```mermaid
sequenceDiagram
    participant K8s as Kubernetes API
    participant R as Reconciler
    participant C as ColonyOS Client
    participant CO as ColonyOS API

    K8s->>R: Reconcile(name, namespace)
    R->>K8s: Get CR
    K8s-->>R: CR spec + status

    alt CR being deleted
        R->>C: Delete resource
        C->>CO: DELETE /api
        CO-->>C: OK
        R->>K8s: Remove finalizer
    else CR created/updated
        R->>C: Get current state
        C->>CO: GET /api
        CO-->>C: Current state

        alt State differs
            R->>C: Update state
            C->>CO: POST/PUT /api
            CO-->>C: New state
        end

        R->>K8s: Update status
    end

    R-->>K8s: Requeue after interval
```

## Namespace and Colony Mapping

Each Kubernetes namespace maps to a ColonyOS colony through a credentials secret:

```mermaid
flowchart TB
    subgraph ns1[Namespace: ml-team]
        Secret1[Secret: colonyos-credentials]
        BP1[Blueprint: gpu-cluster]
        BP2[Blueprint: cpu-cluster]
    end

    subgraph ns2[Namespace: data-team]
        Secret2[Secret: colonyos-credentials]
        BP3[Blueprint: etl-workers]
    end

    subgraph ColonyOS
        Colony1[Colony: ml-prod]
        Colony2[Colony: data-prod]
    end

    Secret1 -->|colonyName: ml-prod| Colony1
    BP1 --> Colony1
    BP2 --> Colony1

    Secret2 -->|colonyName: data-prod| Colony2
    BP3 --> Colony2
```

### Credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: colonyos-credentials
  namespace: ml-team
type: Opaque
data:
  serverHost: c2VydmVyLmNvbG9ueW9zLmlv
  serverPort: NDQz
  tls: dHJ1ZQ==
  colonyName: bWwtcHJvZA==
  colonyPrvKey: <base64-encoded-key>
  executorPrvKey: <base64-encoded-key>
```

## GitOps Integration

```mermaid
flowchart LR
    subgraph Development
        Dev[Developer]
        PR[Pull Request]
    end

    subgraph GitOps
        Repo[Git Repository]
        ArgoCD[ArgoCD]
    end

    subgraph Kubernetes
        Kolony[Kolony Operator]
        CRs[Custom Resources]
    end

    subgraph ColonyOS
        API[ColonyOS API]
        Infra[Distributed Infrastructure]
    end

    Dev -->|commit| PR
    PR -->|merge| Repo
    Repo -->|sync| ArgoCD
    ArgoCD -->|apply| CRs
    Kolony -->|watch| CRs
    Kolony -->|reconcile| API
    API -->|manage| Infra
```

### Repository Structure

```
gitops-repo/
├── clusters/
│   └── production/
│       └── ml-team/
│           ├── namespace.yaml
│           ├── credentials.yaml      # SealedSecret
│           ├── definitions/
│           │   └── gpu-executor.yaml
│           └── blueprints/
│               ├── llm-cluster.yaml
│               └── training-cluster.yaml
└── jobs/
    └── ml-team/
        └── train-run-42.yaml
```

## Process Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Pending: CR Created
    Pending --> Submitted: Controller submits to ColonyOS
    Submitted --> Waiting: Process in queue
    Waiting --> Running: Executor assigned
    Running --> Success: Completed successfully
    Running --> Failed: Error or timeout
    Success --> [*]: CR status updated
    Failed --> [*]: CR status updated

    note right of Pending: Finalizer added
    note right of Submitted: processId stored in status
    note right of Running: Polling for updates
```

## Leader Election and High Availability

```mermaid
flowchart TB
    subgraph Pod1[Pod: kolony-controller-0]
        C1[Controller]
        L1[Leader Election]
    end

    subgraph Pod2[Pod: kolony-controller-1]
        C2[Controller]
        L2[Leader Election]
    end

    subgraph Pod3[Pod: kolony-controller-2]
        C3[Controller]
        L3[Leader Election]
    end

    Lease[Lease: kolony-leader]

    L1 -->|acquire| Lease
    L2 -.->|standby| Lease
    L3 -.->|standby| Lease

    C1 -->|active| Reconcile[Reconciliation]
    C2 -.->|inactive| Reconcile
    C3 -.->|inactive| Reconcile
```

## Error Handling and Retry

```mermaid
flowchart TD
    Start[Reconcile Request] --> Fetch[Fetch CR]
    Fetch --> Check{CR Exists?}

    Check -->|No| Done[Done]
    Check -->|Yes| Sync[Sync to ColonyOS]

    Sync --> Result{Success?}

    Result -->|Yes| Update[Update Status]
    Update --> Requeue[Requeue after interval]
    Requeue --> Done

    Result -->|Transient Error| Retry[Exponential Backoff]
    Retry --> Done

    Result -->|Permanent Error| MarkFailed[Mark as Failed]
    MarkFailed --> Done
```

## Metrics and Observability

The operator exposes Prometheus metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `kolony_reconcile_total` | Counter | Total reconciliations by resource type and result |
| `kolony_reconcile_duration_seconds` | Histogram | Time spent in reconciliation |
| `kolony_colonyos_requests_total` | Counter | API calls to ColonyOS by method and status |
| `kolony_resources_total` | Gauge | Current count of managed resources by type |
| `kolony_process_state` | Gauge | Current process states (waiting, running, etc.) |

## Future Considerations

### ColonyWorkflow CRD

Support for process graphs as a single resource:

```yaml
apiVersion: colonyos.io/v1
kind: ColonyWorkflow
metadata:
  name: ml-pipeline
spec:
  processes:
    - name: preprocess
      funcName: preprocess
      kwargs:
        input: s3://data/raw
    - name: train
      funcName: train
      dependencies: [preprocess]
    - name: evaluate
      funcName: evaluate
      dependencies: [train]
```

### Webhook Integration

- Validating webhooks for CR validation
- Mutating webhooks for defaults injection
- Conversion webhooks for version upgrades

### Multi-Cluster Support

Federation of ColonyOS resources across multiple Kubernetes clusters with a central ColonyOS control plane.
