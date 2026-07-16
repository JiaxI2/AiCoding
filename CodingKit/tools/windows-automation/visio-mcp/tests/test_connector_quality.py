import json
from pathlib import Path

from visio_mcp.layout import plan_diagram
from visio_mcp.quality import connector_quality
from visio_mcp.service import VisioService


def data():
    return json.loads(
        Path("examples/generic-dual-loop-actuator-control.json").read_text(encoding="utf-8")
    )


def test_mock_connector_quality_is_clean(tmp_path, monkeypatch):
    monkeypatch.setenv("VISIO_MCP_ROOT", str(tmp_path))
    monkeypatch.setenv("VISIO_MCP_OUTPUT_ROOTS", "dist")
    service = VisioService(renderer="mock")
    rendered = service.render(data(), tmp_path / "dist" / "engineering.json")
    assert rendered["connectorQuality"]["score"] == 100
    assert rendered["connectorQuality"]["metrics"]["maxEndpointAlignmentError"] == 0
    assert rendered["connectorQuality"]["metrics"]["fullyGluedRatio"] == 1.0
    assert rendered["connectorQuality"]["metrics"]["arrowheadNodeOverlapCount"] == 0
    assert rendered["connectorQuality"]["metrics"]["arrowGeometryUnverifiedCount"] == 0
    assert rendered["connectorQuality"]["metrics"]["nodeTextMisalignedCount"] == 0
    checked = service.quality_check(rendered["sessionId"])
    assert checked["connectors"]["metrics"]["textLineOverlapCount"] == 0
    service.close(rendered["sessionId"])


def test_live_label_on_line_is_rejected(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "connector.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    edge = next(item for item in inspection["edges"] if item["text"])
    edge["labelX"] = (edge["beginX"] + edge["endX"]) / 2
    edge["labelY"] = (edge["beginY"] + edge["endY"]) / 2
    report = connector_quality(plan, inspection)
    assert "TEXT_LINE_OVERLAP" in {finding["code"] for finding in report["findings"]}
    service.renderer.close(rendered["sessionId"])


def test_live_arrow_geometry_is_required(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "connector.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    inspection["edges"][0].pop("endArrow")
    report = connector_quality(plan, inspection)
    assert "ARROW_GEOMETRY_UNVERIFIED" in {
        finding["code"] for finding in report["findings"]
    }
    service.renderer.close(rendered["sessionId"])


def test_live_node_text_must_remain_centered(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "connector.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    inspection["nodes"][0]["horizontalAlign"] = 0
    report = connector_quality(plan, inspection)
    assert "NODE_TEXT_NOT_CENTERED" in {
        finding["code"] for finding in report["findings"]
    }
    service.renderer.close(rendered["sessionId"])


def test_live_font_and_text_safe_area_are_required(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "connector.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    inspection["nodes"][0]["asianFontName"] = "Microsoft YaHei"
    inspection["nodes"][0]["textWidthRatio"] = 0.5
    report = connector_quality(plan, inspection)
    codes = {finding["code"] for finding in report["findings"]}
    assert "NODE_ASIAN_FONT_MISMATCH" in codes
    assert "NODE_TEXT_BLOCK_UTILIZATION_OUT_OF_RANGE" in codes
    service.renderer.close(rendered["sessionId"])


def test_live_font_size_color_and_line_weight_are_required(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "style.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    inspection["nodes"][0]["fontSizePt"] += 1
    inspection["nodes"][0]["textColor"] = "#FF0000"
    inspection["nodes"][0]["lineWeightPt"] += 0.2
    labeled_edge = next(item for item in inspection["edges"] if item["text"])
    labeled_edge["fontSizePt"] += 1
    labeled_edge["lineColor"] = "#FF0000"

    report = connector_quality(plan, inspection)
    codes = {finding["code"] for finding in report["findings"]}
    assert "NODE_FONT_SIZE_MISMATCH" in codes
    assert "NODE_TEXT_COLOR_MISMATCH" in codes
    assert "NODE_LINE_WEIGHT_MISMATCH" in codes
    assert "CONNECTOR_LABEL_FONT_SIZE_MISMATCH" in codes
    assert "CONNECTOR_LINE_COLOR_MISMATCH" in codes
    service.renderer.close(rendered["sessionId"])


def test_mock_caption_is_rendered_with_bound_font(tmp_path):
    diagram = data()
    actuator = next(item for item in diagram["nodes"] if item["id"] == "actuator")
    actuator.update(
        {
            "caption": "Output",
            "captionSide": "top",
            "captionPosition": 0.5,
            "captionOffset": 0.1,
        }
    )
    plan = plan_diagram(diagram)
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "caption.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    report = connector_quality(plan, inspection)
    assert report["metrics"]["captions"] == len(
        [node for node in plan.nodes if node.caption]
    )
    assert report["metrics"]["captionMissingCount"] == 0
    assert report["metrics"]["captionAnchorMisalignedCount"] == 0
    assert report["metrics"]["captionFontMismatchCount"] == 0
    service.renderer.close(rendered["sessionId"])


def test_live_same_axis_peer_gaps_are_absolute(tmp_path):
    plan = plan_diagram(data())
    service = VisioService(renderer="mock")
    rendered = service.renderer.render(plan, tmp_path / "spacing.json")
    inspection = service.renderer.inspect(rendered["sessionId"])
    transform = next(item for item in inspection["nodes"] if item["id"] == "transform")
    transform["x"] += 0.12
    report = connector_quality(plan, inspection)
    assert "SAME_AXIS_GAP_INCONSISTENT" in {
        finding["code"]
        for finding in report["findings"]
    }
    service.renderer.close(rendered["sessionId"])
