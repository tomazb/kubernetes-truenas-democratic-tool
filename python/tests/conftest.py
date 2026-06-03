import os
from pathlib import Path

import pytest


def pytest_collection_modifyitems(config: pytest.Config, items: list[pytest.Item]) -> None:
    for item in items:
        path = Path(str(item.fspath))
        if "tests/unit/" in path.as_posix() and "unit" not in item.keywords:
            item.add_marker(pytest.mark.unit)


def pytest_runtest_setup(item: pytest.Item) -> None:
    path = Path(str(item.fspath))
    if "tests/staging/" in path.as_posix():
        if os.getenv("TEST_STAGING", "").lower() != "true":
            pytest.skip("staging tests require TEST_STAGING=true")
