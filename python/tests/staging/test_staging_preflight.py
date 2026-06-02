import os

import pytest

pytestmark = [pytest.mark.integration, pytest.mark.e2e]


REQUIRED_ENV_VARS = [
    "STAGING_KUBECONFIG",
    "STAGING_NAMESPACE",
    "STAGING_TRUENAS_URL",
    "STAGING_TRUENAS_USERNAME",
    "STAGING_TRUENAS_PASSWORD",
]


def test_staging_environment_contract() -> None:
    missing = [key for key in REQUIRED_ENV_VARS if not os.getenv(key)]
    assert not missing, f"missing required staging env vars: {', '.join(missing)}"
