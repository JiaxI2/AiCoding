from pathlib import Path

import pytest

from visio_mcp.config import Settings


def test_output_root_enforced(tmp_path):
    settings = Settings(tmp_path, (tmp_path / "dist",))
    with pytest.raises(PermissionError):
        settings.ensure_output_allowed(tmp_path / "outside" / "x.vsdx")
