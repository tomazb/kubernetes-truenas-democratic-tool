"""In-process TTL cache for expensive inventory list operations."""

from __future__ import annotations

import threading
import time
from dataclasses import dataclass
from datetime import timedelta
from typing import Any, Callable, Dict, Optional, TypeVar

T = TypeVar("T")


@dataclass
class _CacheEntry:
    value: Any
    expires_at: float
    inserted_at: float


class InventoryCache:
    """Thread-safe TTL cache with optional max size."""

    def __init__(
        self,
        ttl: timedelta,
        max_size: int = 1000,
        enabled: bool = True,
        now: Optional[Callable[[], float]] = None,
    ) -> None:
        self._ttl = ttl.total_seconds()
        self._max_size = max(1, max_size)
        self._enabled = enabled
        self._now = now or time.time
        self._lock = threading.Lock()
        self._entries: Dict[str, _CacheEntry] = {}
        self.hits = 0
        self.misses = 0

    @property
    def enabled(self) -> bool:
        return self._enabled

    def get_or_load(self, operation: str, key: str, loader: Callable[[], T]) -> T:
        """Return cached value or load via loader when missing or expired."""
        if not self._enabled:
            return loader()

        now = self._now()
        with self._lock:
            entry = self._entries.get(key)
            if entry is not None and now <= entry.expires_at:
                self.hits += 1
                return entry.value  # type: ignore[return-value]

        loaded = loader()

        with self._lock:
            self.misses += 1
            self._entries[key] = _CacheEntry(
                value=loaded,
                expires_at=now + self._ttl,
                inserted_at=now,
            )
            if len(self._entries) > self._max_size:
                oldest_key = min(self._entries, key=lambda k: self._entries[k].inserted_at)
                del self._entries[oldest_key]

        return loaded


def namespace_key(prefix: str, namespace: Optional[str]) -> str:
    """Build a cache key with optional namespace scope."""
    if not namespace:
        return f"{prefix}:*"
    return f"{prefix}:{namespace}"
