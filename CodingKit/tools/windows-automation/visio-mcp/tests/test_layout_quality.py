import json
from pathlib import Path

from visio_mcp.layout import plan_diagram
from visio_mcp.quality import structural_quality
from visio_mcp.repair import auto_repair


def data():
    return json.loads(Path("examples/visio-mcp-architecture.json").read_text(encoding="utf-8"))


def engineering_data():
    return json.loads(
        Path("examples/generic-dual-loop-actuator-control.json").read_text(encoding="utf-8")
    )


def container_data():
    return {
        "schemaVersion": 1,
        "document": {
            "id": "container-regression",
            "title": "Container regression",
            "page": {"width": 10, "height": 6, "margin": 0.5},
            "layout": {
                "engine": "manual",
                "direction": "LR",
                "uniformNodeSize": False,
            },
        },
        "nodes": [
            {
                "id": "container",
                "text": "Framework",
                "type": "group",
                "x": 5,
                "y": 3,
                "width": 3.6,
                "height": 1.38,
                "layer": 0,
                "order": 0,
                "sizeClass": "container",
            },
            {
                "id": "first",
                "text": "First",
                "type": "process",
                "x": 4,
                "y": 3,
                "width": 1,
                "height": 0.78,
                "layer": 1,
                "order": 0,
                "sizeClass": "child",
            },
            {
                "id": "second",
                "text": "Second",
                "type": "process",
                "x": 6,
                "y": 3,
                "width": 1,
                "height": 0.78,
                "layer": 2,
                "order": 0,
                "sizeClass": "child",
            },
        ],
        "edges": [],
        "groups": [{"id": "container", "members": ["first", "second"]}],
    }


def test_layout_deterministic():
    assert plan_diagram(data()).to_dict() == plan_diagram(data()).to_dict()


def test_layout_keeps_uniform_sizes_and_alignment():
    plan = plan_diagram(data())
    assert len({round(node.width, 6) for node in plan.nodes}) == 1
    assert len({round(node.height, 6) for node in plan.nodes}) == 1

    primary_row = [node.y for node in plan.nodes if node.order == 0]
    assert max(primary_row) - min(primary_row) < 0.01

    layer_four = [node.x for node in plan.nodes if node.layer == 4]
    assert max(layer_four) - min(layer_four) < 0.01

    report = structural_quality(plan)
    codes = {finding["code"] for finding in report["findings"]}
    assert "INCONSISTENT_NODE_SIZE" not in codes
    assert "LAYER_MISALIGNED" not in codes
    assert "ORDER_MISALIGNED" not in codes
    assert "INCONSISTENT_LAYER_SPACING" not in codes


def test_repair_restores_uniform_size_and_alignment():
    plan = plan_diagram(data())
    plan.nodes[0].width += 0.4
    plan.nodes[1].y += 0.4
    plan.nodes[-1].x += 0.4

    repaired, report = auto_repair(plan)
    codes = {finding["code"] for finding in report["after"]["findings"]}
    assert "INCONSISTENT_NODE_SIZE" not in codes
    assert "LAYER_MISALIGNED" not in codes
    assert "ORDER_MISALIGNED" not in codes
    assert len({round(node.width, 6) for node in repaired.nodes}) == 1


def test_repair_never_reduces_score():
    plan = plan_diagram(data())
    plan.nodes[1].x = plan.nodes[0].x
    plan.nodes[1].y = plan.nodes[0].y
    repaired, report = auto_repair(plan)
    assert report["after"]["score"] >= report["before"]["score"]
    assert structural_quality(repaired)["score"] == report["after"]["score"]


def test_engineering_ports_are_aligned_and_labels_avoid_geometry():
    plan = plan_diagram(engineering_data())
    report = structural_quality(plan)
    assert report["score"] == 100
    assert report["metrics"]["textLineOverlapCount"] == 0
    assert report["metrics"]["textShapeOverlapCount"] == 0
    assert report["metrics"]["connectorCrossingCount"] == 0

    by_id = {edge.id: edge for edge in plan.edges}
    assert by_id["phase_a"].source_port_position == 0.75
    assert by_id["phase_a"].target_port_position == 0.75
    assert by_id["phase_b"].route_points[0][1] == by_id["phase_b"].route_points[-1][1]
    assert all(
        edge.label_anchor_resolved
        for edge in plan.edges
        if edge.label
    )
    assert max(
        abs(edge.label_actual_position - edge.label_position)
        for edge in plan.edges
        if edge.label
    ) <= 0.2
    assert by_id["dq_d"].label_width < 0.4


def test_text_line_overlap_is_blocking():
    plan = plan_diagram(data())
    edge = plan.edges[0]
    edge.label_x = (edge.route_points[0][0] + edge.route_points[-1][0]) / 2
    edge.label_y = edge.route_points[0][1]
    report = structural_quality(plan)
    assert "TEXT_LINE_OVERLAP" in {finding["code"] for finding in report["findings"]}


def test_direct_axis_rejects_inefficient_port_choice():
    diagram = engineering_data()
    edge = next(item for item in diagram["edges"] if item["id"] == "vel_02")
    edge["sourcePort"] = "right"
    report = structural_quality(plan_diagram(diagram))
    assert "INEFFICIENT_CONNECTOR_PORT" in {
        finding["code"] for finding in report["findings"]
    }


def test_same_row_ports_align_by_absolute_coordinate():
    diagram = engineering_data()
    edge = next(item for item in diagram["edges"] if item["id"] == "dq_d")
    edge["targetPortPosition"] = 0.6
    report = structural_quality(plan_diagram(diagram))
    assert "PORT_LANE_MISALIGNED" in {finding["code"] for finding in report["findings"]}


def test_note_cannot_be_a_signal_source():
    diagram = engineering_data()
    node = next(item for item in diagram["nodes"] if item["id"] == "zero_reference")
    node["type"] = "note"
    report = structural_quality(plan_diagram(diagram))
    assert "AMBIGUOUS_SIGNAL_SOURCE" in {
        finding["code"] for finding in report["findings"]
    }


def test_overlapping_connectors_require_a_junction():
    diagram = data()
    first = next(item for item in diagram["edges"] if item["id"] == "e5")
    second = next(item for item in diagram["edges"] if item["id"] == "e6")
    first["sourcePortPosition"] = first["targetPortPosition"] = 0.5
    second["sourcePortPosition"] = second["targetPortPosition"] = 0.5
    report = structural_quality(plan_diagram(diagram))
    assert "CONNECTOR_CROSSING" in {finding["code"] for finding in report["findings"]}


def test_short_arrow_clearance_is_blocking():
    diagram = engineering_data()
    node = next(item for item in diagram["nodes"] if item["id"] == "zero_reference")
    target = next(item for item in diagram["nodes"] if item["id"] == "current_controller")
    target_left = target["x"] - target["width"] / 2
    node["x"] = target_left - node["width"] / 2 - 0.05
    report = structural_quality(plan_diagram(diagram))
    codes = {finding["code"] for finding in report["findings"]}
    assert "ARROW_TERMINAL_CLEARANCE_LOW" in codes
    assert "ARROWHEAD_OVERLAPS_NODE" in codes


def test_caption_anchors_to_shape_side_without_drift():
    diagram = {
        "schemaVersion": 1,
        "document": {
            "id": "caption-regression",
            "title": "Caption regression",
            "page": {"width": 10, "height": 6, "margin": 0.5},
            "layout": {
                "engine": "manual",
                "direction": "LR",
                "uniformNodeSize": False,
            },
        },
        "nodes": [
            {
                "id": "horizontal",
                "text": "H",
                "caption": "Horizontal",
                "captionSide": "top",
                "x": 3,
                "y": 3,
                "width": 1.2,
                "height": 0.6,
                "data": {
                    "sizeEvidence": {
                        "requiredWidth": 0.9,
                        "requiredHeight": 0.45,
                    }
                },
            },
            {
                "id": "vertical",
                "text": "V",
                "caption": "Vertical",
                "x": 7,
                "y": 3,
                "width": 0.6,
                "height": 1.2,
                "data": {
                    "sizeEvidence": {
                        "requiredWidth": 0.45,
                        "requiredHeight": 0.9,
                    }
                },
            },
        ],
        "edges": [],
    }
    plan = plan_diagram(diagram)
    by_id = {node.id: node for node in plan.nodes}
    assert by_id["horizontal"].caption_resolved_side == "top"
    assert by_id["horizontal"].caption_x == by_id["horizontal"].x
    assert by_id["horizontal"].caption_width > 0.9
    assert by_id["vertical"].caption_resolved_side == "right"
    assert by_id["vertical"].caption_y == by_id["vertical"].y
    report = structural_quality(plan)
    assert report["metrics"]["captionAnchorUnresolvedCount"] == 0
    assert report["metrics"]["captionAnchorMisalignedCount"] == 0


def test_nonstandard_text_block_ratio_requires_reason():
    diagram = engineering_data()
    node = next(item for item in diagram["nodes"] if item["id"] == "reference")
    node["textBlockWidthRatio"] = 0.9
    report = structural_quality(plan_diagram(diagram))
    assert "NODE_TEXT_BLOCK_UTILIZATION_NONSTANDARD" in {
        finding["code"]
        for finding in report["findings"]
    }


def test_same_axis_peer_gaps_use_absolute_boundaries():
    plan = plan_diagram(engineering_data())
    report = structural_quality(plan)
    assert report["metrics"]["sameAxisPeerGapGroupCount"] >= 1
    assert report["metrics"]["sameAxisPeerGapInconsistentCount"] == 0
    assert report["metrics"]["maxSameAxisPeerGapSpan"] == 0

    diagram = engineering_data()
    transform = next(item for item in diagram["nodes"] if item["id"] == "transform")
    transform["x"] += 0.12
    report = structural_quality(plan_diagram(diagram))
    assert "SAME_AXIS_GAP_INCONSISTENT" in {
        finding["code"]
        for finding in report["findings"]
    }


def test_target_endpoint_cannot_enter_through_node_interior():
    diagram = engineering_data()
    edge = next(item for item in diagram["edges"] if item["id"] == "zero_d")
    edge["targetPort"] = "right"
    report = structural_quality(plan_diagram(diagram))
    assert "TARGET_ENDPOINT_INTRUSION" in {
        finding["code"] for finding in report["findings"]
    }


def test_multiport_reason_does_not_bypass_oversized_height():
    diagram = engineering_data()
    node = next(item for item in diagram["nodes"] if item["id"] == "transform")
    node["height"] = 2.0
    node["data"] = {"sizeReason": "multiport"}
    report = structural_quality(plan_diagram(diagram))
    assert "OVERSIZED_NODE_WITHOUT_REASON" in {
        finding["code"] for finding in report["findings"]
    }


def test_same_axis_height_is_equal_when_content_allows_it():
    diagram = engineering_data()
    node = next(item for item in diagram["nodes"] if item["id"] == "acc_gain")
    node["height"] = 0.62
    report = structural_quality(plan_diagram(diagram))
    assert "AXIS_SIZE_INCONSISTENT" in {
        finding["code"] for finding in report["findings"]
    }


def test_compact_layout_rejects_excessive_same_axis_gap():
    diagram = engineering_data()
    diagram["document"]["layout"]["nodeGap"] = 0.5
    report = structural_quality(plan_diagram(diagram))
    assert "LAYOUT_TOO_DISPERSED" in {
        finding["code"] for finding in report["findings"]
    }


def test_container_uses_member_envelope_and_bounded_padding():
    report = structural_quality(plan_diagram(container_data()))
    assert report["score"] == 100
    diagram = container_data()
    diagram["nodes"][0]["width"] = 5
    report = structural_quality(plan_diagram(diagram))
    assert "CONTAINER_EXCESS_SLACK" in {
        finding["code"] for finding in report["findings"]
    }
