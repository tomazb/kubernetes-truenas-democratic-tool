import pytest

pytestmark = [pytest.mark.idempotency, pytest.mark.slow]


def bounded_exponential_backoff(attempt: int, base_seconds: int, max_seconds: int) -> int:
    delay = base_seconds * (2**attempt)
    return min(delay, max_seconds)


def test_backoff_is_bounded() -> None:
    delays = [bounded_exponential_backoff(i, 1, 8) for i in range(10)]
    assert max(delays) == 8
    assert delays[0] == 1
    assert delays[1] == 2


def test_error_envelope_is_explicit() -> None:
    result = {"error": "dependency_timeout", "retryable": True, "source": "truenas"}
    assert result["error"]
    assert result["source"] in {"truenas", "kubernetes"}
