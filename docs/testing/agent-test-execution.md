# Autonomous Agent Test Execution Guide

This guide defines deterministic execution order, retry policy, and artifacts for autonomous test runs.

## Execution Order

1. `make ci-precheck`
2. `make test-unit`
3. `make test-integration`
4. `make test-security`
5. `make test-staging` (requires staging env vars)
6. `make test-release-matrix`

Composite command:

- `make test-matrix`

## Stop Conditions

- Stop immediately on lane1 failure (`ci-precheck`, `unit`, `integration`, `security`).
- For staging/release lanes, stop after first failing suite and publish artifacts.
- Do not auto-retry assertion failures.

## Retry Policy

- Allowed retry count: `1` only for infrastructure/transient failures (network timeout, runner interruption).
- No retries for deterministic test failures, schema mismatches, or security threshold failures.

## Parallelization Rules

- Lane1 suites can run in parallel where CI supports it, but final gate must aggregate pass/fail.
- Lane2 and lane3 should run sequentially after lane1 pass to preserve failure triage clarity.

## Artifacts (Required)

- `artifacts/junit/*.xml` (pytest/go junit if generated)
- `artifacts/coverage/*`
- `artifacts/security/*`
- `artifacts/perf/*`
- `artifacts/staging/*`
- `artifacts/summary.json` (high-level run status per lane)

## Automatic Defect Filing Trigger

Open a defect when any of these conditions is met:

- Same test fails in two consecutive matrix runs.
- Security matrix fails with high/critical issue.
- Staging cleanup leak detected.
- Performance budget regression exceeds configured threshold.

Defect payload should include command, failing assertion, commit SHA, lane, and artifact paths.

## Exit Criteria

- All lane1 suites pass.
- Staging suite passes with strict cleanup checks.
- Release matrix passes resilience/security/performance thresholds.
- Summary artifact generated and linked in CI run.
