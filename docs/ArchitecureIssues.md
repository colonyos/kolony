# Architectural Issue: CRD vs BlueprintDefinition Mapping

## Current State

### ColonyOS Model
- **BlueprintDefinition**: Schema/type definition (e.g., "DataLogger", "ExecutorDeployment")
- **Blueprint**: Instance of a BlueprintDefinition (e.g., "data-logger-1")
- Relationship: 1 BlueprintDefinition → N Blueprints

### Kolony (Kubernetes) Model
- Single CRD: `blueprints.colony.colonyos.io`
- Single CRD: `blueprintdefinitions.colony.colonyos.io`
- The `spec.kind` field inside Blueprint determines the ColonyOS type
- All blueprint instances share the same Kubernetes schema

## The Problem

There's a mismatch between the ColonyOS type system and Kubernetes CRDs:

1. **No type-specific validation in Kubernetes**
   - A DataLogger blueprint and an ExecutorDeployment blueprint have the same K8s schema
   - Validation only happens when syncing to ColonyOS, not at admission time

2. **Poor discoverability**
   - `kubectl get blueprints` shows all types mixed together
   - Cannot do `kubectl get dataloggers` or `kubectl get executordeployments`
   - Users must filter by `spec.kind` manually

3. **No schema introspection**
   - `kubectl explain blueprint.spec.data` shows generic `RawExtension`
   - Users don't know what fields are valid for a specific kind

## Possible Solutions

### Option A: Keep Single CRD (Current)

Keep the current approach with a single Blueprint CRD.

**Pros:**
- Simple implementation
- No dynamic CRD management
- Works today

**Cons:**
- No native K8s validation per type
- Poor UX for `kubectl` users
- Doesn't feel "Kubernetes-native"

### Option B: Dynamic CRD Generation

When a BlueprintDefinition is created in Kubernetes, automatically generate a corresponding CRD.

Example: Creating a `DataLogger` BlueprintDefinition would generate:
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: dataloggers.colony.colonyos.io
spec:
  group: colony.colonyos.io
  names:
    kind: DataLogger
    plural: dataloggers
    singular: datalogger
  scope: Namespaced
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          # Generated from BlueprintDefinition.spec.schema
```

**Pros:**
- Full Kubernetes-native experience
- Type-specific validation at admission
- `kubectl get dataloggers` works
- Schema introspection via `kubectl explain`

**Cons:**
- Complex implementation
- Requires cluster-admin privileges to create CRDs
- Need to handle CRD lifecycle (create, update, delete)
- Similar complexity to Crossplane providers
- Race conditions between CRD creation and CR creation

### Option C: Validating Admission Webhook

Keep single Blueprint CRD but add a validating admission webhook that:
1. Looks up the BlueprintDefinition for the `spec.kind`
2. Validates `spec.data` against the BlueprintDefinition's schema

**Pros:**
- Type-specific validation without dynamic CRDs
- Simpler than Option B
- No cluster-admin privileges needed for CRD creation

**Cons:**
- Still no `kubectl get dataloggers`
- Still no schema introspection
- Webhook adds latency and failure modes

### Option D: Aggregated API Server

Implement a Kubernetes Aggregated API Server that dynamically serves different resources based on BlueprintDefinitions.

**Pros:**
- Most flexible
- Full control over API behavior

**Cons:**
- Most complex to implement
- Significant operational overhead
- Overkill for this use case?

## Questions to Consider

1. How important is the "Kubernetes-native" experience for users?
2. What's the expected number of BlueprintDefinition types? (Few vs. many)
3. Is cluster-admin privilege for CRD creation acceptable?
4. Should Kolony work in environments where CRD creation is restricted?
5. How do similar projects (Crossplane, KubeVela) handle this?

## Related Work

- **Crossplane**: Dynamically generates CRDs from provider schemas
- **KubeVela**: Uses a single Application CRD with component types
- **Argo CD ApplicationSets**: Single CRD with templates

## Decision

TBD - Document decision and rationale here.
