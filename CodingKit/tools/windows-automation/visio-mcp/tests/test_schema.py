import json
from pathlib import Path

import pytest

from visio_mcp.errors import ValidationError
from visio_mcp.validation import validate_diagram


def example():
    return json.loads(Path("examples/visio-mcp-architecture.json").read_text(encoding="utf-8"))


def test_example_valid():
    result = validate_diagram(example())
    assert result["valid"] and result["nodeCount"] == 7


def test_duplicate_node_rejected():
    diagram = example()
    diagram["nodes"].append(dict(diagram["nodes"][0]))
    with pytest.raises(ValidationError):
        validate_diagram(diagram)


def test_broken_edge_rejected():
    diagram = example()
    diagram["edges"][0]["to"] = "missing"
    with pytest.raises(ValidationError):
        validate_diagram(diagram)


def test_connector_ports_and_positions_validate():
    diagram = example()
    diagram["edges"][0].update(
        {
            "sourcePort": "right",
            "targetPort": "left",
            "sourcePortPosition": 0.25,
            "targetPortPosition": 0.75,
            "routing": "orthogonal",
            "labelSide": "above",
            "labelOffset": 0.3,
            "labelPosition": 0.5,
            "fontFamily": "Microsoft YaHei",
        }
    )
    assert validate_diagram(diagram)["valid"] is True


def test_caption_typography_and_text_safe_area_validate():
    diagram = example()
    diagram["document"]["typography"] = {
        "fontFamily": "Microsoft YaHei",
        "nodeTextBlockWidthRatio": 0.8,
        "nodeTextBlockHeightRatio": 0.8,
    }
    diagram["nodes"][0].update(
        {
            "caption": "Owner",
            "captionSide": "top",
            "captionPosition": 0.5,
            "captionOffset": 0.1,
            "fontFamily": "Microsoft YaHei",
            "textBlockWidthRatio": 0.8,
            "textBlockHeightRatio": 0.8,
        }
    )
    assert validate_diagram(diagram)["valid"] is True


def test_renderer_effective_fields_register_public_geometry():
    effective = json.loads(
        Path("schemas/renderer-effective-fields.json").read_text(encoding="utf-8")
    )
    assert {
        "caption",
        "captionSide",
        "captionPosition",
        "captionOffset",
        "textBlockWidthRatio",
        "textBlockHeightRatio",
        "fontSizePt",
        "fontRole",
        "lineColor",
        "fillColor",
        "lineWeightPt",
        "cornerRadiusIn",
    }.issubset(effective["node"])
    assert {
        "labelPosition",
        "labelSide",
        "labelOffset",
        "fontFamily",
        "fontSizePt",
        "fontRole",
        "lineColor",
        "lineWeightPt",
    }.issubset(effective["edge"])


def test_style_profile_registry_and_schema_are_valid():
    from jsonschema import Draft202012Validator

    schema = json.loads(
        Path("schemas/style-profile.schema.json").read_text(encoding="utf-8")
    )
    registry = json.loads(
        Path("styles/style-profiles.json").read_text(encoding="utf-8")
    )
    assert list(Draft202012Validator(schema).iter_errors(registry)) == []


def test_compact_layout_and_container_members_validate():
    diagram = example()
    diagram["document"]["layout"]["compact"] = True
    diagram["nodes"].append(
        {
            "id": "container",
            "text": "Runtime",
            "type": "group",
            "layer": 6,
        }
    )
    diagram["groups"] = [{"id": "container", "members": ["visio", "outputs"]}]
    assert validate_diagram(diagram)["valid"] is True


@pytest.mark.parametrize(
    ("field", "value"),
    [
        ("sourcePort", "center"),
        ("routing", "curved"),
        ("sourcePortPosition", 1.1),
        ("labelSide", "center"),
        ("labelPosition", 1.1),
    ],
)
def test_invalid_connector_geometry_rejected(field, value):
    diagram = example()
    diagram["edges"][0][field] = value
    with pytest.raises(ValidationError):
        validate_diagram(diagram)
