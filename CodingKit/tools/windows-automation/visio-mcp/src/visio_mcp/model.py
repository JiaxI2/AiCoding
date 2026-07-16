from __future__ import annotations

from dataclasses import asdict, dataclass
import json
from pathlib import Path
from typing import Any


@dataclass
class NodeBox:
    id: str
    text: str
    type: str
    x: float
    y: float
    width: float
    height: float
    layer: int = 0
    order: int = 0
    style: str | None = None
    size_class: str | None = None
    font_family: str = "SimSun"
    asian_font_family: str = "SimSun"
    font_role: str = "body"
    font_size_pt: float = 10.0
    font_weight: str = "regular"
    font_style: str = "normal"
    text_color: str = "#000000"
    line_color: str = "#000000"
    fill_color: str = "#FFFFFF"
    line_weight_pt: float = 0.75
    corner_radius_in: float = 0.12
    text_block_width_ratio: float = 0.8
    text_block_height_ratio: float = 0.8
    caption: str = ""
    caption_font_size_pt: float = 10.0
    caption_font_weight: str = "regular"
    caption_font_style: str = "normal"
    caption_side: str = "auto"
    caption_position: float = 0.5
    caption_offset: float = 0.1
    caption_x: float | None = None
    caption_y: float | None = None
    caption_width: float = 0.0
    caption_height: float = 0.0
    caption_resolved_side: str | None = None
    caption_anchor_resolved: bool = True
    data: dict[str, Any] | None = None


@dataclass
class Edge:
    id: str
    source: str
    target: str
    label: str = ""
    type: str = "directed"
    style: str | None = None
    source_port: str = "right"
    target_port: str = "left"
    source_port_position: float = 0.5
    target_port_position: float = 0.5
    routing: str = "orthogonal"
    route_points: list[tuple[float, float]] | None = None
    label_side: str = "auto"
    label_offset: float = 0.22
    label_position: float = 0.5
    label_actual_position: float = 0.5
    label_actual_offset: float = 0.22
    label_anchor_x: float | None = None
    label_anchor_y: float | None = None
    label_anchor_resolved: bool = True
    label_x: float | None = None
    label_y: float | None = None
    label_width: float = 0.0
    label_height: float = 0.0
    font_family: str = "SimSun"
    asian_font_family: str = "SimSun"
    font_role: str = "edgeLabel"
    font_size_pt: float = 10.0
    font_weight: str = "regular"
    font_style: str = "normal"
    text_color: str = "#000000"
    line_color: str = "#000000"
    line_weight_pt: float = 0.75


@dataclass
class DiagramPlan:
    document: dict[str, Any]
    nodes: list[NodeBox]
    edges: list[Edge]
    groups: list[dict[str, Any]]
    metadata: dict[str, Any]

    def to_dict(self) -> dict[str, Any]:
        return {
            "document": self.document,
            "nodes": [asdict(item) for item in self.nodes],
            "edges": [asdict(item) for item in self.edges],
            "groups": self.groups,
            "metadata": self.metadata,
        }


def load_json(path: str | Path) -> dict[str, Any]:
    return json.loads(Path(path).read_text(encoding="utf-8"))
