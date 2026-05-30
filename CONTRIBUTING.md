# Contributing to Kubernetes TrueNAS Democratic Tool

Thank you for your interest in contributing! We welcome contributions from the community.

## Canonical playbook

Repository standards, PR policy, per-PR specs, verification gates, and execution tracking are defined in **[AGENTS.md](AGENTS.md)**. Read it before opening a PR. `CLAUDE.md` is a legacy pointer to the same playbook.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct.

## How to Contribute

### Reporting Issues

Before creating an issue:

1. Check existing issues to avoid duplicates
2. Use the issue templates when available
3. Include as much detail as possible

### Suggesting Features

1. Check the [roadmap](docs/PRD.md) to see if it is already planned
2. Open a feature request using the template
3. Explain the use case and benefits

### Contributing Code

#### Development Setup

1. **Fork and clone the repository**

   ```bash
   git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
   cd kubernetes-truenas-democratic-tool
   ```

2. **Set up development environment**

   ```bash
   make dev-setup
   source venv/bin/activate
   ```

3. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

#### Development Process

We follow Test-Driven Development (TDD) for changed behavior:

1. **Write tests first** for new logic
2. **Implement** minimal code to make tests pass
3. **Refactor** while keeping tests green
4. **Document** user-visible changes in the same PR when feasible

#### Code Standards

**Python:**

- Follow PEP 8
- Use type hints where practical
- Maximum line length: 100
- Use Black for formatting

**Go:**

- Follow standard Go conventions
- Use `gofmt` and `golangci-lint`
- Write idiomatic Go code

**General:**

- Write clear, self-documenting code
- Keep functions focused
- Match existing patterns in the codebase

#### Testing Requirements

- Interim Python coverage gate: **70%** (see backlog BL-20260528-python-coverage-gate; target is 90%)
- All new features must have tests
- Tests must be deterministic
- Mock external dependencies in unit tests

#### Commit Messages

Follow conventional commits format:

```
type(scope): subject

body

footer
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:

```
feat(api): add orphan snapshot route

- Implement GET /api/v1/orphans/snapshots
- Add table-driven handler tests

Closes #123
```

#### Pull Request Process

1. **Ensure tests pass**

   ```bash
   make test-all
   make ci-precheck
   ```

2. **Check code quality**

   ```bash
   make lint-all
   make security-scan
   ```

3. **Add a design spec** under `docs/superpowers/specs/` for implementation PRs (see AGENTS.md)

4. **Sign your commits**

   ```bash
   git commit -s -m "Your commit message"
   ```

5. **Create pull request**
   - Use the PR template and mandatory checklist from AGENTS.md
   - Link the design spec
   - Reference related issues

6. **Code review**
   - Address reviewer feedback
   - Keep PR focused and reviewable

### Testing

#### Running Tests

```bash
make test-all           # Go + Python full suites
make test-unit          # Unit tests only
make go-test-coverage   # Go coverage HTML report
make python-test        # Python tests with coverage (70% gate)
make test-watch         # Python unit tests in watch mode
```

#### Writing Tests

```python
import pytest
from unittest.mock import Mock

def test_orphan_detection():
    """Test that orphaned PVs are correctly identified."""
    k8s_client = Mock()
    truenas_client = Mock()
    orphans = detect_orphaned_pvs(k8s_client, truenas_client)
    assert len(orphans) == 2
```

### Documentation

- Keep docs aligned with **implemented** behavior (see README maturity table)
- Go vs Python config differences: [docs/config-compatibility.md](docs/config-compatibility.md)
- API route status: [docs/api-endpoints.md](docs/api-endpoints.md)

### Security

- Never commit credentials
- Validate all inputs
- Report security issues privately (see SECURITY.md)

## Getting Help

- [Documentation](docs/)
- [GitHub Discussions](https://github.com/tomazb/kubernetes-truenas-democratic-tool/discussions)
- [GitHub Issues](https://github.com/tomazb/kubernetes-truenas-democratic-tool/issues)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## Thank You!

Your contributions make this project better for everyone.
