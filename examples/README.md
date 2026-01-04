# Kolony Examples

This directory contains example YAML files for using Kolony CRDs.

## Prerequisites

1. Kolony operator installed in your cluster
2. A `colonyos-credentials` secret in your target namespace
3. A `container-executor` registered with ColonyOS

## Files

| File | Description |
|------|-------------|
| `credentials-secret.yaml` | Template for ColonyOS credentials secret |
| `colonyprocess-simple.yaml` | Simple hello world container |
| `colonyprocess-docker-executor.yaml` | Container with echo, date, hostname |

## Quick Start

```bash
# 1. Create credentials secret (edit with your values first!)
kubectl apply -f credentials-secret.yaml

# 2. Submit a container process (use 'create' with generateName)
kubectl create -f colonyprocess-simple.yaml

# 3. Watch until completion
kubectl get colonyprocess -w
```

The process transitions: Pending -> Waiting -> Running -> Success

## Check Status

```bash
# Get process state
kubectl get colonyprocess <name> -o jsonpath='{.status.state}'

# Detailed status
kubectl describe colonyprocess <name>

# Full YAML
kubectl get colonyprocess <name> -o yaml
```

Note: The `output` field contains explicit return values from the executor, not stdout. Container stdout/stderr is available in the executor's logs.
