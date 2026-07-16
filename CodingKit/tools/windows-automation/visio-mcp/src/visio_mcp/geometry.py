from __future__ import annotations

from math import hypot
from typing import Any, Iterable


PORT_FACTORS = {
    "left": (0.0, 0.5),
    "right": (1.0, 0.5),
    "top": (0.5, 1.0),
    "bottom": (0.5, 0.0),
}

PORT_NORMALS = {
    "left": (-1.0, 0.0),
    "right": (1.0, 0.0),
    "top": (0.0, 1.0),
    "bottom": (0.0, -1.0),
}

ARROWHEAD_LENGTH = 0.18
ARROWHEAD_HALF_WIDTH = 0.07
ENDPOINT_TAIL_CLEARANCE = 0.08
TEXT_LINE_CLEARANCE = 0.08
MAX_LABEL_NORMAL_EXTRA = 0.16
MAX_LABEL_TANGENTIAL_SHIFT = 0.20


def _value(item: Any, name: str) -> float:
    if isinstance(item, dict):
        return float(item[name])
    return float(getattr(item, name))


def _identity(item: Any) -> str:
    if isinstance(item, dict):
        return str(item["id"])
    return str(getattr(item, "id"))


def port_point(node: Any, port: str, position: float = 0.5) -> tuple[float, float]:
    factor_x, factor_y = PORT_FACTORS[port]
    if port in ("left", "right"):
        factor_y = position
    else:
        factor_x = position
    return (
        _value(node, "x") + (factor_x - 0.5) * _value(node, "width"),
        _value(node, "y") + (factor_y - 0.5) * _value(node, "height"),
    )


def resolve_ports(
    source: Any,
    target: Any,
    direction: str,
    source_port: str = "auto",
    target_port: str = "auto",
) -> tuple[str, str]:
    if _identity(source) == _identity(target):
        if source_port == "auto" and target_port == "auto":
            return "right", "top"
        if source_port == "auto":
            source_port = "right" if target_port != "right" else "bottom"
        if target_port == "auto":
            target_port = "top" if source_port != "top" else "left"
        return source_port, target_port

    delta_x = _value(target, "x") - _value(source, "x")
    delta_y = _value(target, "y") - _value(source, "y")
    horizontal = abs(delta_x) > abs(delta_y)
    if abs(abs(delta_x) - abs(delta_y)) < 1e-9:
        horizontal = direction in ("LR", "RL")

    if horizontal:
        inferred_source = "right" if delta_x >= 0 else "left"
        inferred_target = "left" if delta_x >= 0 else "right"
    else:
        inferred_source = "top" if delta_y >= 0 else "bottom"
        inferred_target = "bottom" if delta_y >= 0 else "top"
    return (
        inferred_source if source_port == "auto" else source_port,
        inferred_target if target_port == "auto" else target_port,
    )


def _deduplicate(points: Iterable[tuple[float, float]]) -> list[tuple[float, float]]:
    output: list[tuple[float, float]] = []
    for point in points:
        if not output or hypot(point[0] - output[-1][0], point[1] - output[-1][1]) > 1e-9:
            output.append(point)
    return output


def connector_points(
    source: Any,
    target: Any,
    source_port: str,
    target_port: str,
    routing: str,
    source_position: float = 0.5,
    target_position: float = 0.5,
    loop_gap: float = 0.6,
) -> list[tuple[float, float]]:
    start = port_point(source, source_port, source_position)
    end = port_point(target, target_port, target_position)
    if routing == "straight":
        return [start, end]

    if _identity(source) == _identity(target):
        right = _value(source, "x") + _value(source, "width") / 2 + loop_gap
        top = _value(source, "y") + _value(source, "height") / 2 + loop_gap
        return _deduplicate([start, (right, start[1]), (right, top), (end[0], top), end])

    horizontal_ports = {"left", "right"}
    vertical_ports = {"top", "bottom"}
    if source_port in horizontal_ports and target_port in horizontal_ports:
        if source_port == target_port:
            outer_x = (
                max(start[0], end[0]) + loop_gap
                if source_port == "right"
                else min(start[0], end[0]) - loop_gap
            )
            return _deduplicate([start, (outer_x, start[1]), (outer_x, end[1]), end])
        middle_x = (start[0] + end[0]) / 2
        return _deduplicate([start, (middle_x, start[1]), (middle_x, end[1]), end])
    if source_port in vertical_ports and target_port in vertical_ports:
        if source_port == target_port:
            outer_y = (
                max(start[1], end[1]) + loop_gap
                if source_port == "top"
                else min(start[1], end[1]) - loop_gap
            )
            return _deduplicate([start, (start[0], outer_y), (end[0], outer_y), end])
        middle_y = (start[1] + end[1]) / 2
        return _deduplicate([start, (start[0], middle_y), (end[0], middle_y), end])
    if source_port in horizontal_ports:
        return _deduplicate([start, (end[0], start[1]), end])
    return _deduplicate([start, (start[0], end[1]), end])


def segments(points: list[tuple[float, float]]) -> list[tuple[tuple[float, float], tuple[float, float]]]:
    return list(zip(points, points[1:]))


def simplify_polyline(
    points: list[tuple[float, float]],
    tolerance: float = 1e-6,
) -> list[tuple[float, float]]:
    simplified = _deduplicate(points)
    changed = True
    while changed and len(simplified) > 2:
        changed = False
        output = [simplified[0]]
        for current, following in zip(simplified[1:-1], simplified[2:]):
            previous = output[-1]
            first_x, first_y = current[0] - previous[0], current[1] - previous[1]
            second_x, second_y = following[0] - current[0], following[1] - current[1]
            cross = first_x * second_y - first_y * second_x
            dot = first_x * second_x + first_y * second_y
            if abs(cross) <= tolerance and dot >= -tolerance:
                changed = True
                continue
            output.append(current)
        output.append(simplified[-1])
        simplified = output
    return simplified


def endpoint_segment_state(
    points: list[tuple[float, float]],
    port: str,
    *,
    at_start: bool,
) -> tuple[float, float]:
    if len(points) < 2:
        return 0.0, 0.0
    anchor, adjacent = (points[0], points[1]) if at_start else (points[-1], points[-2])
    delta_x = adjacent[0] - anchor[0]
    delta_y = adjacent[1] - anchor[1]
    normal_x, normal_y = PORT_NORMALS[port]
    return hypot(delta_x, delta_y), delta_x * normal_x + delta_y * normal_y


def arrowhead_bounds(
    points: list[tuple[float, float]],
    length: float = ARROWHEAD_LENGTH,
    half_width: float = ARROWHEAD_HALF_WIDTH,
) -> tuple[float, float, float, float] | None:
    if len(points) < 2:
        return None
    previous, tip = points[-2], points[-1]
    delta_x = tip[0] - previous[0]
    delta_y = tip[1] - previous[1]
    segment_length = hypot(delta_x, delta_y)
    if segment_length <= 1e-9:
        return None
    unit_x, unit_y = delta_x / segment_length, delta_y / segment_length
    base_x = tip[0] - unit_x * length
    base_y = tip[1] - unit_y * length
    normal_x, normal_y = -unit_y, unit_x
    base_first = (
        base_x + normal_x * half_width,
        base_y + normal_y * half_width,
    )
    base_second = (
        base_x - normal_x * half_width,
        base_y - normal_y * half_width,
    )
    return (
        min(tip[0], base_first[0], base_second[0]),
        min(tip[1], base_first[1], base_second[1]),
        max(tip[0], base_first[0], base_second[0]),
        max(tip[1], base_first[1], base_second[1]),
    )


def polyline_length(points: list[tuple[float, float]]) -> float:
    return sum(
        hypot(end[0] - start[0], end[1] - start[1])
        for start, end in segments(points)
    )


def required_port_span(positions: Iterable[float]) -> float:
    ordered = sorted({round(float(position), 6) for position in positions})
    if not ordered:
        return 0.0
    edge_fraction = min(ordered[0], 1.0 - ordered[-1])
    edge_requirement = 0.15 / max(edge_fraction, 1e-6)
    if len(ordered) == 1:
        return min(edge_requirement, 0.6)
    adjacent_delta = min(
        second - first for first, second in zip(ordered, ordered[1:])
    )
    lane_requirement = 0.22 / max(adjacent_delta, 1e-6)
    return max(edge_requirement, lane_requirement)


def rectangle(
    center_x: float,
    center_y: float,
    width: float,
    height: float,
) -> tuple[float, float, float, float]:
    return (
        center_x - width / 2,
        center_y - height / 2,
        center_x + width / 2,
        center_y + height / 2,
    )


def node_rectangle(node: Any, padding: float = 0.0) -> tuple[float, float, float, float]:
    return (
        _value(node, "x") - _value(node, "width") / 2 - padding,
        _value(node, "y") - _value(node, "height") / 2 - padding,
        _value(node, "x") + _value(node, "width") / 2 + padding,
        _value(node, "y") + _value(node, "height") / 2 + padding,
    )


def expand_rectangle(
    bounds: tuple[float, float, float, float],
    padding: float,
) -> tuple[float, float, float, float]:
    return (
        bounds[0] - padding,
        bounds[1] - padding,
        bounds[2] + padding,
        bounds[3] + padding,
    )


def rectangles_overlap(
    first: tuple[float, float, float, float],
    second: tuple[float, float, float, float],
) -> bool:
    return not (
        first[2] <= second[0]
        or second[2] <= first[0]
        or first[3] <= second[1]
        or second[3] <= first[1]
    )


def segment_intersects_rectangle(
    start: tuple[float, float],
    end: tuple[float, float],
    bounds: tuple[float, float, float, float],
) -> bool:
    delta_x = end[0] - start[0]
    delta_y = end[1] - start[1]
    p = (-delta_x, delta_x, -delta_y, delta_y)
    q = (
        start[0] - bounds[0],
        bounds[2] - start[0],
        start[1] - bounds[1],
        bounds[3] - start[1],
    )
    lower, upper = 0.0, 1.0
    for coefficient, distance in zip(p, q):
        if abs(coefficient) < 1e-12:
            if distance < 0:
                return False
            continue
        ratio = distance / coefficient
        if coefficient < 0:
            lower = max(lower, ratio)
        else:
            upper = min(upper, ratio)
        if lower > upper:
            return False
    return True


def segment_crosses_segment(
    first_start: tuple[float, float],
    first_end: tuple[float, float],
    second_start: tuple[float, float],
    second_end: tuple[float, float],
    tolerance: float = 1e-6,
) -> bool:
    def cross(
        first: tuple[float, float],
        second: tuple[float, float],
        third: tuple[float, float],
    ) -> float:
        return (second[0] - first[0]) * (third[1] - first[1]) - (
            second[1] - first[1]
        ) * (third[0] - first[0])

    def close(first: tuple[float, float], second: tuple[float, float]) -> bool:
        return hypot(first[0] - second[0], first[1] - second[1]) <= tolerance

    first_cross_start = cross(first_start, first_end, second_start)
    first_cross_end = cross(first_start, first_end, second_end)
    second_cross_start = cross(second_start, second_end, first_start)
    second_cross_end = cross(second_start, second_end, first_end)
    collinear = all(
        abs(value) <= tolerance
        for value in (first_cross_start, first_cross_end, second_cross_start, second_cross_end)
    )
    if collinear:
        use_x = abs(first_end[0] - first_start[0]) >= abs(first_end[1] - first_start[1])
        first_values = (first_start[0], first_end[0]) if use_x else (first_start[1], first_end[1])
        second_values = (second_start[0], second_end[0]) if use_x else (second_start[1], second_end[1])
        overlap = min(max(first_values), max(second_values)) - max(
            min(first_values),
            min(second_values),
        )
        return overlap > tolerance

    intersects = (
        first_cross_start * first_cross_end <= tolerance
        and second_cross_start * second_cross_end <= tolerance
    )
    if not intersects:
        return False
    return not any(
        close(first, second)
        for first in (first_start, first_end)
        for second in (second_start, second_end)
    )


def estimate_label_size(
    text: str,
    font_size_pt: float = 10.0,
    width_safety: float = 1.0,
) -> tuple[float, float]:
    lines = text.splitlines() or [""]
    line_units = [
        sum(2.0 if ord(character) > 255 else 1.0 for character in line)
        for line in lines
    ]
    scale = max(0.6, float(font_size_pt) / 10.0)
    width = min(
        6.0,
        max(
            0.28 * scale,
            max(line_units, default=0.0) * 0.065 * scale + 0.14,
        )
        * max(1.0, width_safety),
    )
    height = max(0.28 * scale, len(lines) * 0.22 * scale + 0.06)
    return width, height


def estimate_node_text_size(
    text: str,
    font_size_pt: float = 10.0,
) -> tuple[float, float]:
    lines = text.splitlines() or [""]
    line_units = [
        sum(2.0 if ord(character) > 255 else 1.0 for character in line)
        for line in lines
    ]
    scale = max(0.6, float(font_size_pt) / 10.0)
    width = max(
        0.18 * scale,
        max(line_units, default=0.0) * 0.065 * scale + 0.08,
    )
    height = max(0.18 * scale, len(lines) * 0.2 * scale + 0.04)
    return width, height


def caption_center(
    node: Any,
    side: str,
    position: float,
    offset: float,
    width: float,
    height: float,
) -> tuple[float, float]:
    left = _value(node, "x") - _value(node, "width") / 2
    right = _value(node, "x") + _value(node, "width") / 2
    bottom = _value(node, "y") - _value(node, "height") / 2
    top = _value(node, "y") + _value(node, "height") / 2
    if side == "top":
        return left + position * _value(node, "width"), top + offset + height / 2
    if side == "bottom":
        return left + position * _value(node, "width"), bottom - offset - height / 2
    if side == "left":
        return left - offset - width / 2, bottom + position * _value(node, "height")
    return right + offset + width / 2, bottom + position * _value(node, "height")


def place_caption(
    node: Any,
    width: float,
    height: float,
    preferred_side: str,
    position: float,
    offset: float,
    nodes: list[Any],
    route_segments: list[tuple[tuple[float, float], tuple[float, float]]],
    occupied: list[tuple[float, float, float, float]],
    page_width: float,
    page_height: float,
    margin: float,
) -> tuple[float, float, str, bool]:
    if preferred_side == "auto":
        tall = _value(node, "height") >= _value(node, "width") * 1.35
        sides = ["right", "left", "top", "bottom"] if tall else ["top", "bottom", "right", "left"]
    else:
        sides = [preferred_side]
    owner_id = _identity(node)
    for side in sides:
        center_x, center_y = caption_center(
            node,
            side,
            position,
            offset,
            width,
            height,
        )
        bounds = rectangle(center_x, center_y, width, height)
        if (
            bounds[0] < margin
            or bounds[1] < margin
            or bounds[2] > page_width - margin
            or bounds[3] > page_height - margin
        ):
            continue
        if any(
            _identity(other) != owner_id
            and rectangles_overlap(bounds, node_rectangle(other, 0.02))
            for other in nodes
        ):
            continue
        protected = expand_rectangle(bounds, TEXT_LINE_CLEARANCE)
        if any(
            segment_intersects_rectangle(start, end, protected)
            for start, end in route_segments
        ):
            continue
        if any(rectangles_overlap(bounds, other) for other in occupied):
            continue
        return center_x, center_y, side, True

    fallback_side = sides[0]
    center_x, center_y = caption_center(
        node,
        fallback_side,
        position,
        offset,
        width,
        height,
    )
    return center_x, center_y, fallback_side, False


def _label_segment_candidates(
    route_points: list[tuple[float, float]],
    preferred_side: str,
    requested_position: float,
) -> list[tuple[tuple[float, float], tuple[float, float], float, float, float]]:
    raw_segments = segments(route_points)
    lengths = [
        hypot(end[0] - start[0], end[1] - start[1])
        for start, end in raw_segments
    ]
    total = sum(lengths)
    if total <= 1e-9:
        return []
    target_distance = max(0.0, min(1.0, requested_position)) * total
    output = []
    walked = 0.0
    for (start, end), length in zip(raw_segments, lengths):
        if length <= 1e-9:
            continue
        horizontal = abs(end[0] - start[0]) >= abs(end[1] - start[1])
        if preferred_side in ("above", "below") and not horizontal:
            walked += length
            continue
        if preferred_side in ("left", "right") and horizontal:
            walked += length
            continue
        distance_to_segment = max(walked - target_distance, target_distance - (walked + length), 0.0)
        output.append((start, end, walked, length, distance_to_segment))
        walked += length
    return sorted(output, key=lambda item: (item[4], -item[3]))


def place_label(
    route_points: list[tuple[float, float]],
    width: float,
    height: float,
    preferred_side: str,
    requested_offset: float,
    requested_position: float,
    nodes: list[Any],
    route_segments: list[tuple[tuple[float, float], tuple[float, float]]],
    occupied: list[tuple[float, float, float, float]],
    page_width: float,
    page_height: float,
    margin: float,
) -> tuple[float, float, str, float, float, bool, float, float]:
    route_points = simplify_polyline(route_points)
    total_length = polyline_length(route_points)
    segment_candidates = _label_segment_candidates(
        route_points,
        preferred_side,
        requested_position,
    )
    if not segment_candidates:
        anchor_x, anchor_y = route_points[0] if route_points else (0.0, 0.0)
        return (
            anchor_x,
            anchor_y,
            preferred_side if preferred_side != "auto" else "above",
            requested_offset,
            requested_position,
            False,
            anchor_x,
            anchor_y,
        )

    for start, end, walked, segment_length, _distance in segment_candidates:
        horizontal = abs(end[0] - start[0]) >= abs(end[1] - start[1])
        if preferred_side == "auto":
            sides = ["above", "below"] if horizontal else ["right", "left"]
        else:
            sides = [preferred_side]
        target_distance = max(0.0, min(1.0, requested_position)) * total_length
        base_position = max(0.15, min(0.85, (target_distance - walked) / segment_length))
        tangential_budget = min(MAX_LABEL_TANGENTIAL_SHIFT, segment_length * 0.15)
        tangential_shifts = (
            0.0,
            -tangential_budget / 2,
            tangential_budget / 2,
            -tangential_budget,
            tangential_budget,
        )
        unit_x = (end[0] - start[0]) / segment_length
        unit_y = (end[1] - start[1]) / segment_length
        for extra in (0.0, 0.08, MAX_LABEL_NORMAL_EXTRA):
            for side in sides:
                half_extent = height / 2 if side in ("above", "below") else width / 2
                actual_offset = max(
                    requested_offset,
                    half_extent + TEXT_LINE_CLEARANCE + 0.01,
                ) + extra
                for tangential_shift in tangential_shifts:
                    local_position = max(
                        0.1,
                        min(
                            0.9,
                            base_position + tangential_shift / segment_length,
                        ),
                    )
                    anchor_x = start[0] + (end[0] - start[0]) * local_position
                    anchor_y = start[1] + (end[1] - start[1]) * local_position
                    delta = {
                        "above": (0.0, actual_offset),
                        "below": (0.0, -actual_offset),
                        "left": (-actual_offset, 0.0),
                        "right": (actual_offset, 0.0),
                    }[side]
                    center_x, center_y = anchor_x + delta[0], anchor_y + delta[1]
                    bounds = rectangle(center_x, center_y, width, height)
                    if (
                        bounds[0] < margin
                        or bounds[1] < margin
                        or bounds[2] > page_width - margin
                        or bounds[3] > page_height - margin
                    ):
                        continue
                    if any(
                        rectangles_overlap(bounds, node_rectangle(node, 0.04))
                        for node in nodes
                    ):
                        continue
                    protected = expand_rectangle(bounds, TEXT_LINE_CLEARANCE)
                    if any(
                        segment_intersects_rectangle(segment_start, segment_end, protected)
                        for segment_start, segment_end in route_segments
                    ):
                        continue
                    if any(rectangles_overlap(bounds, other) for other in occupied):
                        continue
                    actual_position = (
                        walked + local_position * segment_length
                    ) / total_length
                    return (
                        center_x,
                        center_y,
                        side,
                        actual_offset,
                        actual_position,
                        True,
                        anchor_x,
                        anchor_y,
                    )

    start, end, walked, segment_length, _distance = segment_candidates[0]
    horizontal = abs(end[0] - start[0]) >= abs(end[1] - start[1])
    side = (
        preferred_side
        if preferred_side != "auto"
        else ("above" if horizontal else "right")
    )
    target_distance = max(0.0, min(1.0, requested_position)) * total_length
    local_position = max(0.15, min(0.85, (target_distance - walked) / segment_length))
    anchor_x = start[0] + (end[0] - start[0]) * local_position
    anchor_y = start[1] + (end[1] - start[1]) * local_position
    half_extent = height / 2 if side in ("above", "below") else width / 2
    actual_offset = max(
        requested_offset,
        half_extent + TEXT_LINE_CLEARANCE + 0.01,
    )
    delta = {
        "above": (0.0, actual_offset),
        "below": (0.0, -actual_offset),
        "left": (-actual_offset, 0.0),
        "right": (actual_offset, 0.0),
    }[side]
    actual_position = (walked + local_position * segment_length) / total_length
    return (
        anchor_x + delta[0],
        anchor_y + delta[1],
        side,
        actual_offset,
        actual_position,
        False,
        anchor_x,
        anchor_y,
    )
