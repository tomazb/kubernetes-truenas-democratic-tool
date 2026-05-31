"""Unit tests for inventory TTL cache."""

from datetime import timedelta

from truenas_storage_monitor.inventory_cache import InventoryCache, namespace_key


def test_cache_hit_and_miss():
    now = [1000.0]

    cache = InventoryCache(
        ttl=timedelta(minutes=1),
        max_size=10,
        now=lambda: now[0],
    )

    loads = {"count": 0}

    def loader():
        loads["count"] += 1
        return "value"

    assert cache.get_or_load("k8s_pvs", "k8s_pvs", loader) == "value"
    assert cache.get_or_load("k8s_pvs", "k8s_pvs", loader) == "value"
    assert loads["count"] == 1
    assert cache.hits == 1
    assert cache.misses == 1


def test_cache_expiry():
    now = [1000.0]
    cache = InventoryCache(
        ttl=timedelta(minutes=1),
        now=lambda: now[0],
    )

    loads = {"count": 0}

    def loader():
        loads["count"] += 1
        return loads["count"]

    assert cache.get_or_load("k8s_pvs", "k8s_pvs", loader) == 1
    now[0] += 120
    assert cache.get_or_load("k8s_pvs", "k8s_pvs", loader) == 2
    assert loads["count"] == 2


def test_cache_disabled_passthrough():
    cache = InventoryCache(ttl=timedelta(minutes=1), enabled=False)
    loads = {"count": 0}

    def loader():
        loads["count"] += 1
        return "direct"

    cache.get_or_load("k8s_pvs", "k8s_pvs", loader)
    cache.get_or_load("k8s_pvs", "k8s_pvs", loader)
    assert loads["count"] == 2


def test_namespace_key():
    assert namespace_key("k8s_pvcs", None) == "k8s_pvcs:*"
    assert namespace_key("k8s_pvcs", "apps") == "k8s_pvcs:apps"
