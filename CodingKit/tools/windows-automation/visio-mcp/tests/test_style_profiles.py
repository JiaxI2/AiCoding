import json
from pathlib import Path

import pytest

from visio_mcp.errors import ValidationError
from visio_mcp.layout import plan_diagram
from visio_mcp.validation import validate_diagram


def example():
    return json.loads(
        Path("examples/generic-dual-loop-actuator-control.json").read_text(
            encoding="utf-8"
        )
    )


def test_default_profile_preserves_the_original_compact_visual_baseline():
    plan = plan_diagram(example())
    nodes = {node.id: node for node in plan.nodes}
    edges = {edge.id: edge for edge in plan.edges}

    assert nodes["reference"].font_family == "SimSun"
    assert nodes["reference"].asian_font_family == "SimSun"
    assert nodes["reference"].font_size_pt == 10
    assert nodes["reference"].line_weight_pt == 0.75
    assert nodes["reference"].corner_radius_in == 0.12
    assert nodes["reference"].text_block_width_ratio == 0.8
    assert nodes["sum_position"].font_family == "SimSun"
    assert edges["main_01"].font_size_pt == 10
    assert edges["main_01"].line_weight_pt == 0.75


def test_compact_json_profile_changes_only_the_high_impact_defaults(
    tmp_path,
    monkeypatch,
):
    registry = json.loads(
        Path("styles/style-profiles.json").read_text(encoding="utf-8")
    )
    profile = registry["profiles"]["engineering-standard"]
    profile["font"]["ordinaryFamily"] = "Microsoft YaHei"
    profile["text"]["fontSizePt"] = 11
    profile["text"]["safeRatio"] = 0.75
    profile["line"]["weightPt"] = 0.9
    profile["line"]["cornerRadiusIn"] = 0.04
    target = tmp_path / "style-profiles.json"
    target.write_text(
        json.dumps(registry, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    monkeypatch.setenv("VISIO_MCP_STYLE_PROFILES", str(target))

    plan = plan_diagram(example())
    assert plan.nodes[0].font_family == "Microsoft YaHei"
    assert plan.nodes[0].font_size_pt == 11
    assert plan.nodes[0].text_block_width_ratio == 0.75
    assert plan.nodes[0].line_weight_pt == 0.9
    assert plan.nodes[0].corner_radius_in == 0.04
    assert plan.edges[0].line_weight_pt == 0.9


def test_document_shorthands_override_font_and_line_without_a_new_profile():
    diagram = example()
    diagram["document"]["typography"] = {
        "fontSizePt": 11,
        "mathFontSizePt": 12,
    }
    diagram["document"]["appearance"] = {
        "lineWeightPt": 0.8,
        "nodeCornerRadiusIn": 0.03,
    }
    plan = plan_diagram(diagram)
    nodes = {node.id: node for node in plan.nodes}

    assert nodes["reference"].font_size_pt == 11
    assert nodes["sum_position"].font_size_pt == 11
    assert nodes["zero_reference"].font_size_pt == 12
    assert nodes["reference"].line_weight_pt == 0.8
    assert nodes["reference"].corner_radius_in == 0.03
    assert plan.edges[0].line_weight_pt == 0.8


def test_semantic_style_and_explicit_overrides_are_resolved():
    diagram = example()
    diagram["nodes"][0].update(
        {
            "style": "feedback",
            "fontSizePt": 12,
            "lineWeightPt": 1.2,
            "cornerRadiusIn": 0,
        }
    )
    diagram["edges"][0]["style"] = "warning"
    plan = plan_diagram(diagram)

    assert plan.nodes[0].fill_color == "#D9EAF7"
    assert plan.nodes[0].line_color == "#1F4E79"
    assert plan.nodes[0].line_weight_pt == 1.2
    assert plan.nodes[0].corner_radius_in == 0
    assert plan.edges[0].line_color == "#C00000"


def test_unknown_style_profile_is_rejected():
    diagram = example()
    diagram["document"]["styleProfile"] = "missing"
    with pytest.raises(ValidationError):
        validate_diagram(diagram)
