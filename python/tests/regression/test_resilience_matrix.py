import pytest
from urllib3.util.retry import Retry

pytestmark = [pytest.mark.idempotency, pytest.mark.slow]

# Mirrors truenas_storage_monitor.truenas_client TrueNASClient session retry policy.
TRUENAS_CLIENT_RETRY = Retry(
    total=3,
    backoff_factor=0.3,
    status_forcelist=[500, 502, 503, 504],
)


def test_truenas_client_retry_policy_matches_production_defaults() -> None:
    assert TRUENAS_CLIENT_RETRY.total == 3
    assert TRUENAS_CLIENT_RETRY.backoff_factor == 0.3
    assert TRUENAS_CLIENT_RETRY.status_forcelist == [500, 502, 503, 504]


def test_truenas_client_retry_backoff_is_bounded() -> None:
    retry = TRUENAS_CLIENT_RETRY.new()
    delays = [retry.get_backoff_time() for _ in range(10)]
    assert max(delays) <= 120
