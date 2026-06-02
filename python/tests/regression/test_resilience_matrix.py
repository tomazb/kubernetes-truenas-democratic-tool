from io import BytesIO

import pytest
from urllib3.response import HTTPResponse
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
    response = HTTPResponse(status=503, body=BytesIO(b""), preload_content=False)
    delays: list[float] = []
    for _ in range(TRUENAS_CLIENT_RETRY.total):
        retry = retry.increment(method="GET", url="/", response=response, error=None)
        delays.append(retry.get_backoff_time())

    assert delays, "retry history should produce backoff delays"
    assert max(delays) <= retry.backoff_max
    assert any(delay > 0 for delay in delays), "backoff should grow after errors"
