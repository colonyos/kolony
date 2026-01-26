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
| `echo-blueprint.yaml` | Echo executor deployment blueprint |
| `datalogger-definition.yaml` | DataLogger BlueprintDefinition |
| `datalogger-blueprint.yaml` | DataLogger device blueprint |

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

## DataLogger Example

The DataLogger example demonstrates declarative management of data logging devices using the Blueprint pattern.

```bash
# 1. Register the DataLogger kind
kubectl apply -f datalogger-definition.yaml

# 2. Create a data logger device
kubectl apply -f datalogger-blueprint.yaml

# 3. Watch the status
kubectl get blueprints -w

# 4. Update the application version
kubectl patch blueprint data-logger-1 --type=merge \
  -p '{"spec":{"data":{"appVersion":"3.0"}}}'

# 5. Check sync status
kubectl describe blueprint data-logger-1
```

The reconciler ensures the device's actual state matches the desired state defined in the blueprint.
