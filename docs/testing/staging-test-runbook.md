# Staging Test Runbook (Kubernetes + TrueNAS)

This runbook defines required preconditions and execution steps for staging-only tests.

## Preconditions

- Kubernetes context points to non-production staging cluster.
- Democratic CSI components are healthy in target namespace.
- TrueNAS staging endpoint is reachable from runner.
- Dedicated disposable dataset/pool prefix exists for test assets.
- Secrets are injected via environment variables (never committed).

## Required Environment Variables

- `TEST_STAGING=true`
- `STAGING_KUBECONFIG` (path to kubeconfig)
- `STAGING_NAMESPACE` (test namespace, or namespace prefix for per-run creation)
- `STAGING_TRUENAS_URL`
- `STAGING_TRUENAS_USERNAME`
- `STAGING_TRUENAS_PASSWORD`

Optional:

- `STAGING_RESOURCE_PREFIX` (default: `ktdt`)
- `STAGING_CLEANUP_STRICT=true|false` (default: `true`)
- `STAGING_TIMEOUT_SECONDS` (default: `900`)

## Preflight Checklist

1. Verify kube context and namespace access:
   - `kubectl --kubeconfig "$STAGING_KUBECONFIG" get ns`
2. Verify democratic-csi health:
   - `kubectl --kubeconfig "$STAGING_KUBECONFIG" -n "$STAGING_NAMESPACE" get pods`
3. Verify TrueNAS connectivity with service credentials.
4. Confirm test dataset namespace is disposable.

## Execution

- Run staging lane:
  - `make test-staging`
- Run full matrix:
  - `make test-matrix`

## Isolation Rules

- Every run uses a unique resource prefix (`<prefix>-<timestamp>`).
- Test-created namespaces and resources must be cleaned up at end of run.
- If strict cleanup is enabled and leftovers exist, fail the run.

## Failure Handling

- Collect and archive:
  - pytest junit XML
  - API response payload samples for failing checks
  - `kubectl get events -A` snapshot
  - relevant TrueNAS API error payloads (with secrets redacted)
- Open a defect with:
  - failing command
  - environment metadata (cluster name, namespace, image SHAs)
  - first failing assertion and artifact links

## Security Notes

- Never print full secrets in logs.
- Prefer short-lived credentials for staging runs.
- Rotate staging credentials on schedule or after suspicious failure events.
