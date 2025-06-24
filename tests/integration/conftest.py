"""Configuration for integration tests."""

import pytest


def pytest_configure(config):
    """Configure pytest markers."""
    config.addinivalue_line(
        "markers", "integration: mark test as an integration test"
    )
    config.addinivalue_line(
        "markers", "e2e: mark test as an end-to-end test"
    )


def pytest_collection_modifyitems(config, items):
    """Modify test collection to add skip markers."""
    # Add skip marker for integration tests if --integration not specified
    if not config.getoption("--integration", default=False):
        skip_integration = pytest.mark.skip(reason="integration tests not enabled (use --integration)")
        for item in items:
            if "integration" in item.keywords:
                item.add_marker(skip_integration)


def pytest_addoption(parser):
    """Add command line options."""
    parser.addoption(
        "--integration",
        action="store_true",
        default=False,
        help="run integration tests (requires K8s/TrueNAS connectivity)"
    )
    parser.addoption(
        "--k8s-config",
        action="store",
        default=None,
        help="path to Kubernetes config file"
    )
    parser.addoption(
        "--truenas-url",
        action="store",
        default=None,
        help="TrueNAS URL for integration tests"
    )