# Contributing to Kubernetes TrueNAS Democratic Tool

Thank you for your interest in contributing! We welcome contributions from the community and are grateful for any help you can provide.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please read it to understand the expectations for all contributors.

## How to Contribute

### Reporting Issues

Before creating an issue, please:
1. Check existing issues to avoid duplicates
2. Use the issue templates provided
3. Include as much detail as possible

### Suggesting Features

1. Check the [roadmap](docs/PRD.md) to see if it's already planned
2. Open a feature request using the template
3. Explain the use case and benefits

### Contributing Code

#### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/yourusername/kubernetes-truenas-democratic-tool.git
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

We follow Test-Driven Development (TDD):

1. **Write tests first**
   ```bash
   # Create a test file
   touch python/tests/unit/test_your_feature.py
   
   # Write failing test
   # Run test to see it fail
   pytest python/tests/unit/test_your_feature.py -v
   ```

2. **Implement the feature**
   - Write minimal code to make tests pass
   - Follow existing code style
   - Add appropriate logging

3. **Refactor**
   - Improve code quality
   - Ensure tests still pass
   - Check coverage

4. **Document**
   - Add docstrings to all functions
   - Update README if needed
   - Add examples if applicable

#### Code Standards

**Python:**
- Follow PEP 8
- Use type hints
- Maximum line length: 100
- Use Black for formatting

**Go:**
- Follow standard Go conventions
- Use `gofmt` and `golangci-lint`
- Write idiomatic Go code

**General:**
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused
- Follow SOLID principles

#### Testing Requirements

- Minimum 90% code coverage
- All new features must have tests
- Tests must be deterministic
- Mock external dependencies
- Include both positive and negative test cases

#### Commit Messages

Follow conventional commits format:
```
type(scope): subject

body

footer
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tool changes

Example:
```
feat(cli): add json output format for orphans command

- Add --format flag with json option
- Structure output for machine parsing
- Update tests and documentation

Closes #123
```

#### Pull Request Process

1. **Ensure all tests pass**
   ```bash
   make test-all
   ```

2. **Check code quality**
   ```bash
   make lint-all
   make security-scan
   ```

3. **Update documentation**
   - Update README if needed
   - Add/update docstrings
   - Update CHANGELOG

4. **Sign your commits**
   ```bash
   git commit -s -m "Your commit message"
   ```

5. **Create pull request**
   - Use the PR template
   - Reference related issues
   - Provide clear description
   - Include test results

6. **Code review**
   - Address reviewer feedback
   - Keep PR focused and small
   - Be responsive to comments

### Testing

#### Running Tests

```bash
# All tests
make test-all

# Unit tests only
make test-unit

# With coverage
make test-coverage

# Watch mode (Python)
make test-watch
```

#### Writing Tests

```python
# Example unit test
import pytest
from unittest.mock import Mock

def test_orphan_detection():
    """Test that orphaned PVs are correctly identified."""
    # Arrange
    k8s_client = Mock()
    truenas_client = Mock()
    
    # Act
    orphans = detect_orphaned_pvs(k8s_client, truenas_client)
    
    # Assert
    assert len(orphans) == 2
    assert orphans[0].name == "pv-12345"
```

### Documentation

- Use clear, concise language
- Include code examples
- Keep documentation up-to-date
- Use proper markdown formatting

### Security

- Never commit credentials
- Validate all inputs
- Follow security best practices
- Report security issues privately

## Getting Help

- Check the [documentation](docs/)
- Ask in [GitHub Discussions](https://github.com/yourusername/kubernetes-truenas-democratic-tool/discussions)
- Join our community chat (coming soon)

## Recognition

Contributors will be recognized in:
- The CONTRIBUTORS file
- Release notes
- Project documentation

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## Thank You!

Your contributions make this project better for everyone. We appreciate your time and effort!