"""Unit tests for timezone helpers."""

from datetime import datetime, timezone

import pytest

from truenas_storage_monitor.time_utils import ensure_utc, parse_rfc3339, resource_age, utc_now


class TestTimeUtils:
    """Test cases for time utility helpers."""

    def test_utc_now_is_aware(self):
        """utc_now returns timezone-aware UTC datetimes."""
        now = utc_now()
        assert now.tzinfo is not None
        assert now.tzinfo == timezone.utc

    def test_ensure_utc_from_naive(self):
        """Naive datetimes are treated as UTC."""
        naive = datetime(2024, 1, 1, 12, 0, 0)
        result = ensure_utc(naive)
        assert result.tzinfo == timezone.utc
        assert result.hour == 12

    def test_ensure_utc_from_aware(self):
        """Aware datetimes are converted to UTC."""
        aware = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
        result = ensure_utc(aware)
        assert result.tzinfo == timezone.utc

    def test_parse_rfc3339_with_z_suffix(self):
        """RFC3339 timestamps with Z suffix parse to aware UTC."""
        parsed = parse_rfc3339("2024-06-01T10:15:30Z")
        assert parsed.tzinfo == timezone.utc
        assert parsed.year == 2024

    def test_parse_rfc3339_with_offset(self):
        """RFC3339 timestamps with numeric offsets parse to UTC."""
        parsed = parse_rfc3339("2024-06-01T10:15:30+00:00")
        assert parsed.tzinfo == timezone.utc

    def test_parse_rfc3339_naive_input(self):
        """Naive RFC3339-like timestamps become aware UTC."""
        parsed = parse_rfc3339("2024-06-01T10:15:30")
        assert parsed.tzinfo == timezone.utc

    def test_parse_rfc3339_empty_raises(self):
        """Empty timestamp strings raise ValueError."""
        with pytest.raises(ValueError):
            parse_rfc3339("")

    def test_resource_age_returns_string(self):
        """resource_age formats a non-empty age string."""
        created = utc_now()
        age = resource_age(created)
        assert age is not None
        assert "0:00:" in age
