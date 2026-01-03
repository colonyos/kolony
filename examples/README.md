# Kolony Examples

This directory contains example YAML files demonstrating how to use Kolony CRDs.

## Prerequisites

Before applying these examples, ensure you have:

1. Kolony operator installed in your cluster
2. A `colonyos-credentials` secret in your target namespace

## Files

### Credentials

| File | Description |
|------|-------------|
| `credentials-secret.yaml` | Template for ColonyOS credentials secret |

### BlueprintDefinitions

| File | Description |
|------|-------------|
| `blueprintdefinition-executor.yaml` | Definition for executor deployments |
| `blueprintdefinition-gpu-cluster.yaml` | Definition for GPU compute clusters |

### Blueprints

| File | Description |
|------|-------------|
| `blueprint-llm-executor.yaml` | LLM inference executor deployment |
| `blueprint-etl-workers.yaml` | ETL data processing workers |

### ColonyProcesses

| File | Description |
|------|-------------|
| `colonyprocess-simple.yaml` | Basic hello world example |
| `colonyprocess-ml-training.yaml` | ML training with GPU requirements |
| `colonyprocess-data-export.yaml` | Data export to cloud storage |
| `colonyprocess-batch-render.yaml` | 3D rendering job |

## Quick Start

```bash
# 1. Create credentials secret (edit with your values first!)
kubectl apply -f credentials-secret.yaml

# 2. Create a BlueprintDefinition
kubectl apply -f blueprintdefinition-executor.yaml

# 3. Create a Blueprint instance
kubectl apply -f blueprint-llm-executor.yaml

# 4. Submit a process
kubectl apply -f colonyprocess-simple.yaml

# 5. Check status
kubectl get blueprintdefinitions
kubectl get blueprints
kubectl get colonyprocesses
```

## Monitoring

Watch process status:

```bash
kubectl get colonyprocess -w
```

Get detailed status:

```bash
kubectl describe colonyprocess <name>
```

View process output:

```bash
kubectl get colonyprocess <name> -o jsonpath='{.status.output}'
```
