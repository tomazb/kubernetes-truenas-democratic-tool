# Operator-first onboarding (low risk)

This guide is the safest path to evaluate the project in a cluster.

It is intentionally conservative:

- Use a non-production cluster first.
- Keep API exposure internal-only.
- Use implemented routes only.
- Do not perform destructive cleanup based only on first-pass findings.

For product maturity context, see [PRD baseline](../PRD.md#2-current-baseline-shipped-now) and [API endpoint maturity](../api-endpoints.md).

## Safety boundaries

- Treat this tool as **read-oriented monitoring** for onboarding.
- Keep both services as `ClusterIP` only during evaluation.
- Keep TLS verification enabled for TrueNAS (`truenas.insecure: false`).
- Use a least-privilege TrueNAS service account where possible.
- Avoid public ingress/route/load balancer exposure for the API until your own auth and network controls are in place.

## Prerequisites

- Non-production Kubernetes/OpenShift cluster with democratic-csi.
- Access to create resources in `truenas-monitor` namespace.
- Access to create cluster-scoped RBAC objects (`ClusterRole`, `ClusterRoleBinding`).
- Reachable TrueNAS endpoint and credentials for a scoped service account.
- Container images built/pushed and pinned to immutable tags or digests.

## Preflight checklist (before apply)

1. Confirm context points to intended non-production cluster:

```bash
kubectl config current-context
kubectl get ns
```

2. Validate manifests server-side:

```bash
kubectl apply --dry-run=server -f deploy/kubernetes/
kubectl diff -f deploy/kubernetes/
```

3. Validate expected RBAC verbs for the service account:

```bash
kubectl auth can-i --as=system:serviceaccount:truenas-monitor:truenas-monitor list persistentvolumes
kubectl auth can-i --as=system:serviceaccount:truenas-monitor:truenas-monitor list volumesnapshots.snapshot.storage.k8s.io
kubectl auth can-i --as=system:serviceaccount:truenas-monitor:truenas-monitor list storageclasses.storage.k8s.io
```

4. Edit secret placeholders and verify no defaults remain:

- Update `deploy/kubernetes/secret.yaml`:
  - `TRUENAS_URL`
  - `TRUENAS_USERNAME`
  - `TRUENAS_PASSWORD`
  - Optional `SLACK_WEBHOOK`
- Never apply with default credentials such as `TRUENAS_PASSWORD: "changeme"` or `TRUENAS_USERNAME: "admin"`.
- Generate unique credentials and inject them via secure secret-management workflows.
- Treat `SLACK_WEBHOOK` as sensitive; set it only when needed and never commit real webhook values.

5. Review config and deployment defaults:

- `deploy/kubernetes/configmap.yaml`:
  - Verify `kubernetes.namespace` matches your democratic-csi namespace.
  - Keep `truenas.insecure` disabled (commented).
- `deploy/kubernetes/monitor-deployment.yaml` and `deploy/kubernetes/api-deployment.yaml`:
  - Replace `:latest` images with pinned tags/digests.
- `deploy/kubernetes/services.yaml`:
  - Keep `type: ClusterIP` during onboarding.

## Deploy sequence

Apply baseline manifests:

```bash
kubectl apply -f deploy/kubernetes/
```

Confirm resources are up:

```bash
kubectl -n truenas-monitor get pods,svc,cm,secret
kubectl -n truenas-monitor get deploy
```

Check logs for startup and connectivity errors:

```bash
kubectl -n truenas-monitor logs deploy/truenas-monitor --tail=200
kubectl -n truenas-monitor logs deploy/truenas-api --tail=200
```

## Verify implemented API routes only

Port-forward API service:

```bash
kubectl -n truenas-monitor port-forward svc/truenas-api 8080:8080
```

In another shell, run smoke checks:

```bash
curl -sS http://127.0.0.1:8080/health
curl -sS http://127.0.0.1:8080/ready
curl -sS http://127.0.0.1:8080/api/v1/orphans
curl -sS http://127.0.0.1:8080/api/v1/orphans/pvs
curl -sS http://127.0.0.1:8080/api/v1/resources/pvs
curl -sS http://127.0.0.1:8080/api/v1/truenas/volumes
curl -sS http://127.0.0.1:8080/api/v1/validate
```

Expected onboarding behavior:

- Implemented routes return normal JSON payloads.
- Non-implemented routes return HTTP 501 with `{"error":"not_implemented", ...}`.

## Triage and safe interpretation

- Treat orphan findings as investigation leads, not immediate delete actions.
- Validate each candidate against workload ownership and recent lifecycle events.
- Use `age_threshold` query tuning to reduce noise during first runs.
- Record findings and decision rationale before manual cleanup.

## Stop conditions

Stop and fix configuration if any of these occur:

- `/ready` fails repeatedly after pods become ready.
- Startup logs show TrueNAS auth or TLS verification failures.
- API responses are mostly errors/timeouts instead of data/501 contracts.
- Cluster policy/security review flags cluster-scope RBAC as unacceptable.

## Rollback

If onboarding needs to be rolled back:

```bash
kubectl delete -f deploy/kubernetes/
```

Then verify namespace and cluster objects are removed or intentionally retained:

```bash
kubectl get clusterrole truenas-monitor
kubectl get clusterrolebinding truenas-monitor
kubectl get ns truenas-monitor
```

## Next step

After operator onboarding is stable, use the local evaluation path in [local-evaluator.md](local-evaluator.md) for iterative diagnostics and development.
