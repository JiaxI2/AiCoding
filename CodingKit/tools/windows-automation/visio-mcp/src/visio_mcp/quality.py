from __future__ import annotations

from collections import defaultdict
from dataclasses import asdict, dataclass
import math
from pathlib import Path

from PIL import Image, ImageStat

from .geometry import (
    ARROWHEAD_LENGTH,
    ENDPOINT_TAIL_CLEARANCE,
    arrowhead_bounds,
    caption_center,
    endpoint_segment_state,
    estimate_node_text_size,
    expand_rectangle,
    node_rectangle,
    polyline_length,
    port_point,
    rectangle,
    rectangles_overlap,
    required_port_span,
    segment_crosses_segment,
    segment_intersects_rectangle,
    segments,
    simplify_polyline,
)
from .model import DiagramPlan, NodeBox
from .styling import (
    char_style_code,
    normalize_color,
    resolve_document_style,
    resolve_text_role,
)


@dataclass
class Finding:
    severity: str
    code: str
    message: str
    objects: list[str]
    repairable: bool


def _overlap(first: NodeBox, second: NodeBox) -> float:
    first_x1, first_x2 = first.x - first.width / 2, first.x + first.width / 2
    first_y1, first_y2 = first.y - first.height / 2, first.y + first.height / 2
    second_x1, second_x2 = second.x - second.width / 2, second.x + second.width / 2
    second_y1, second_y2 = second.y - second.height / 2, second.y + second.height / 2
    width = max(0.0, min(first_x2, second_x2) - max(first_x1, second_x1))
    height = max(0.0, min(first_y2, second_y2) - max(first_y1, second_y1))
    return width * height


def _allowed_dimension(required: float) -> float:
    return max(required * 1.45, required + 0.45)


def _normalized_font_name(value: object) -> str:
    normalized = "".join(character for character in str(value or "").lower() if character.isalnum())
    aliases = {
        "微软雅黑": "microsoftyahei",
        "microsoftyahei": "microsoftyahei",
        "microsoftyaheiui": "microsoftyaheiui",
        "宋体": "simsun",
        "simsun": "simsun",
    }
    return aliases.get(normalized, normalized)


def _normalized_color(value: object) -> str:
    try:
        return normalize_color(str(value or ""))
    except ValueError:
        return ""


def _same_axis_peer_gap_groups(
    plan: DiagramPlan,
    measured_nodes: dict[str, object] | None = None,
) -> dict[tuple[str, int], list[tuple[str, float, str, str]]]:
    expected_nodes = {node.id: node for node in plan.nodes}
    measured = measured_nodes or expected_nodes
    direction = plan.document.get("layout", {}).get("direction", "LR")
    grouped_pairs: dict[
        tuple[str, int],
        dict[tuple[str, str], tuple[str, float, str, str]],
    ] = defaultdict(dict)

    def numeric(item: object, name: str) -> float:
        if isinstance(item, dict):
            return float(item[name])
        return float(getattr(item, name))

    for edge in plan.edges:
        source_expected = expected_nodes[edge.source]
        target_expected = expected_nodes[edge.target]
        if (
            not source_expected.size_class
            or source_expected.size_class != target_expected.size_class
            or source_expected.type in ("group", "note", "junction")
            or target_expected.type in ("group", "note", "junction")
        ):
            continue
        axis_id = (
            source_expected.order
            if direction in ("LR", "RL")
            else source_expected.layer
        )
        target_axis_id = (
            target_expected.order
            if direction in ("LR", "RL")
            else target_expected.layer
        )
        if axis_id != target_axis_id:
            continue
        source = measured.get(edge.source)
        target = measured.get(edge.target)
        if source is None or target is None:
            continue
        if direction in ("LR", "RL"):
            if abs(numeric(source, "y") - numeric(target, "y")) > 0.05:
                continue
            gap = abs(numeric(source, "x") - numeric(target, "x")) - (
                numeric(source, "width") + numeric(target, "width")
            ) / 2
        else:
            if abs(numeric(source, "x") - numeric(target, "x")) > 0.05:
                continue
            gap = abs(numeric(source, "y") - numeric(target, "y")) - (
                numeric(source, "height") + numeric(target, "height")
            ) / 2
        pair = tuple(sorted((edge.source, edge.target)))
        grouped_pairs[(source_expected.size_class, axis_id)].setdefault(
            pair,
            (edge.id, gap, edge.source, edge.target),
        )
    return {
        key: list(pairs.values())
        for key, pairs in grouped_pairs.items()
        if len(pairs) >= 2
    }


def _node_required_sizes(plan: DiagramPlan) -> dict[str, tuple[float, float]]:
    port_positions = defaultdict(lambda: defaultdict(list))
    for edge in plan.edges:
        port_positions[edge.source][edge.source_port].append(edge.source_port_position)
        port_positions[edge.target][edge.target_port].append(edge.target_port_position)

    required = {}
    for node in plan.nodes:
        text_width, text_height = estimate_node_text_size(
            node.text,
            node.font_size_pt,
        )
        required_width = text_width / max(0.1, node.text_block_width_ratio)
        required_height = text_height / max(0.1, node.text_block_height_ratio)
        required_width = max(
            required_width,
            required_port_span(port_positions[node.id]["top"]),
            required_port_span(port_positions[node.id]["bottom"]),
        )
        required_height = max(
            required_height,
            required_port_span(port_positions[node.id]["left"]),
            required_port_span(port_positions[node.id]["right"]),
        )
        evidence = (node.data or {}).get("sizeEvidence", {})
        if isinstance(evidence, dict):
            required_width = max(required_width, float(evidence.get("requiredWidth", 0) or 0))
            required_height = max(required_height, float(evidence.get("requiredHeight", 0) or 0))
        required[node.id] = (required_width, required_height)

    size_class_required = defaultdict(lambda: [0.0, 0.0])
    for node in plan.nodes:
        if not node.size_class:
            continue
        width, height = required[node.id]
        size_class_required[node.size_class][0] = max(
            size_class_required[node.size_class][0],
            width,
        )
        size_class_required[node.size_class][1] = max(
            size_class_required[node.size_class][1],
            height,
        )
    for node in plan.nodes:
        if node.size_class:
            required[node.id] = tuple(size_class_required[node.size_class])
    return required


def structural_quality(plan: DiagramPlan) -> dict:
    page = plan.document.get("page", {})
    page_width = float(page.get("width", 16))
    page_height = float(page.get("height", 9))
    margin = float(page.get("margin", 0.5))
    layout = plan.document.get("layout", {})
    engine = layout.get("engine", "layered")
    direction = layout.get("direction", "LR")
    uniform_node_size = bool(layout.get("uniformNodeSize", True))
    alignment_tolerance = 0.05
    findings: list[Finding] = []
    required_sizes = _node_required_sizes(plan)
    max_text_width_ratio = 0.0
    max_text_height_ratio = 0.0
    oversized_node_count = 0
    undersized_node_count = 0
    node_text_block_utilization_nonstandard_count = 0
    font_below_readable_minimum_count = 0
    font_role_size_drift_count = 0
    max_font_role_size_span = 0.0
    text_block_width_ratios: list[float] = []
    text_block_height_ratios: list[float] = []
    style_profile = resolve_document_style(plan.document)
    typography = style_profile["typography"]
    default_text_block_width_ratio = float(
        typography.get("nodeTextBlockWidthRatio", 0.8)
    )
    default_text_block_height_ratio = float(
        typography.get("nodeTextBlockHeightRatio", 0.8)
    )
    for node in plan.nodes:
        outside = (
            node.x - node.width / 2 < margin
            or node.x + node.width / 2 > page_width - margin
            or node.y - node.height / 2 < margin
            or node.y + node.height / 2 > page_height - margin
        )
        if outside:
            findings.append(Finding("error", "OUT_OF_PAGE", f"{node.id} exceeds page bounds", [node.id], True))
        text_width, text_height = estimate_node_text_size(
            node.text,
            node.font_size_pt,
        )
        width_ratio = text_width / max(0.01, node.width)
        height_ratio = text_height / max(0.01, node.height)
        max_text_width_ratio = max(max_text_width_ratio, width_ratio)
        max_text_height_ratio = max(max_text_height_ratio, height_ratio)
        text_block_width_ratios.append(node.text_block_width_ratio)
        text_block_height_ratios.append(node.text_block_height_ratio)
        standard_width_ratio = (
            min(default_text_block_width_ratio, 0.7)
            if node.type == "decision"
            else default_text_block_width_ratio
        )
        standard_height_ratio = (
            min(default_text_block_height_ratio, 0.7)
            if node.type == "decision"
            else default_text_block_height_ratio
        )
        layout_reason = str((node.data or {}).get("textLayoutReason", "")).strip()
        typography_reason = str(
            (node.data or {}).get("typographyReason", "")
        ).strip()
        role_style = resolve_text_role(style_profile, node.font_role)
        readable_minimum = float(
            role_style.get(
                "minimumFontSizePt",
                typography.get("minimumFontSizePt", 10),
            )
        )
        if node.font_size_pt + 0.01 < readable_minimum and not typography_reason:
            font_below_readable_minimum_count += 1
            findings.append(
                Finding(
                    "error",
                    "FONT_BELOW_READABLE_MINIMUM",
                    (
                        f"{node.id} uses {node.font_size_pt:.2f} pt for role "
                        f"{node.font_role}; minimum is {readable_minimum:.2f} pt"
                    ),
                    [node.id],
                    True,
                )
            )
        if (
            abs(node.text_block_width_ratio - standard_width_ratio) > 0.02
            or abs(node.text_block_height_ratio - standard_height_ratio) > 0.02
        ) and not layout_reason:
            node_text_block_utilization_nonstandard_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_TEXT_BLOCK_UTILIZATION_NONSTANDARD",
                    f"{node.id} overrides the standard text safe-area ratio without measured evidence",
                    [node.id],
                    True,
                )
            )
        required_width, required_height = required_sizes[node.id]
        if node.type != "group" and (
            node.width + alignment_tolerance < required_width
            or node.height + alignment_tolerance < required_height
        ):
            undersized_node_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_OVERFLOW_RISK",
                    f"{node.id} is smaller than its content and port safety envelope",
                    [node.id],
                    True,
                )
            )
        oversized_width = node.width > _allowed_dimension(required_width) + alignment_tolerance
        oversized_height = node.height > _allowed_dimension(required_height) + alignment_tolerance
        if node.type != "group" and (oversized_width or oversized_height):
            oversized_node_count += 1
            dimensions = []
            if oversized_width:
                dimensions.append("width")
            if oversized_height:
                dimensions.append("height")
            findings.append(
                Finding(
                    "error",
                    "OVERSIZED_NODE_WITHOUT_REASON",
                    f"{node.id} has excessive {' and '.join(dimensions)} beyond its measured content or architecture need",
                    [node.id],
                    True,
                )
            )

        if node.caption:
            caption_style = resolve_text_role(style_profile, "caption")
            caption_minimum = float(
                caption_style.get(
                    "minimumFontSizePt",
                    typography.get("minimumFontSizePt", 10),
                )
            )
            if (
                node.caption_font_size_pt + 0.01 < caption_minimum
                and not typography_reason
            ):
                font_below_readable_minimum_count += 1
                findings.append(
                    Finding(
                        "error",
                        "FONT_BELOW_READABLE_MINIMUM",
                        (
                            f"{node.id} caption uses "
                            f"{node.caption_font_size_pt:.2f} pt; minimum is "
                            f"{caption_minimum:.2f} pt"
                        ),
                        [node.id],
                        True,
                    )
                )

    font_role_groups = defaultdict(list)
    for node in plan.nodes:
        if node.size_class:
            font_role_groups[(node.font_role, node.size_class)].append(node)
    for (font_role, size_class), nodes in sorted(font_role_groups.items()):
        if len(nodes) < 2:
            continue
        span = max(node.font_size_pt for node in nodes) - min(
            node.font_size_pt for node in nodes
        )
        max_font_role_size_span = max(max_font_role_size_span, span)
        if span > 0.25:
            font_role_size_drift_count += 1
            findings.append(
                Finding(
                    "error",
                    "FONT_ROLE_SIZE_DRIFT",
                    (
                        f"role {font_role} size class {size_class} spans "
                        f"{span:.2f} pt"
                    ),
                    [node.id for node in nodes],
                    True,
                )
            )

    for edge in plan.edges:
        role_style = resolve_text_role(style_profile, edge.font_role)
        readable_minimum = float(
            role_style.get(
                "minimumFontSizePt",
                typography.get("minimumFontSizePt", 10),
            )
        )
        if edge.font_size_pt + 0.01 < readable_minimum:
            font_below_readable_minimum_count += 1
            findings.append(
                Finding(
                    "error",
                    "FONT_BELOW_READABLE_MINIMUM",
                    (
                        f"{edge.id} uses {edge.font_size_pt:.2f} pt for role "
                        f"{edge.font_role}; minimum is {readable_minimum:.2f} pt"
                    ),
                    [edge.id],
                    True,
                )
            )
    width_span = 0.0
    height_span = 0.0
    if plan.nodes:
        widths = [node.width for node in plan.nodes]
        heights = [node.height for node in plan.nodes]
        width_span = max(widths) - min(widths)
        height_span = max(heights) - min(heights)
        if uniform_node_size and (width_span > alignment_tolerance or height_span > alignment_tolerance):
            findings.append(
                Finding(
                    "error",
                    "INCONSISTENT_NODE_SIZE",
                    "uniform node sizing is enabled but node bounds differ",
                    [node.id for node in plan.nodes],
                    True,
                )
            )

    size_classes = defaultdict(list)
    for node in plan.nodes:
        if node.size_class:
            size_classes[node.size_class].append(node)
    inconsistent_size_class_count = 0
    for size_class, nodes in sorted(size_classes.items()):
        class_width_span = max(node.width for node in nodes) - min(node.width for node in nodes)
        class_height_span = max(node.height for node in nodes) - min(node.height for node in nodes)
        if class_width_span > alignment_tolerance or class_height_span > alignment_tolerance:
            inconsistent_size_class_count += 1
            findings.append(
                Finding(
                    "error",
                    "SIZE_CLASS_INCONSISTENT",
                    f"size class {size_class} contains inconsistent bounds",
                    [node.id for node in nodes],
                    True,
                )
            )

    layer_groups = defaultdict(list)
    order_groups = defaultdict(list)
    for node in plan.nodes:
        layer_groups[node.layer].append(node)
        order_groups[node.order].append(node)

    axis_size_inconsistent_count = 0
    axis_groups = order_groups if direction in ("LR", "RL") else layer_groups
    for axis, nodes in sorted(axis_groups.items()):
        comparable = [
            node
            for node in nodes
            if node.type not in ("group", "note", "junction")
        ]
        if len(comparable) < 2:
            continue
        width_comparable = [
            node
            for node in comparable
            if float(
                ((node.data or {}).get("sizeEvidence") or {}).get(
                    "requiredWidth",
                    0,
                )
                or 0
            )
            <= 0
        ]
        height_comparable = [
            node
            for node in comparable
            if float(
                ((node.data or {}).get("sizeEvidence") or {}).get(
                    "requiredHeight",
                    0,
                )
                or 0
            )
            <= 0
        ]
        inconsistent_dimensions = []
        if len(width_comparable) >= 2:
            max_width = max(node.width for node in width_comparable)
            min_width = min(node.width for node in width_comparable)
            width_can_be_common = all(
                max_width
                <= _allowed_dimension(required_sizes[node.id][0])
                + alignment_tolerance
                for node in width_comparable
            )
            if (
                max_width - min_width > alignment_tolerance
                and width_can_be_common
            ):
                inconsistent_dimensions.append("width")
        if len(height_comparable) >= 2:
            max_height = max(node.height for node in height_comparable)
            min_height = min(node.height for node in height_comparable)
            height_can_be_common = all(
                max_height
                <= _allowed_dimension(required_sizes[node.id][1])
                + alignment_tolerance
                for node in height_comparable
            )
            if (
                max_height - min_height > alignment_tolerance
                and height_can_be_common
            ):
                inconsistent_dimensions.append("height")
        if inconsistent_dimensions:
            axis_size_inconsistent_count += 1
            findings.append(
                Finding(
                    "error",
                    "AXIS_SIZE_INCONSISTENT",
                    (
                        f"axis {axis} can share a common "
                        f"{' and '.join(inconsistent_dimensions)} without "
                        f"violating content limits"
                    ),
                    [node.id for node in comparable],
                    True,
                )
            )

    max_layer_alignment_error = 0.0
    for layer, nodes in layer_groups.items():
        values = [node.x if direction in ("LR", "RL") else node.y for node in nodes]
        error = max(values) - min(values)
        max_layer_alignment_error = max(max_layer_alignment_error, error)
        if error > alignment_tolerance:
            findings.append(
                Finding(
                    "error",
                    "LAYER_MISALIGNED",
                    f"layer {layer} nodes do not share a common center axis",
                    [node.id for node in nodes],
                    True,
                )
            )

    max_order_alignment_error = 0.0
    for order, nodes in order_groups.items():
        if len(nodes) < 2 or len({node.layer for node in nodes}) < 2:
            continue
        values = [node.y if direction in ("LR", "RL") else node.x for node in nodes]
        error = max(values) - min(values)
        max_order_alignment_error = max(max_order_alignment_error, error)
        if error > alignment_tolerance:
            findings.append(
                Finding(
                    "error",
                    "ORDER_MISALIGNED",
                    f"order {order} nodes do not share a common row or column",
                    [node.id for node in nodes],
                    True,
                )
            )

    layer_centers = []
    for layer in sorted(layer_groups):
        nodes = layer_groups[layer]
        values = [node.x if direction in ("LR", "RL") else node.y for node in nodes]
        layer_centers.append(sum(values) / len(values))
    layer_gaps = [abs(second - first) for first, second in zip(layer_centers, layer_centers[1:])]
    layer_gap_span = max(layer_gaps) - min(layer_gaps) if layer_gaps else 0.0
    if engine != "manual" and layer_gap_span > alignment_tolerance:
        findings.append(
            Finding(
                "warning",
                "INCONSISTENT_LAYER_SPACING",
                "layer center spacing is inconsistent",
                [node.id for node in plan.nodes],
                True,
            )
        )

    same_axis_peer_gap_groups = _same_axis_peer_gap_groups(plan)
    same_axis_peer_gap_inconsistent_count = 0
    max_same_axis_peer_gap_span = 0.0
    for (size_class, axis_id), items in sorted(same_axis_peer_gap_groups.items()):
        gaps = [item[1] for item in items]
        gap_span = max(gaps) - min(gaps)
        max_same_axis_peer_gap_span = max(max_same_axis_peer_gap_span, gap_span)
        if gap_span > 0.03:
            same_axis_peer_gap_inconsistent_count += 1
            findings.append(
                Finding(
                    "error",
                    "SAME_AXIS_GAP_INCONSISTENT",
                    (
                        f"size class {size_class} axis {axis_id} uses unequal absolute "
                        f"boundary gaps: {', '.join(f'{gap:.3f}' for gap in gaps)} in"
                    ),
                    sorted(
                        {
                            node_id
                            for _edge_id, _gap, source_id, target_id in items
                            for node_id in (source_id, target_id)
                        }
                    ),
                    True,
                )
            )

    for index, first in enumerate(plan.nodes):
        for second in plan.nodes[index + 1 :]:
            if first.type == "group" or second.type == "group":
                continue
            if _overlap(first, second) > 0.02:
                findings.append(
                    Finding(
                        "error",
                        "NODE_OVERLAP",
                        f"{first.id} overlaps {second.id}",
                        [first.id, second.id],
                        True,
                    )
                )

    nodes_by_id = {node.id: node for node in plan.nodes}
    container_member_outside_count = 0
    container_padding_low_count = 0
    container_excess_slack_count = 0
    for group in plan.groups:
        container = nodes_by_id.get(str(group.get("id", "")))
        members = [
            nodes_by_id[member_id]
            for member_id in group.get("members", [])
            if member_id in nodes_by_id
        ]
        if container is None or container.type != "group" or not members:
            continue
        member_left = min(node.x - node.width / 2 for node in members)
        member_right = max(node.x + node.width / 2 for node in members)
        member_bottom = min(node.y - node.height / 2 for node in members)
        member_top = max(node.y + node.height / 2 for node in members)
        container_left, container_bottom, container_right, container_top = node_rectangle(container)
        paddings = (
            member_left - container_left,
            container_right - member_right,
            member_bottom - container_bottom,
            container_top - member_top,
        )
        if min(paddings) < -alignment_tolerance:
            container_member_outside_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONTAINER_MEMBER_OUTSIDE",
                    f"{container.id} does not fully contain its declared members",
                    [container.id, *[node.id for node in members]],
                    True,
                )
            )
            continue
        member_width = member_right - member_left
        member_height = member_top - member_bottom
        required_padding_x = max(0.3, member_width * 0.05)
        required_padding_y = max(0.3, member_height * 0.05)
        if min(paddings[0], paddings[1]) + alignment_tolerance < required_padding_x or min(
            paddings[2],
            paddings[3],
        ) + alignment_tolerance < required_padding_y:
            container_padding_low_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONTAINER_PADDING_LOW",
                    f"{container.id} has less than the standardized member padding",
                    [container.id, *[node.id for node in members]],
                    True,
                )
            )
        required_width = member_width + 2 * required_padding_x
        required_height = member_height + 2 * required_padding_y
        if (
            container.width > required_width + max(0.6, member_width * 0.1)
            or container.height > required_height + max(0.6, member_height * 0.1)
        ):
            container_excess_slack_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONTAINER_EXCESS_SLACK",
                    f"{container.id} is larger than required by its member envelope and bounded padding",
                    [container.id, *[node.id for node in members]],
                    True,
                )
            )

    compact_layout = bool(layout.get("compact", False))
    content_width_utilization = 0.0
    content_height_utilization = 0.0
    if plan.nodes:
        content_bounds = [node_rectangle(node) for node in plan.nodes]
        content_bounds.extend(
            rectangle(
                node.caption_x,
                node.caption_y,
                node.caption_width,
                node.caption_height,
            )
            for node in plan.nodes
            if node.caption and node.caption_x is not None and node.caption_y is not None
        )
        content_left = min(bounds[0] for bounds in content_bounds)
        content_right = max(bounds[2] for bounds in content_bounds)
        content_bottom = min(bounds[1] for bounds in content_bounds)
        content_top = max(bounds[3] for bounds in content_bounds)
        content_width_utilization = (content_right - content_left) / max(0.01, page_width)
        content_height_utilization = (content_top - content_bottom) / max(0.01, page_height)
        if compact_layout and (
            content_width_utilization < 0.75 or content_height_utilization < 0.65
        ):
            findings.append(
                Finding(
                    "error",
                    "LOW_PAGE_UTILIZATION",
                    "compact layout leaves excessive page space outside the content envelope",
                    [node.id for node in plan.nodes],
                    True,
                )
            )

    planned_segments = [
        (edge.id, start, end)
        for edge in plan.edges
        for start, end in segments(edge.route_points or [])
    ]
    total_connector_length = sum(
        polyline_length(edge.route_points or []) for edge in plan.edges
    )
    total_bend_count = sum(
        max(0, len(simplify_polyline(edge.route_points or [])) - 2)
        for edge in plan.edges
    )
    max_same_axis_gap = 0.0
    max_same_axis_gap_edge = None
    max_allowed_axis_gap = max(0.8, float(layout.get("nodeGap", 0.6)) * 1.5)
    label_bounds: list[tuple[str, tuple[float, float, float, float]]] = []
    text_bounds: list[tuple[str, tuple[float, float, float, float]]] = []
    text_line_overlap_count = 0
    text_line_low_clearance_count = 0
    text_shape_overlap_count = 0
    text_text_overlap_count = 0
    connector_crossing_count = 0
    inefficient_port_count = 0
    misaligned_port_lane_count = 0
    max_port_lane_alignment_error = 0.0
    ambiguous_signal_source_count = 0
    connector_connector_crossing_count = 0
    source_endpoint_intrusion_count = 0
    target_endpoint_intrusion_count = 0
    arrow_terminal_clearance_low_count = 0
    arrowhead_node_overlap_count = 0
    connector_label_anchor_unresolved_count = 0
    connector_label_drift_excessive_count = 0
    max_connector_label_position_shift = 0.0
    max_connector_label_offset = 0.0
    caption_anchor_unresolved_count = 0
    caption_anchor_misaligned_count = 0
    max_caption_anchor_error = 0.0

    for node in plan.nodes:
        if not node.caption or node.caption_x is None or node.caption_y is None:
            continue
        bounds = rectangle(
            node.caption_x,
            node.caption_y,
            node.caption_width,
            node.caption_height,
        )
        text_bounds.append((f"caption:{node.id}", bounds))
        if not node.caption_anchor_resolved:
            caption_anchor_unresolved_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_ANCHOR_UNRESOLVED",
                    f"{node.id} caption cannot be placed inside its bounded anchor rules",
                    [node.id],
                    True,
                )
            )
        expected_x, expected_y = caption_center(
            node,
            node.caption_resolved_side or node.caption_side,
            node.caption_position,
            node.caption_offset,
            node.caption_width,
            node.caption_height,
        )
        anchor_error = math.hypot(node.caption_x - expected_x, node.caption_y - expected_y)
        max_caption_anchor_error = max(max_caption_anchor_error, anchor_error)
        if anchor_error > 0.02:
            caption_anchor_misaligned_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_ANCHOR_MISALIGNED",
                    f"{node.id} caption is detached from its requested shape-side anchor",
                    [node.id],
                    True,
                )
            )
        crossing_edges = sorted(
            {
                edge_id
                for edge_id, start, end in planned_segments
                if segment_intersects_rectangle(start, end, bounds)
            }
        )
        if crossing_edges:
            text_line_overlap_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_LINE_OVERLAP",
                    f"{node.id} caption intersects connector geometry",
                    [node.id, *crossing_edges],
                    True,
                )
            )
        low_clearance_edges = sorted(
            {
                edge_id
                for edge_id, start, end in planned_segments
                if edge_id not in crossing_edges
                and segment_intersects_rectangle(start, end, expand_rectangle(bounds, 0.08))
            }
        )
        if low_clearance_edges:
            text_line_low_clearance_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_LINE_CLEARANCE_LOW",
                    f"{node.id} caption is closer than 0.08 in to connector geometry",
                    [node.id, *low_clearance_edges],
                    True,
                )
            )
        crossing_nodes = [
            other.id
            for other in plan.nodes
            if rectangles_overlap(bounds, node_rectangle(other, 0.02))
        ]
        if crossing_nodes:
            text_shape_overlap_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_SHAPE_OVERLAP",
                    f"{node.id} caption intersects a node frame",
                    [node.id, *crossing_nodes],
                    True,
                )
            )

    for edge in plan.edges:
        source_node = nodes_by_id[edge.source]
        target_node = nodes_by_id[edge.target]
        start_point = port_point(source_node, edge.source_port, edge.source_port_position)
        end_point = port_point(target_node, edge.target_port, edge.target_port_position)
        route_points = simplify_polyline(edge.route_points or [])
        source_segment_length, source_outward_projection = endpoint_segment_state(
            route_points,
            edge.source_port,
            at_start=True,
        )
        target_segment_length, target_outward_projection = endpoint_segment_state(
            route_points,
            edge.target_port,
            at_start=False,
        )
        if source_outward_projection < -0.01:
            source_endpoint_intrusion_count += 1
            findings.append(
                Finding(
                    "error",
                    "SOURCE_ENDPOINT_INTRUSION",
                    f"{edge.id} leaves the source through its interior instead of outward from the selected port",
                    [edge.id, source_node.id],
                    True,
                )
            )
        if target_outward_projection < -0.01:
            target_endpoint_intrusion_count += 1
            findings.append(
                Finding(
                    "error",
                    "TARGET_ENDPOINT_INTRUSION",
                    f"{edge.id} approaches the target through its interior instead of from outside the selected port",
                    [edge.id, target_node.id],
                    True,
                )
            )
        if source_segment_length + 1e-9 < ENDPOINT_TAIL_CLEARANCE:
            findings.append(
                Finding(
                    "error",
                    "SOURCE_TAIL_CLEARANCE_LOW",
                    f"{edge.id} has insufficient visible tail clearance outside the source boundary",
                    [edge.id, source_node.id],
                    True,
                )
            )
        if edge.type in ("directed", "dependency"):
            if target_segment_length + 1e-9 < ARROWHEAD_LENGTH:
                arrow_terminal_clearance_low_count += 1
                findings.append(
                    Finding(
                        "error",
                        "ARROW_TERMINAL_CLEARANCE_LOW",
                        f"{edge.id} terminal segment is too short for the standardized arrowhead",
                        [edge.id, source_node.id, target_node.id],
                        True,
                    )
                )
            arrow_bounds = arrowhead_bounds(route_points)
            if arrow_bounds is not None:
                overlapping_nodes = [
                    node.id
                    for node in plan.nodes
                    if node.id != target_node.id
                    and rectangles_overlap(arrow_bounds, node_rectangle(node))
                ]
                if overlapping_nodes:
                    arrowhead_node_overlap_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "ARROWHEAD_OVERLAPS_NODE",
                            f"{edge.id} arrowhead overlaps a node instead of remaining outside the frame",
                            [edge.id, *overlapping_nodes],
                            True,
                        )
                    )
        if source_node.type == "note" and edge.type in ("directed", "dependency"):
            ambiguous_signal_source_count += 1
            findings.append(
                Finding(
                    "error",
                    "AMBIGUOUS_SIGNAL_SOURCE",
                    f"{edge.id} uses a note as a signal source",
                    [edge.id, source_node.id],
                    True,
                )
            )
        if source_node.id != target_node.id:
            same_column = abs(source_node.x - target_node.x) <= 0.03
            same_row = abs(source_node.y - target_node.y) <= 0.03
            same_axis_series = (
                source_node.order == target_node.order
                and (
                    (direction in ("LR", "RL") and same_row)
                    or (direction in ("TB", "BT") and same_column)
                )
            )
            if same_axis_series:
                if direction in ("LR", "RL"):
                    boundary_gap = abs(source_node.x - target_node.x) - (
                        source_node.width + target_node.width
                    ) / 2
                else:
                    boundary_gap = abs(source_node.y - target_node.y) - (
                        source_node.height + target_node.height
                    ) / 2
                if boundary_gap > max_same_axis_gap:
                    max_same_axis_gap = boundary_gap
                    max_same_axis_gap_edge = edge.id
            if same_column and not same_row:
                if edge.source_port not in ("top", "bottom") or edge.target_port not in ("top", "bottom"):
                    inefficient_port_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "INEFFICIENT_CONNECTOR_PORT",
                            f"{edge.id} can use a direct vertical port pair",
                            [edge.id, source_node.id, target_node.id],
                            True,
                        )
                    )
                lane_error = abs(start_point[0] - end_point[0])
                max_port_lane_alignment_error = max(max_port_lane_alignment_error, lane_error)
                if lane_error > 0.01:
                    misaligned_port_lane_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "PORT_LANE_MISALIGNED",
                            f"{edge.id} vertical port lane is not aligned",
                            [edge.id, source_node.id, target_node.id],
                            True,
                        )
                    )
            elif same_row and not same_column:
                if edge.source_port not in ("left", "right") or edge.target_port not in ("left", "right"):
                    inefficient_port_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "INEFFICIENT_CONNECTOR_PORT",
                            f"{edge.id} can use a direct horizontal port pair",
                            [edge.id, source_node.id, target_node.id],
                            True,
                        )
                    )
                lane_error = abs(start_point[1] - end_point[1])
                max_port_lane_alignment_error = max(max_port_lane_alignment_error, lane_error)
                if lane_error > 0.01:
                    misaligned_port_lane_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "PORT_LANE_MISALIGNED",
                            f"{edge.id} horizontal port lane is not aligned",
                            [edge.id, source_node.id, target_node.id],
                            True,
                        )
                    )

        if edge.label and edge.label_x is not None and edge.label_y is not None:
            if not edge.label_anchor_resolved:
                connector_label_anchor_unresolved_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_ANCHOR_UNRESOLVED",
                        f"{edge.id} label cannot be placed within the bounded midpoint anchor",
                        [edge.id],
                        True,
                    )
                )
            position_shift = abs(edge.label_actual_position - edge.label_position)
            max_connector_label_position_shift = max(
                max_connector_label_position_shift,
                position_shift,
            )
            max_connector_label_offset = max(
                max_connector_label_offset,
                edge.label_actual_offset,
            )
            if position_shift > 0.2:
                connector_label_drift_excessive_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_DRIFT_EXCESSIVE",
                        f"{edge.id} label drifts too far from its requested relative position",
                        [edge.id],
                        True,
                    )
                )
            bounds = rectangle(
                edge.label_x,
                edge.label_y,
                edge.label_width,
                edge.label_height,
            )
            label_bounds.append((edge.id, bounds))
            text_bounds.append((edge.id, bounds))
            crossing_edges = sorted(
                {
                    edge_id
                    for edge_id, start, end in planned_segments
                    if segment_intersects_rectangle(start, end, bounds)
                }
            )
            if crossing_edges:
                text_line_overlap_count += 1
                findings.append(
                    Finding(
                        "error",
                        "TEXT_LINE_OVERLAP",
                        f"{edge.id} label intersects connector geometry",
                        [edge.id, *crossing_edges],
                        True,
                    )
                )
            low_clearance_edges = sorted(
                {
                    edge_id
                    for edge_id, start, end in planned_segments
                    if edge_id not in crossing_edges
                    and segment_intersects_rectangle(start, end, expand_rectangle(bounds, 0.08))
                }
            )
            if low_clearance_edges:
                text_line_low_clearance_count += 1
                findings.append(
                    Finding(
                        "error",
                        "TEXT_LINE_CLEARANCE_LOW",
                        f"{edge.id} label is closer than 0.08 in to connector geometry",
                        [edge.id, *low_clearance_edges],
                        True,
                    )
                )
            crossing_nodes = [
                node.id
                for node in plan.nodes
                if rectangles_overlap(bounds, node_rectangle(node, 0.02))
            ]
            if crossing_nodes:
                text_shape_overlap_count += 1
                findings.append(
                    Finding(
                        "error",
                        "TEXT_SHAPE_OVERLAP",
                        f"{edge.id} label intersects a node frame",
                        [edge.id, *crossing_nodes],
                        True,
                    )
                )

        crossed_nodes = sorted(
            {
                node.id
                for node in plan.nodes
                if node.id not in (edge.source, edge.target)
                for start, end in segments(edge.route_points or [])
                if segment_intersects_rectangle(start, end, node_rectangle(node, -0.04))
            }
        )
        if crossed_nodes:
            connector_crossing_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_CROSSES_NODE",
                    f"{edge.id} crosses an unrelated node",
                    [edge.id, *crossed_nodes],
                    True,
                )
            )

    if (
        compact_layout
        and max_same_axis_gap_edge is not None
        and max_same_axis_gap > max_allowed_axis_gap
    ):
        findings.append(
            Finding(
                "error",
                "LAYOUT_TOO_DISPERSED",
                f"{max_same_axis_gap_edge} leaves {max_same_axis_gap:.2f} in between same-axis peers",
                [max_same_axis_gap_edge],
                True,
            )
        )

    for index, (first_id, first_bounds) in enumerate(text_bounds):
        for second_id, second_bounds in text_bounds[index + 1 :]:
            if rectangles_overlap(first_bounds, second_bounds):
                text_text_overlap_count += 1
                findings.append(
                    Finding(
                        "error",
                        "TEXT_TEXT_OVERLAP",
                        f"{first_id} and {second_id} labels overlap",
                        [first_id, second_id],
                        True,
                    )
                )

    crossing_pairs = set()
    for index, (first_id, first_start, first_end) in enumerate(planned_segments):
        for second_id, second_start, second_end in planned_segments[index + 1 :]:
            if first_id == second_id:
                continue
            if segment_crosses_segment(first_start, first_end, second_start, second_end):
                crossing_pairs.add(tuple(sorted((first_id, second_id))))
    for first_id, second_id in sorted(crossing_pairs):
        connector_connector_crossing_count += 1
        findings.append(
            Finding(
                "error",
                "CONNECTOR_CROSSING",
                f"{first_id} crosses {second_id} without a junction",
                [first_id, second_id],
                True,
            )
        )

    score = max(0, 100 - sum(15 if item.severity == "error" else 5 for item in findings))
    return {
        "score": score,
        "findingCount": len(findings),
        "findings": [asdict(item) for item in findings],
        "metrics": {
            "nodes": len(plan.nodes),
            "edges": len(plan.edges),
            "pageWidth": page_width,
            "pageHeight": page_height,
            "uniformNodeSize": uniform_node_size,
            "widthSpan": round(width_span, 4),
            "heightSpan": round(height_span, 4),
            "maxLayerAlignmentError": round(max_layer_alignment_error, 4),
            "maxOrderAlignmentError": round(max_order_alignment_error, 4),
            "layerGapSpan": round(layer_gap_span, 4),
            "sameAxisPeerGapGroupCount": len(same_axis_peer_gap_groups),
            "sameAxisPeerGapInconsistentCount": same_axis_peer_gap_inconsistent_count,
            "maxSameAxisPeerGapSpan": round(max_same_axis_peer_gap_span, 4),
            "connectorLabels": len(label_bounds),
            "textLineOverlapCount": text_line_overlap_count,
            "textLineLowClearanceCount": text_line_low_clearance_count,
            "textShapeOverlapCount": text_shape_overlap_count,
            "textTextOverlapCount": text_text_overlap_count,
            "connectorCrossingCount": connector_crossing_count,
            "inefficientPortCount": inefficient_port_count,
            "misalignedPortLaneCount": misaligned_port_lane_count,
            "maxPortLaneAlignmentError": round(max_port_lane_alignment_error, 4),
            "ambiguousSignalSourceCount": ambiguous_signal_source_count,
            "connectorConnectorCrossingCount": connector_connector_crossing_count,
            "sizeClassCount": len(size_classes),
            "inconsistentSizeClassCount": inconsistent_size_class_count,
            "axisSizeInconsistentCount": axis_size_inconsistent_count,
            "maxTextWidthRatio": round(max_text_width_ratio, 4),
            "maxTextHeightRatio": round(max_text_height_ratio, 4),
            "minNodeTextBlockWidthRatio": round(min(text_block_width_ratios, default=0.0), 4),
            "maxNodeTextBlockWidthRatio": round(max(text_block_width_ratios, default=0.0), 4),
            "minNodeTextBlockHeightRatio": round(min(text_block_height_ratios, default=0.0), 4),
            "maxNodeTextBlockHeightRatio": round(max(text_block_height_ratios, default=0.0), 4),
            "nodeTextBlockUtilizationNonstandardCount": node_text_block_utilization_nonstandard_count,
            "fontBelowReadableMinimumCount": font_below_readable_minimum_count,
            "fontRoleSizeDriftCount": font_role_size_drift_count,
            "maxFontRoleSizeSpanPt": round(max_font_role_size_span, 4),
            "undersizedNodeCount": undersized_node_count,
            "oversizedNodeCount": oversized_node_count,
            "containerMemberOutsideCount": container_member_outside_count,
            "containerPaddingLowCount": container_padding_low_count,
            "containerExcessSlackCount": container_excess_slack_count,
            "compactLayout": compact_layout,
            "contentWidthUtilization": round(content_width_utilization, 4),
            "contentHeightUtilization": round(content_height_utilization, 4),
            "maxSameAxisGap": round(max_same_axis_gap, 4),
            "maxAllowedAxisGap": round(max_allowed_axis_gap, 4),
            "totalConnectorLength": round(total_connector_length, 4),
            "totalBendCount": total_bend_count,
            "sourceEndpointIntrusionCount": source_endpoint_intrusion_count,
            "targetEndpointIntrusionCount": target_endpoint_intrusion_count,
            "arrowTerminalClearanceLowCount": arrow_terminal_clearance_low_count,
            "arrowheadNodeOverlapCount": arrowhead_node_overlap_count,
            "connectorLabelAnchorUnresolvedCount": connector_label_anchor_unresolved_count,
            "connectorLabelDriftExcessiveCount": connector_label_drift_excessive_count,
            "maxConnectorLabelPositionShift": round(max_connector_label_position_shift, 4),
            "maxConnectorLabelOffset": round(max_connector_label_offset, 4),
            "captions": len(
                [
                    node
                    for node in plan.nodes
                    if node.caption and node.caption_x is not None and node.caption_y is not None
                ]
            ),
            "captionAnchorUnresolvedCount": caption_anchor_unresolved_count,
            "captionAnchorMisalignedCount": caption_anchor_misaligned_count,
            "maxCaptionAnchorError": round(max_caption_anchor_error, 4),
        },
    }


def connector_quality(plan: DiagramPlan, inspection: dict) -> dict:
    findings: list[Finding] = []
    nodes = {item["id"]: item for item in inspection.get("nodes", [])}
    expected_nodes = {item.id: item for item in plan.nodes}
    actual_edges = {item["id"]: item for item in inspection.get("edges", [])}
    actual_captions = {
        item.get("ownerNodeId", item["id"]): item
        for item in inspection.get("captions", [])
    }
    expected_edges = {edge.id: edge for edge in plan.edges}
    endpoint_tolerance = 0.03
    max_endpoint_error = 0.0
    misaligned_endpoint_count = 0
    route_style_mismatch_count = 0
    missing_connector_count = 0
    not_fully_glued_count = 0
    fully_glued_count = 0
    node_text_misaligned_count = 0
    max_node_text_center_error = 0.0
    source_endpoint_intrusion_count = 0
    target_endpoint_intrusion_count = 0
    arrow_terminal_clearance_low_count = 0
    arrowhead_node_overlap_count = 0
    arrow_geometry_unverified_count = 0
    node_text_block_utilization_mismatch_count = 0
    max_node_text_block_ratio_error = 0.0
    node_font_mismatch_count = 0
    node_asian_font_mismatch_count = 0
    node_font_size_mismatch_count = 0
    node_font_style_mismatch_count = 0
    node_text_color_mismatch_count = 0
    node_line_color_mismatch_count = 0
    node_fill_color_mismatch_count = 0
    node_line_weight_mismatch_count = 0
    node_corner_radius_mismatch_count = 0
    max_font_size_error_pt = 0.0
    max_line_weight_error_pt = 0.0
    max_node_corner_radius_error_in = 0.0
    connector_label_anchor_misaligned_count = 0
    max_connector_label_anchor_error = 0.0
    connector_label_font_mismatch_count = 0
    connector_label_asian_font_mismatch_count = 0
    connector_label_font_size_mismatch_count = 0
    connector_label_font_style_mismatch_count = 0
    connector_label_text_color_mismatch_count = 0
    connector_line_color_mismatch_count = 0
    connector_line_weight_mismatch_count = 0
    caption_missing_count = 0
    caption_anchor_misaligned_count = 0
    max_caption_anchor_error = 0.0
    caption_font_mismatch_count = 0
    caption_asian_font_mismatch_count = 0
    caption_font_size_mismatch_count = 0
    caption_font_style_mismatch_count = 0
    caption_text_color_mismatch_count = 0
    live_font_role_size_drift_count = 0
    max_live_font_role_size_span = 0.0
    same_axis_peer_gap_inconsistent_count = 0
    max_same_axis_peer_gap_span = 0.0

    for node_id, node in nodes.items():
        expected_node = expected_nodes.get(node_id)
        text_center_error = math.hypot(
            float(node.get("textPinX", node["x"])) - float(node["x"]),
            float(node.get("textPinY", node["y"])) - float(node["y"]),
        )
        max_node_text_center_error = max(max_node_text_center_error, text_center_error)
        centered = (
            bool(node.get("textBlockExists", False))
            and int(round(float(node.get("horizontalAlign", -1)))) == 1
            and int(round(float(node.get("verticalAlign", -1)))) == 1
            and text_center_error <= 0.02
        )
        if not centered:
            node_text_misaligned_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_TEXT_NOT_CENTERED",
                    f"{node_id} text block is not horizontally and vertically centered",
                    [node_id],
                    True,
                )
            )
        if expected_node is None:
            continue
        width_ratio = float(
            node.get(
                "textWidthRatio",
                float(node.get("textWidth", 0.0)) / max(0.01, float(node["width"])),
            )
        )
        height_ratio = float(
            node.get(
                "textHeightRatio",
                float(node.get("textHeight", 0.0)) / max(0.01, float(node["height"])),
            )
        )
        ratio_error = max(
            abs(width_ratio - expected_node.text_block_width_ratio),
            abs(height_ratio - expected_node.text_block_height_ratio),
        )
        max_node_text_block_ratio_error = max(
            max_node_text_block_ratio_error,
            ratio_error,
        )
        if ratio_error > 0.02:
            node_text_block_utilization_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_TEXT_BLOCK_UTILIZATION_OUT_OF_RANGE",
                    f"{node_id} live text block does not match the bounded safe-area ratio",
                    [node_id],
                    True,
                )
            )
        expected_font = _normalized_font_name(expected_node.font_family)
        if _normalized_font_name(node.get("fontName")) != expected_font:
            node_font_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_FONT_MISMATCH",
                    f"{node_id} does not use the requested font family",
                    [node_id],
                    True,
                )
            )
        expected_asian_font = _normalized_font_name(
            expected_node.asian_font_family
        )
        if (
            _normalized_font_name(node.get("asianFontName"))
            != expected_asian_font
        ):
            node_asian_font_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_ASIAN_FONT_MISMATCH",
                    f"{node_id} Asian font does not use the requested font family",
                    [node_id],
                    True,
                )
            )
        font_size_error = abs(
            float(node.get("fontSizePt", 0.0)) - expected_node.font_size_pt
        )
        max_font_size_error_pt = max(max_font_size_error_pt, font_size_error)
        if font_size_error > 0.25:
            node_font_size_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_FONT_SIZE_MISMATCH",
                    f"{node_id} does not use the requested role font size",
                    [node_id],
                    True,
                )
            )
        expected_char_style = char_style_code(
            expected_node.font_weight,
            expected_node.font_style,
        )
        if int(node.get("charStyle", -1)) != expected_char_style:
            node_font_style_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_FONT_STYLE_MISMATCH",
                    f"{node_id} does not use the requested font weight/style",
                    [node_id],
                    True,
                )
            )
        if _normalized_color(node.get("textColor")) != expected_node.text_color:
            node_text_color_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_TEXT_COLOR_MISMATCH",
                    f"{node_id} does not use the requested text color",
                    [node_id],
                    True,
                )
            )
        if _normalized_color(node.get("lineColor")) != expected_node.line_color:
            node_line_color_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_LINE_COLOR_MISMATCH",
                    f"{node_id} does not use the requested frame color",
                    [node_id],
                    True,
                )
            )
        if _normalized_color(node.get("fillColor")) != expected_node.fill_color:
            node_fill_color_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_FILL_COLOR_MISMATCH",
                    f"{node_id} does not use the requested fill color",
                    [node_id],
                    True,
                )
            )
        line_weight_error = abs(
            float(node.get("lineWeightPt", 0.0))
            - expected_node.line_weight_pt
        )
        max_line_weight_error_pt = max(
            max_line_weight_error_pt,
            line_weight_error,
        )
        if line_weight_error > 0.1:
            node_line_weight_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_LINE_WEIGHT_MISMATCH",
                    f"{node_id} does not use the requested frame weight",
                    [node_id],
                    True,
                )
            )
        corner_radius_error = abs(
            float(node.get("cornerRadiusIn", 0.0))
            - expected_node.corner_radius_in
        )
        max_node_corner_radius_error_in = max(
            max_node_corner_radius_error_in,
            corner_radius_error,
        )
        if corner_radius_error > 0.005:
            node_corner_radius_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "NODE_CORNER_RADIUS_MISMATCH",
                    f"{node_id} does not use the requested corner radius",
                    [node_id],
                    True,
                )
            )

    live_font_role_groups = defaultdict(list)
    for expected_node in plan.nodes:
        if not expected_node.size_class:
            continue
        actual_node = nodes.get(expected_node.id)
        if actual_node is None:
            continue
        live_font_role_groups[
            (expected_node.font_role, expected_node.size_class)
        ].append((expected_node.id, float(actual_node.get("fontSizePt", 0.0))))
    for (font_role, size_class), items in sorted(live_font_role_groups.items()):
        if len(items) < 2:
            continue
        sizes = [item[1] for item in items]
        span = max(sizes) - min(sizes)
        max_live_font_role_size_span = max(
            max_live_font_role_size_span,
            span,
        )
        if span > 0.25:
            live_font_role_size_drift_count += 1
            findings.append(
                Finding(
                    "error",
                    "FONT_ROLE_SIZE_DRIFT",
                    (
                        f"live role {font_role} size class {size_class} "
                        f"spans {span:.2f} pt"
                    ),
                    [item[0] for item in items],
                    True,
                )
            )

    live_same_axis_peer_gap_groups = _same_axis_peer_gap_groups(plan, nodes)
    for (size_class, axis_id), items in sorted(live_same_axis_peer_gap_groups.items()):
        gaps = [item[1] for item in items]
        gap_span = max(gaps) - min(gaps)
        max_same_axis_peer_gap_span = max(max_same_axis_peer_gap_span, gap_span)
        if gap_span > 0.03:
            same_axis_peer_gap_inconsistent_count += 1
            findings.append(
                Finding(
                    "error",
                    "SAME_AXIS_GAP_INCONSISTENT",
                    (
                        f"live size class {size_class} axis {axis_id} uses unequal "
                        f"absolute boundary gaps: {', '.join(f'{gap:.3f}' for gap in gaps)} in"
                    ),
                    sorted(
                        {
                            node_id
                            for _edge_id, _gap, source_id, target_id in items
                            for node_id in (source_id, target_id)
                        }
                    ),
                    True,
                )
            )

    actual_segments: list[tuple[str, tuple[float, float], tuple[float, float]]] = []
    label_bounds: list[tuple[str, tuple[float, float, float, float]]] = []
    caption_bounds: list[tuple[str, tuple[float, float, float, float]]] = []
    for expected_node in plan.nodes:
        if (
            not expected_node.caption
            or expected_node.caption_x is None
            or expected_node.caption_y is None
        ):
            continue
        actual_caption = actual_captions.get(expected_node.id)
        if actual_caption is None:
            caption_missing_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_NOT_RENDERED",
                    f"{expected_node.id} caption is missing from the live page",
                    [expected_node.id],
                    False,
                )
            )
            continue
        anchor_error = math.hypot(
            float(actual_caption["x"]) - expected_node.caption_x,
            float(actual_caption["y"]) - expected_node.caption_y,
        )
        max_caption_anchor_error = max(max_caption_anchor_error, anchor_error)
        if anchor_error > 0.02:
            caption_anchor_misaligned_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_ANCHOR_MISALIGNED",
                    f"{expected_node.id} live caption is detached from its shape anchor",
                    [expected_node.id],
                    True,
                )
            )
        expected_font = _normalized_font_name(expected_node.font_family)
        if _normalized_font_name(actual_caption.get("fontName")) != expected_font:
            caption_font_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_FONT_MISMATCH",
                    f"{expected_node.id} caption does not use the requested font family",
                    [expected_node.id],
                    True,
                )
            )
        expected_asian_font = _normalized_font_name(
            expected_node.asian_font_family
        )
        if (
            _normalized_font_name(actual_caption.get("asianFontName"))
            != expected_asian_font
        ):
            caption_asian_font_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_ASIAN_FONT_MISMATCH",
                    f"{expected_node.id} caption Asian font does not use the requested font family",
                    [expected_node.id],
                    True,
                )
            )
        font_size_error = abs(
            float(actual_caption.get("fontSizePt", 0.0))
            - expected_node.caption_font_size_pt
        )
        max_font_size_error_pt = max(max_font_size_error_pt, font_size_error)
        if font_size_error > 0.25:
            caption_font_size_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_FONT_SIZE_MISMATCH",
                    f"{expected_node.id} caption does not use the requested role font size",
                    [expected_node.id],
                    True,
                )
            )
        expected_char_style = char_style_code(
            expected_node.caption_font_weight,
            expected_node.caption_font_style,
        )
        if int(actual_caption.get("charStyle", -1)) != expected_char_style:
            caption_font_style_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_FONT_STYLE_MISMATCH",
                    f"{expected_node.id} caption font style differs from the plan",
                    [expected_node.id],
                    True,
                )
            )
        if (
            _normalized_color(actual_caption.get("textColor"))
            != expected_node.text_color
        ):
            caption_text_color_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CAPTION_TEXT_COLOR_MISMATCH",
                    f"{expected_node.id} caption does not use the requested text color",
                    [expected_node.id],
                    True,
                )
            )
        caption_bounds.append(
            (
                f"caption:{expected_node.id}",
                rectangle(
                    float(actual_caption["x"]),
                    float(actual_caption["y"]),
                    float(actual_caption["width"]),
                    float(actual_caption["height"]),
                ),
            )
        )
    for edge in plan.edges:
        actual = actual_edges.get(edge.id)
        if actual is None:
            missing_connector_count += 1
            findings.append(
                Finding("error", "CONNECTOR_MISSING", f"{edge.id} is missing from the live page", [edge.id], False)
            )
            continue

        source = nodes.get(edge.source)
        target = nodes.get(edge.target)
        if source is not None and target is not None:
            expected_begin = port_point(source, edge.source_port, edge.source_port_position)
            expected_end = port_point(target, edge.target_port, edge.target_port_position)
            begin_error = math.hypot(
                float(actual.get("beginX", expected_begin[0])) - expected_begin[0],
                float(actual.get("beginY", expected_begin[1])) - expected_begin[1],
            )
            end_error = math.hypot(
                float(actual.get("endX", expected_end[0])) - expected_end[0],
                float(actual.get("endY", expected_end[1])) - expected_end[1],
            )
            edge_error = max(begin_error, end_error)
            max_endpoint_error = max(max_endpoint_error, edge_error)
            if edge_error > endpoint_tolerance:
                misaligned_endpoint_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_ENDPOINT_MISALIGNED",
                        f"{edge.id} endpoint is not on the requested side center",
                        [edge.id, edge.source, edge.target],
                        True,
                    )
                )

        expected_route_style = 1 if edge.routing == "orthogonal" else 2
        actual_route_style = actual.get("shapeRouteStyle")
        if actual_route_style is not None and int(round(float(actual_route_style))) != expected_route_style:
            route_style_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_ROUTE_STYLE_MISMATCH",
                    f"{edge.id} route style differs from the plan",
                    [edge.id],
                    True,
                )
            )
        if _normalized_color(actual.get("lineColor")) != edge.line_color:
            connector_line_color_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_LINE_COLOR_MISMATCH",
                    f"{edge.id} does not use the requested line color",
                    [edge.id],
                    True,
                )
            )
        line_weight_error = abs(
            float(actual.get("lineWeightPt", 0.0)) - edge.line_weight_pt
        )
        max_line_weight_error_pt = max(
            max_line_weight_error_pt,
            line_weight_error,
        )
        if line_weight_error > 0.1:
            connector_line_weight_mismatch_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_LINE_WEIGHT_MISMATCH",
                    f"{edge.id} does not use the requested line weight",
                    [edge.id],
                    True,
                )
            )

        glue_count = actual.get("glueCount")
        if glue_count is not None:
            if int(glue_count) >= 2:
                fully_glued_count += 1
            else:
                not_fully_glued_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_NOT_FULLY_GLUED",
                        f"{edge.id} is not glued at both endpoints",
                        [edge.id, edge.source, edge.target],
                        True,
                    )
                )

        raw_points = actual.get("pathPoints") or []
        points = simplify_polyline(
            [(float(point[0]), float(point[1])) for point in raw_points]
        )
        if len(points) < 2 and "beginX" in actual and "endX" in actual:
            points = [
                (float(actual["beginX"]), float(actual["beginY"])),
                (float(actual["endX"]), float(actual["endY"])),
            ]
        source_segment_length, source_outward_projection = endpoint_segment_state(
            points,
            edge.source_port,
            at_start=True,
        )
        target_segment_length, target_outward_projection = endpoint_segment_state(
            points,
            edge.target_port,
            at_start=False,
        )
        if source_outward_projection < -0.01:
            source_endpoint_intrusion_count += 1
            findings.append(
                Finding(
                    "error",
                    "SOURCE_ENDPOINT_INTRUSION",
                    f"{edge.id} live path leaves the source through its interior",
                    [edge.id, edge.source],
                    True,
                )
            )
        if target_outward_projection < -0.01:
            target_endpoint_intrusion_count += 1
            findings.append(
                Finding(
                    "error",
                    "TARGET_ENDPOINT_INTRUSION",
                    f"{edge.id} live path approaches the target through its interior",
                    [edge.id, edge.target],
                    True,
                )
            )
        if source_segment_length + 1e-9 < ENDPOINT_TAIL_CLEARANCE:
            findings.append(
                Finding(
                    "error",
                    "SOURCE_TAIL_CLEARANCE_LOW",
                    f"{edge.id} live tail is too short outside the source frame",
                    [edge.id, edge.source],
                    True,
                )
            )
        if edge.type in ("directed", "dependency"):
            arrow_geometry_verified = (
                int(round(float(actual.get("beginArrow", -1)))) == 0
                and int(round(float(actual.get("endArrow", -1)))) == 13
                and int(round(float(actual.get("endArrowSize", -1)))) == 2
                and float(actual.get("lineWeight", 0)) > 0
            )
            if not arrow_geometry_verified:
                arrow_geometry_unverified_count += 1
                findings.append(
                    Finding(
                        "error",
                        "ARROW_GEOMETRY_UNVERIFIED",
                        f"{edge.id} live arrow style, size, or line weight is not the calibrated contract",
                        [edge.id],
                        True,
                    )
                )
            if target_segment_length + 1e-9 < ARROWHEAD_LENGTH:
                arrow_terminal_clearance_low_count += 1
                findings.append(
                    Finding(
                        "error",
                        "ARROW_TERMINAL_CLEARANCE_LOW",
                        f"{edge.id} live terminal segment is too short for its arrowhead",
                        [edge.id, edge.source, edge.target],
                        True,
                    )
                )
            live_arrow_bounds = arrowhead_bounds(points)
            if live_arrow_bounds is not None:
                overlapping_nodes = [
                    node_id
                    for node_id, node in nodes.items()
                    if node_id != edge.target
                    and rectangles_overlap(live_arrow_bounds, node_rectangle(node))
                ]
                if overlapping_nodes:
                    arrowhead_node_overlap_count += 1
                    findings.append(
                        Finding(
                            "error",
                            "ARROWHEAD_OVERLAPS_NODE",
                            f"{edge.id} live arrowhead overlaps a node",
                            [edge.id, *overlapping_nodes],
                            True,
                        )
                    )
        actual_segments.extend((edge.id, start, end) for start, end in segments(points))

        if edge.label and all(
            key in actual for key in ("labelX", "labelY", "labelWidth", "labelHeight")
        ):
            anchor_error = math.hypot(
                float(actual["labelX"]) - float(edge.label_x),
                float(actual["labelY"]) - float(edge.label_y),
            )
            max_connector_label_anchor_error = max(
                max_connector_label_anchor_error,
                anchor_error,
            )
            if anchor_error > 0.03:
                connector_label_anchor_misaligned_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_ANCHOR_MISALIGNED",
                        f"{edge.id} live label is detached from its planned connector anchor",
                        [edge.id],
                        True,
                    )
                )
            expected_font = _normalized_font_name(edge.font_family)
            if _normalized_font_name(actual.get("fontName")) != expected_font:
                connector_label_font_mismatch_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_FONT_MISMATCH",
                        f"{edge.id} label does not use the requested font family",
                        [edge.id],
                    True,
                )
            )
            expected_asian_font = _normalized_font_name(
                edge.asian_font_family
            )
            if (
                _normalized_font_name(actual.get("asianFontName"))
                != expected_asian_font
            ):
                connector_label_asian_font_mismatch_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_ASIAN_FONT_MISMATCH",
                        f"{edge.id} label Asian font does not use the requested font family",
                        [edge.id],
                        True,
                    )
                )
            font_size_error = abs(
                float(actual.get("fontSizePt", 0.0)) - edge.font_size_pt
            )
            max_font_size_error_pt = max(
                max_font_size_error_pt,
                font_size_error,
            )
            if font_size_error > 0.25:
                connector_label_font_size_mismatch_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_FONT_SIZE_MISMATCH",
                        f"{edge.id} label does not use the requested role font size",
                        [edge.id],
                        True,
                    )
                )
            expected_char_style = char_style_code(
                edge.font_weight,
                edge.font_style,
            )
            if int(actual.get("charStyle", -1)) != expected_char_style:
                connector_label_font_style_mismatch_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_FONT_STYLE_MISMATCH",
                        f"{edge.id} label font style differs from the plan",
                        [edge.id],
                        True,
                    )
                )
            if _normalized_color(actual.get("textColor")) != edge.text_color:
                connector_label_text_color_mismatch_count += 1
                findings.append(
                    Finding(
                        "error",
                        "CONNECTOR_LABEL_TEXT_COLOR_MISMATCH",
                        f"{edge.id} label does not use the requested text color",
                        [edge.id],
                        True,
                    )
                )
            label_bounds.append(
                (
                    edge.id,
                    rectangle(
                        float(actual["labelX"]),
                        float(actual["labelY"]),
                        float(actual["labelWidth"]),
                        float(actual["labelHeight"]),
                    ),
                )
            )
        elif edge.label:
            connector_label_anchor_misaligned_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_LABEL_NOT_RENDERED",
                    f"{edge.id} live connector label is missing",
                    [edge.id],
                    False,
                )
            )

    text_line_overlap_count = 0
    text_line_low_clearance_count = 0
    text_shape_overlap_count = 0
    text_text_overlap_count = 0
    connector_crossing_count = 0
    connector_connector_crossing_count = 0
    external_text_bounds = [*caption_bounds, *label_bounds]
    for text_id, bounds in external_text_bounds:
        crossing_edges = sorted(
            {
                actual_edge_id
                for actual_edge_id, start, end in actual_segments
                if segment_intersects_rectangle(start, end, bounds)
            }
        )
        if crossing_edges:
            text_line_overlap_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_LINE_OVERLAP",
                    f"{text_id} live text intersects connector geometry",
                    [text_id, *crossing_edges],
                    True,
                )
            )
        low_clearance_edges = sorted(
            {
                actual_edge_id
                for actual_edge_id, start, end in actual_segments
                if actual_edge_id not in crossing_edges
                and segment_intersects_rectangle(start, end, expand_rectangle(bounds, 0.08))
            }
        )
        if low_clearance_edges:
            text_line_low_clearance_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_LINE_CLEARANCE_LOW",
                    f"{text_id} live text is closer than 0.08 in to connector geometry",
                    [text_id, *low_clearance_edges],
                    True,
                )
            )
        crossing_nodes = [
            node_id
            for node_id, node in nodes.items()
            if rectangles_overlap(bounds, node_rectangle(node, 0.02))
        ]
        if crossing_nodes:
            text_shape_overlap_count += 1
            findings.append(
                Finding(
                    "error",
                    "TEXT_SHAPE_OVERLAP",
                    f"{text_id} live text intersects a node frame",
                    [text_id, *crossing_nodes],
                    True,
                )
            )

    for index, (first_id, first_bounds) in enumerate(external_text_bounds):
        for second_id, second_bounds in external_text_bounds[index + 1 :]:
            if rectangles_overlap(first_bounds, second_bounds):
                text_text_overlap_count += 1
                findings.append(
                    Finding(
                        "error",
                        "TEXT_TEXT_OVERLAP",
                        f"{first_id} and {second_id} live labels overlap",
                        [first_id, second_id],
                        True,
                    )
                )

    for edge_id, start, end in actual_segments:
        expected = expected_edges.get(edge_id)
        if expected is None:
            continue
        crossed_nodes = [
            node_id
            for node_id, node in nodes.items()
            if node_id not in (expected.source, expected.target)
            and segment_intersects_rectangle(start, end, node_rectangle(node, -0.04))
        ]
        if crossed_nodes:
            connector_crossing_count += 1
            findings.append(
                Finding(
                    "error",
                    "CONNECTOR_CROSSES_NODE",
                    f"{edge_id} live path crosses an unrelated node",
                    [edge_id, *crossed_nodes],
                    True,
                )
            )

    crossing_pairs = set()
    for index, (first_id, first_start, first_end) in enumerate(actual_segments):
        for second_id, second_start, second_end in actual_segments[index + 1 :]:
            if first_id == second_id:
                continue
            if segment_crosses_segment(first_start, first_end, second_start, second_end):
                crossing_pairs.add(tuple(sorted((first_id, second_id))))
    for first_id, second_id in sorted(crossing_pairs):
        connector_connector_crossing_count += 1
        findings.append(
            Finding(
                "error",
                "CONNECTOR_CROSSING",
                f"{first_id} live path crosses {second_id} without a junction",
                [first_id, second_id],
                True,
            )
        )

    score = max(0, 100 - sum(15 if item.severity == "error" else 5 for item in findings))
    return {
        "score": score,
        "findingCount": len(findings),
        "findings": [asdict(item) for item in findings],
        "metrics": {
            "connectors": len(plan.edges),
            "inspectedConnectors": len(actual_edges),
            "maxEndpointAlignmentError": round(max_endpoint_error, 4),
            "misalignedEndpointCount": misaligned_endpoint_count,
            "routeStyleMismatchCount": route_style_mismatch_count,
            "missingConnectorCount": missing_connector_count,
            "notFullyGluedCount": not_fully_glued_count,
            "fullyGluedRatio": round(fully_glued_count / max(1, len(plan.edges)), 4),
            "connectorLabels": len(label_bounds),
            "textLineOverlapCount": text_line_overlap_count,
            "textLineLowClearanceCount": text_line_low_clearance_count,
            "textShapeOverlapCount": text_shape_overlap_count,
            "textTextOverlapCount": text_text_overlap_count,
            "connectorCrossingCount": connector_crossing_count,
            "connectorConnectorCrossingCount": connector_connector_crossing_count,
            "sourceEndpointIntrusionCount": source_endpoint_intrusion_count,
            "targetEndpointIntrusionCount": target_endpoint_intrusion_count,
            "arrowTerminalClearanceLowCount": arrow_terminal_clearance_low_count,
            "arrowheadNodeOverlapCount": arrowhead_node_overlap_count,
            "arrowGeometryUnverifiedCount": arrow_geometry_unverified_count,
            "nodeTextMisalignedCount": node_text_misaligned_count,
            "maxNodeTextCenterError": round(max_node_text_center_error, 4),
            "nodeTextBlockUtilizationMismatchCount": node_text_block_utilization_mismatch_count,
            "maxNodeTextBlockRatioError": round(max_node_text_block_ratio_error, 4),
            "nodeFontMismatchCount": node_font_mismatch_count,
            "nodeAsianFontMismatchCount": node_asian_font_mismatch_count,
            "nodeFontSizeMismatchCount": node_font_size_mismatch_count,
            "nodeFontStyleMismatchCount": node_font_style_mismatch_count,
            "nodeTextColorMismatchCount": node_text_color_mismatch_count,
            "nodeLineColorMismatchCount": node_line_color_mismatch_count,
            "nodeFillColorMismatchCount": node_fill_color_mismatch_count,
            "nodeLineWeightMismatchCount": node_line_weight_mismatch_count,
            "nodeCornerRadiusMismatchCount": node_corner_radius_mismatch_count,
            "connectorLabelAnchorMisalignedCount": connector_label_anchor_misaligned_count,
            "maxConnectorLabelAnchorError": round(max_connector_label_anchor_error, 4),
            "connectorLabelFontMismatchCount": connector_label_font_mismatch_count,
            "connectorLabelAsianFontMismatchCount": connector_label_asian_font_mismatch_count,
            "connectorLabelFontSizeMismatchCount": connector_label_font_size_mismatch_count,
            "connectorLabelFontStyleMismatchCount": connector_label_font_style_mismatch_count,
            "connectorLabelTextColorMismatchCount": connector_label_text_color_mismatch_count,
            "connectorLineColorMismatchCount": connector_line_color_mismatch_count,
            "connectorLineWeightMismatchCount": connector_line_weight_mismatch_count,
            "captions": len(caption_bounds),
            "captionMissingCount": caption_missing_count,
            "captionAnchorMisalignedCount": caption_anchor_misaligned_count,
            "maxCaptionAnchorError": round(max_caption_anchor_error, 4),
            "captionFontMismatchCount": caption_font_mismatch_count,
            "captionAsianFontMismatchCount": caption_asian_font_mismatch_count,
            "captionFontSizeMismatchCount": caption_font_size_mismatch_count,
            "captionFontStyleMismatchCount": caption_font_style_mismatch_count,
            "captionTextColorMismatchCount": caption_text_color_mismatch_count,
            "maxFontSizeErrorPt": round(max_font_size_error_pt, 4),
            "maxLineWeightErrorPt": round(max_line_weight_error_pt, 4),
            "maxNodeCornerRadiusErrorIn": round(
                max_node_corner_radius_error_in,
                4,
            ),
            "fontRoleSizeDriftCount": live_font_role_size_drift_count,
            "maxFontRoleSizeSpanPt": round(
                max_live_font_role_size_span,
                4,
            ),
            "sameAxisPeerGapGroupCount": len(live_same_axis_peer_gap_groups),
            "sameAxisPeerGapInconsistentCount": same_axis_peer_gap_inconsistent_count,
            "maxSameAxisPeerGapSpan": round(max_same_axis_peer_gap_span, 4),
        },
    }


def image_quality(path: str | Path) -> dict:
    image = Image.open(path).convert("RGB")
    width, height = image.size
    mean = sum(ImageStat.Stat(image).mean) / 3.0
    gray = image.convert("L")
    histogram = gray.histogram()
    total = max(1, width * height)
    white_ratio = sum(histogram[245:256]) / total
    dark_ratio = sum(histogram[:40]) / total
    box = gray.point(lambda pixel: 255 if pixel < 245 else 0).getbbox()
    content_fill = 0.0
    border_touch = False
    if box:
        x1, y1, x2, y2 = box
        content_fill = ((x2 - x1) * (y2 - y1)) / total
        border_touch = x1 <= 2 or y1 <= 2 or x2 >= width - 2 or y2 >= height - 2
    findings = []
    if white_ratio > 0.997 and dark_ratio < 0.0005:
        findings.append(
            {"severity": "error", "code": "NEAR_BLANK", "message": "Rendered image is nearly blank", "repairable": False}
        )
    if border_touch:
        findings.append(
            {
                "severity": "warning",
                "code": "CONTENT_TOUCHES_BORDER",
                "message": "Content touches image border",
                "repairable": True,
            }
        )
    if content_fill < 0.08:
        findings.append(
            {
                "severity": "warning",
                "code": "LOW_CONTENT_FILL",
                "message": "Diagram uses little canvas area",
                "repairable": True,
            }
        )
    score = max(
        0,
        100
        - 20 * sum(item["severity"] == "error" for item in findings)
        - 7 * sum(item["severity"] == "warning" for item in findings),
    )
    return {
        "score": score,
        "width": width,
        "height": height,
        "meanLuminance": round(mean, 2),
        "whiteRatio": round(white_ratio, 4),
        "darkRatio": round(dark_ratio, 4),
        "contentFill": round(content_fill, 4),
        "findings": findings,
    }
