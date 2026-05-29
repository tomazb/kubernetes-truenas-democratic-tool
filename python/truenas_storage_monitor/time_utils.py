"""Timezone-aware datetime helpers."""

from datetime import datetime, timezone
from typing import Optional


def utc_now() -> datetime:
    """Return the current time as a timezone-aware UTC datetime."""
    return datetime.now(timezone.utc)


def ensure_utc(dt: datetime) -> datetime:
    """Return a timezone-aware UTC datetime.

    Naive datetimes are treated as UTC.
    """
    if dt.tzinfo is None:
        return dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)


def parse_rfc3339(value: str) -> datetime:
    """Parse an RFC3339 / Kubernetes timestamp into aware UTC."""
    if not value:
        raise ValueError("timestamp value must not be empty")

    normalized = value.replace("Z", "+00:00")
    parsed = datetime.fromisoformat(normalized)
    return ensure_utc(parsed)


def resource_age(created: Optional[datetime]) -> Optional[str]:
    """Format age string for a resource creation time."""
    if created is None:
        return None
    return str(utc_now() - ensure_utc(created))
