from __future__ import annotations

from collections import defaultdict, deque

from .geometry import (
    connector_points,
    estimate_label_size,
    place_caption,
    place_label,
    rectangle,
    resolve_ports,
    segments,
)
from .model import DiagramPlan, Edge, NodeBox
from .styling import (
    normalize_color,
    resolve_document_style,
    resolve_line_weight,
    resolve_role_font_size,
    resolve_style_token,
    resolve_text_role,
)


DEFAULT_SIZE = {
    "actor": (2.2, 0.9),
    "process": (2.5, 1.0),
    "component": (2.7, 1.1),
    "database": (2.5, 1.2),
    "decision": (2.0, 1.3),
    "junction": (0.35, 0.35),
    "note": (2.4, 1.1),
    "group": (3.0, 1.5),
}


def _infer_layers(nodes: list[dict], edges: list[dict]) -> dict[str, int]:
    explicit = {node["id"]: node.get("layer") for node in nodes if "layer" in node}
    if len(explicit) == len(nodes):
        return {key: int(value) for key, value in explicit.items()}

    incoming = defaultdict(int)
    outgoing = defaultdict(list)
    ids = [node["id"] for node in nodes]
    for edge in edges:
        outgoing[edge["from"]].append(edge["to"])
        incoming[edge["to"]] += 1
    queue = deque([node_id for node_id in ids if incoming[node_id] == 0])
    layers = {node_id: 0 for node_id in queue}
    while queue:
        current = queue.popleft()
        for target in outgoing[current]:
            layers[target] = max(layers.get(target, 0), layers[current] + 1)
            incoming[target] -= 1
            if incoming[target] == 0:
                queue.append(target)
    for node_id in ids:
        layers.setdefault(node_id, int(explicit.get(node_id, 0) or 0))
    return layers


def _alternating_offset(position: int, step: float) -> float:
    if position == 0:
        return 0.0
    distance = ((position + 1) // 2) * step
    return distance if position % 2 == 1 else -distance


def plan_diagram(data: dict) -> DiagramPlan:
    document = data["document"]
    page = document.get("page", {})
    page_width = float(page.get("width", 16))
    page_height = float(page.get("height", 9))
    margin = float(page.get("margin", 0.5))
    layout = document.get("layout", {})
    engine = layout.get("engine", "layered")
    direction = layout.get("direction", "LR")
    node_gap = float(layout.get("nodeGap", 0.6))
    layer_gap = float(layout.get("layerGap", 1.2))
    uniform_node_size = bool(layout.get("uniformNodeSize", True))
    uniform_width = float(layout.get("nodeWidth", 2.4))
    uniform_height = float(layout.get("nodeHeight", 1.0))
    style_profile = resolve_document_style(document)
    typography = style_profile["typography"]
    appearance = style_profile["appearance"]
    default_text_block_width_ratio = float(
        typography.get("nodeTextBlockWidthRatio", 0.8)
    )
    default_text_block_height_ratio = float(
        typography.get("nodeTextBlockHeightRatio", 0.8)
    )
    caption_role_style = resolve_text_role(style_profile, "caption")
    caption_font_size_pt = resolve_role_font_size(
        style_profile,
        "caption",
        page_width,
        page_height,
    )

    layers = _infer_layers(data["nodes"], data["edges"])
    buckets = defaultdict(list)
    for index, raw in enumerate(data["nodes"]):
        buckets[layers[raw["id"]]].append((raw.get("order", index), raw))
    for layer in buckets:
        buckets[layer].sort(key=lambda item: item[0])

    layer_keys = sorted(buckets)
    dimensions = {}
    for raw in data["nodes"]:
        node_type = raw.get("type", "process")
        default_width, default_height = DEFAULT_SIZE.get(node_type, (2.5, 1.0))
        dimensions[raw["id"]] = (
            uniform_width if uniform_node_size else float(raw.get("width", default_width)),
            uniform_height if uniform_node_size else float(raw.get("height", default_height)),
        )

    if engine != "manual" and layer_keys:
        primary_available = (page_width if direction in ("LR", "RL") else page_height) - 2 * margin
        primary_sizes = []
        for layer in layer_keys:
            dimension_index = 0 if direction in ("LR", "RL") else 1
            primary_sizes.append(max(dimensions[raw["id"]][dimension_index] for _, raw in buckets[layer]))
        effective_gap = layer_gap
        required = sum(primary_sizes) + max(0, len(primary_sizes) - 1) * effective_gap
        if required > primary_available and len(primary_sizes) > 1:
            effective_gap = min(layer_gap, 0.35)
            if uniform_node_size:
                fitted_size = (primary_available - (len(primary_sizes) - 1) * effective_gap) / len(primary_sizes)
                fitted_size = max(1.2 if direction in ("LR", "RL") else 0.7, fitted_size)
                if direction in ("LR", "RL"):
                    uniform_width = min(uniform_width, fitted_size)
                else:
                    uniform_height = min(uniform_height, fitted_size)
                dimensions = {
                    node_id: (
                        uniform_width,
                        uniform_height,
                    )
                    for node_id in dimensions
                }
                primary_sizes = [uniform_width if direction in ("LR", "RL") else uniform_height] * len(layer_keys)
        layer_centers = {}
        cursor = margin
        for layer, primary_size in zip(layer_keys, primary_sizes):
            layer_centers[layer] = cursor + primary_size / 2
            cursor += primary_size + effective_gap
    else:
        layer_centers = {}

    nodes: list[NodeBox] = []
    for layer, entries in sorted(buckets.items()):
        max_layer_width = max(dimensions[raw["id"]][0] for _, raw in entries)
        max_layer_height = max(dimensions[raw["id"]][1] for _, raw in entries)
        for position, (_, raw) in enumerate(entries):
            node_type = raw.get("type", "process")
            width, height = dimensions[raw["id"]]
            default_font_role = {
                "group": "groupTitle",
                "junction": "operator",
                "note": "note",
            }.get(node_type, "body")
            font_role = str(raw.get("fontRole", default_font_role))
            font_role_style = resolve_text_role(style_profile, font_role)
            primary_font_family = (
                typography["mathFontFamily"]
                if font_role_style["fontFamilyRole"] == "math"
                else typography["latinFontFamily"]
            )
            legacy_font_family = raw.get("fontFamily")
            font_family = str(
                raw.get(
                    "latinFontFamily",
                    legacy_font_family or primary_font_family,
                )
            )
            asian_font_family = str(
                raw.get(
                    "asianFontFamily",
                    legacy_font_family or typography["asianFontFamily"],
                )
            )
            node_style = resolve_style_token(
                style_profile,
                "nodeStyles",
                raw.get("style"),
            )
            text_color = normalize_color(
                str(
                    raw.get(
                        "textColor",
                        node_style.get("textColor", appearance["textColor"]),
                    )
                )
            )
            line_color = normalize_color(
                str(
                    raw.get(
                        "lineColor",
                        node_style.get("lineColor", appearance["nodeLineColor"]),
                    )
                )
            )
            fill_color = normalize_color(
                str(
                    raw.get(
                        "fillColor",
                        node_style.get("fillColor", appearance["nodeFillColor"]),
                    )
                )
            )
            default_line_weight_pt = resolve_line_weight(
                style_profile,
                float(
                    node_style.get(
                        "lineWeightPt",
                        appearance["nodeLineWeightPt"],
                    )
                ),
                page_width,
                page_height,
            )
            line_weight_pt = float(
                raw.get("lineWeightPt", default_line_weight_pt)
            )
            corner_radius_in = float(
                raw.get(
                    "cornerRadiusIn",
                    node_style.get(
                        "cornerRadiusIn",
                        appearance["nodeCornerRadiusIn"],
                    ),
                )
            )
            text_block_width_ratio = float(
                raw.get("textBlockWidthRatio", default_text_block_width_ratio)
            )
            text_block_height_ratio = float(
                raw.get("textBlockHeightRatio", default_text_block_height_ratio)
            )
            if node_type == "decision":
                if "textBlockWidthRatio" not in raw:
                    text_block_width_ratio = min(text_block_width_ratio, 0.7)
                if "textBlockHeightRatio" not in raw:
                    text_block_height_ratio = min(text_block_height_ratio, 0.7)
            if engine == "manual" and "x" in raw and "y" in raw:
                x, y = float(raw["x"]), float(raw["y"])
            elif direction in ("LR", "RL"):
                x = layer_centers[layer]
                y = page_height / 2 + _alternating_offset(position, max_layer_height + node_gap)
                if direction == "RL":
                    x = page_width - x
            else:
                x = page_width / 2 + _alternating_offset(position, max_layer_width + node_gap)
                y = page_height - layer_centers[layer]
                if direction == "BT":
                    y = page_height - y
            nodes.append(
                NodeBox(
                    id=raw["id"],
                    text=raw["text"],
                    type=node_type,
                    x=x,
                    y=y,
                    width=width,
                    height=height,
                    layer=layer,
                    order=int(raw.get("order", position)),
                    style=raw.get("style"),
                    size_class=raw.get("sizeClass"),
                    font_family=font_family,
                    asian_font_family=asian_font_family,
                    font_role=font_role,
                    font_size_pt=float(
                        raw.get(
                            "fontSizePt",
                            resolve_role_font_size(
                                style_profile,
                                font_role,
                                page_width,
                                page_height,
                            ),
                        )
                    ),
                    font_weight=str(
                        raw.get("fontWeight", font_role_style["fontWeight"])
                    ),
                    font_style=str(
                        raw.get("fontStyle", font_role_style["fontStyle"])
                    ),
                    text_color=text_color,
                    line_color=line_color,
                    fill_color=fill_color,
                    line_weight_pt=line_weight_pt,
                    corner_radius_in=corner_radius_in,
                    text_block_width_ratio=text_block_width_ratio,
                    text_block_height_ratio=text_block_height_ratio,
                    caption=str(raw.get("caption", "")),
                    caption_font_size_pt=float(
                        raw.get("captionFontSizePt", caption_font_size_pt)
                    ),
                    caption_font_weight=str(
                        raw.get(
                            "captionFontWeight",
                            caption_role_style["fontWeight"],
                        )
                    ),
                    caption_font_style=str(
                        raw.get(
                            "captionFontStyle",
                            caption_role_style["fontStyle"],
                        )
                    ),
                    caption_side=raw.get("captionSide", "auto"),
                    caption_position=float(raw.get("captionPosition", 0.5)),
                    caption_offset=float(raw.get("captionOffset", 0.1)),
                    data=raw.get("data"),
                )
            )

    nodes_by_id = {node.id: node for node in nodes}
    edges = []
    for raw in data["edges"]:
        source = nodes_by_id[raw["from"]]
        target = nodes_by_id[raw["to"]]
        font_role = str(raw.get("fontRole", "edgeLabel"))
        font_role_style = resolve_text_role(style_profile, font_role)
        primary_font_family = (
            typography["mathFontFamily"]
            if font_role_style["fontFamilyRole"] == "math"
            else typography["latinFontFamily"]
        )
        legacy_font_family = raw.get("fontFamily")
        edge_style = resolve_style_token(
            style_profile,
            "edgeStyles",
            raw.get("style"),
        )
        source_port, target_port = resolve_ports(
            source,
            target,
            direction,
            raw.get("sourcePort", "auto"),
            raw.get("targetPort", "auto"),
        )
        routing = raw.get("routing", "orthogonal")
        edges.append(
            Edge(
                id=raw["id"],
                source=raw["from"],
                target=raw["to"],
                label=raw.get("label", ""),
                type=raw.get("type", "directed"),
                style=raw.get("style"),
                source_port=source_port,
                target_port=target_port,
                source_port_position=float(raw.get("sourcePortPosition", 0.5)),
                target_port_position=float(raw.get("targetPortPosition", 0.5)),
                routing=routing,
                route_points=connector_points(
                    source,
                    target,
                    source_port,
                    target_port,
                    routing,
                    float(raw.get("sourcePortPosition", 0.5)),
                    float(raw.get("targetPortPosition", 0.5)),
                ),
                label_side=raw.get("labelSide", "auto"),
                label_offset=float(raw.get("labelOffset", 0.22)),
                label_position=float(raw.get("labelPosition", 0.5)),
                font_family=str(
                    raw.get(
                        "latinFontFamily",
                        legacy_font_family or primary_font_family,
                    )
                ),
                asian_font_family=str(
                    raw.get(
                        "asianFontFamily",
                        legacy_font_family or typography["asianFontFamily"],
                    )
                ),
                font_role=font_role,
                font_size_pt=float(
                    raw.get(
                        "fontSizePt",
                        resolve_role_font_size(
                            style_profile,
                            font_role,
                            page_width,
                            page_height,
                        ),
                    )
                ),
                font_weight=str(
                    raw.get("fontWeight", font_role_style["fontWeight"])
                ),
                font_style=str(
                    raw.get("fontStyle", font_role_style["fontStyle"])
                ),
                text_color=normalize_color(
                    str(
                        raw.get(
                            "textColor",
                            edge_style.get("textColor", appearance["textColor"]),
                        )
                    )
                ),
                line_color=normalize_color(
                    str(
                        raw.get(
                            "lineColor",
                            edge_style.get(
                                "lineColor",
                                appearance["connectorLineColor"],
                            ),
                        )
                    )
                ),
                line_weight_pt=float(
                    raw.get(
                        "lineWeightPt",
                        resolve_line_weight(
                            style_profile,
                            float(
                                edge_style.get(
                                    "lineWeightPt",
                                    appearance["connectorLineWeightPt"],
                                )
                            ),
                            page_width,
                            page_height,
                        ),
                    )
                ),
            )
        )

    route_segments = [
        segment
        for edge in edges
        for segment in segments(edge.route_points or [])
    ]
    occupied_text = []
    for node in nodes:
        if not node.caption:
            continue
        node.caption_width, node.caption_height = estimate_label_size(
            node.caption,
            node.caption_font_size_pt,
            width_safety=1.15,
        )
        (
            node.caption_x,
            node.caption_y,
            node.caption_resolved_side,
            node.caption_anchor_resolved,
        ) = place_caption(
            node,
            node.caption_width,
            node.caption_height,
            node.caption_side,
            node.caption_position,
            node.caption_offset,
            nodes,
            route_segments,
            occupied_text,
            page_width,
            page_height,
            margin,
        )
        occupied_text.append(
            rectangle(
                node.caption_x,
                node.caption_y,
                node.caption_width,
                node.caption_height,
            )
        )
    for edge in edges:
        if not edge.label:
            continue
        edge.label_width, edge.label_height = estimate_label_size(
            edge.label,
            edge.font_size_pt,
        )
        (
            edge.label_x,
            edge.label_y,
            edge.label_side,
            edge.label_actual_offset,
            edge.label_actual_position,
            edge.label_anchor_resolved,
            edge.label_anchor_x,
            edge.label_anchor_y,
        ) = place_label(
            edge.route_points or [],
            edge.label_width,
            edge.label_height,
            edge.label_side,
            edge.label_offset,
            edge.label_position,
            nodes,
            route_segments,
            occupied_text,
            page_width,
            page_height,
            margin,
        )
        occupied_text.append(
            rectangle(edge.label_x, edge.label_y, edge.label_width, edge.label_height)
        )
    return DiagramPlan(
        document,
        nodes,
        edges,
        data.get("groups", []),
        data.get("metadata", {}),
    )
