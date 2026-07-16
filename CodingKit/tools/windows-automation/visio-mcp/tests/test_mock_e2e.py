import json
from pathlib import Path

from visio_mcp.service import VisioService


def test_mock_render_snapshot_inspect(tmp_path, monkeypatch):
    monkeypatch.setenv("VISIO_MCP_ROOT", str(tmp_path))
    monkeypatch.setenv("VISIO_MCP_OUTPUT_ROOTS", "dist")
    service = VisioService(renderer="mock")
    data = json.loads(Path("examples/visio-mcp-architecture.json").read_text(encoding="utf-8"))
    rendered = service.render(data, tmp_path / "dist" / "diagram.json")
    snapshot = service.snapshot(rendered["sessionId"], tmp_path / "dist" / "diagram.png")
    assert Path(snapshot["output"]).exists()
    inspected = service.inspect(rendered["sessionId"])
    assert len(inspected["nodes"]) == len(data["nodes"])
    assert snapshot["imageQuality"]["score"] >= 70
    service.close(rendered["sessionId"])
