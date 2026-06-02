import pytest
from truenas_storage_monitor.config import Config

pytestmark = [pytest.mark.security, pytest.mark.slow]


def test_secret_redaction_masking() -> None:
    pytest.skip("No shared redaction helper exists yet; add coverage when implemented.")


def test_tls_default_mode_is_secure(tmp_path) -> None:
    config_file = tmp_path / "config.yaml"
    config_file.write_text(
        "openshift:\n"
        "  namespace: default\n"
        "monitoring:\n"
        "  orphan_threshold: 24h\n"
        "truenas:\n"
        "  url: https://truenas.example.local\n"
        "  username: admin\n"
        "  password: x\n",
        encoding="utf-8",
    )
    cfg = Config(str(config_file))

    truenas_cfg = cfg.truenas_config()
    assert truenas_cfg.verify_ssl is True
