from __future__ import annotations

import json
from math import hypot
from pathlib import Path
from uuid import uuid4

from PIL import Image, ImageDraw, ImageFont

from ..geometry import port_point
from ..model import DiagramPlan
from ..styling import char_style_code, resolve_document_style
from .base import Renderer


class MockRenderer(Renderer):
    def __init__(self) -> None:
        self.sessions: dict[str, dict] = {}

    def doctor(self) -> dict:
        return {"available": True, "renderer": "mock", "visibleEditing": False}

    @staticmethod
    def _font(
        scale: int,
        font_family: str,
        font_size_pt: float,
        font_weight: str = "regular",
        font_style: str = "normal",
    ):
        size = max(8, round(font_size_pt * scale / 72))
        key = (
            font_family.casefold(),
            font_weight,
            font_style,
        )
        font_files = {
            ("simsun", "regular", "normal"): "simsun.ttc",
            ("simsun", "bold", "normal"): "simsun.ttc",
            ("microsoft yahei", "regular", "normal"): "msyh.ttc",
            ("microsoft yahei", "bold", "normal"): "msyhbd.ttc",
            ("times new roman", "regular", "normal"): "times.ttf",
            ("times new roman", "bold", "normal"): "timesbd.ttf",
            ("times new roman", "regular", "italic"): "timesi.ttf",
            ("times new roman", "bold", "italic"): "timesbi.ttf",
            ("arial", "regular", "normal"): "arial.ttf",
            ("arial", "bold", "normal"): "arialbd.ttf",
            ("arial", "regular", "italic"): "ariali.ttf",
            ("arial", "bold", "italic"): "arialbi.ttf",
            ("cambria math", "regular", "normal"): "cambria.ttc",
        }
        path = Path(r"C:\Windows\Fonts") / font_files.get(
            key,
            font_files.get(
                (font_family.casefold(), "regular", "normal"),
                "msyh.ttc",
            ),
        )
        if path.exists():
            return ImageFont.truetype(str(path), size)
        return ImageFont.load_default()

    def render(self, plan: DiagramPlan, output: Path, visible: bool = False) -> dict:
        session_id = str(uuid4())
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(plan.to_dict(), ensure_ascii=False, indent=2), encoding="utf-8")
        self.sessions[session_id] = {"plan": plan, "output": str(output)}
        return {"sessionId": session_id, "output": str(output), "renderer": "mock", "visible": False}

    def snapshot(self, session_id: str, output: Path) -> dict:
        plan = self.sessions[session_id]["plan"]
        page = plan.document.get("page", {})
        page_width, page_height = float(page.get("width", 16)), float(page.get("height", 9))
        scale = 100
        appearance = resolve_document_style(plan.document)["appearance"]
        image = Image.new(
            "RGB",
            (int(page_width * scale), int(page_height * scale)),
            appearance["pageBackgroundColor"],
        )
        draw = ImageDraw.Draw(image)
        by_id = {node.id: node for node in plan.nodes}
        for edge in plan.edges:
            points = [
                (point[0] * scale, (page_height - point[1]) * scale)
                for point in (edge.route_points or [])
            ]
            if len(points) >= 2:
                draw.line(
                    points,
                    fill=edge.line_color,
                    width=max(1, round(edge.line_weight_pt * scale / 72)),
                    joint="curve",
                )
        for node in plan.nodes:
            x1 = (node.x - node.width / 2) * scale
            y1 = (page_height - (node.y + node.height / 2)) * scale
            x2 = (node.x + node.width / 2) * scale
            y2 = (page_height - (node.y - node.height / 2)) * scale
            line_width = max(1, round(node.line_weight_pt * scale / 72))
            if node.type == "decision":
                draw.polygon(
                    (
                        ((x1 + x2) / 2, y1),
                        (x2, (y1 + y2) / 2),
                        ((x1 + x2) / 2, y2),
                        (x1, (y1 + y2) / 2),
                    ),
                    fill=node.fill_color,
                    outline=node.line_color,
                )
            elif node.type == "junction":
                draw.ellipse(
                    (x1, y1, x2, y2),
                    fill=node.fill_color,
                    outline=node.line_color,
                    width=line_width,
                )
            elif node.type == "note":
                pass
            else:
                radius = max(0, round(node.corner_radius_in * scale))
                if radius:
                    draw.rounded_rectangle(
                        (x1, y1, x2, y2),
                        radius=radius,
                        fill=node.fill_color,
                        outline=node.line_color,
                        width=line_width,
                    )
                else:
                    draw.rectangle(
                        (x1, y1, x2, y2),
                        fill=node.fill_color,
                        outline=node.line_color,
                        width=line_width,
                    )
            font = self._font(
                scale,
                node.font_family,
                node.font_size_pt,
                node.font_weight,
                node.font_style,
            )
            text_box = draw.multiline_textbbox(
                (0, 0),
                node.text,
                font=font,
                spacing=3,
                align="center",
            )
            text_width = text_box[2] - text_box[0]
            text_height = text_box[3] - text_box[1]
            draw.multiline_text(
                (
                    (x1 + x2) / 2 - text_width / 2,
                    (y1 + y2) / 2 - text_height / 2,
                ),
                node.text,
                fill=node.text_color,
                font=font,
                spacing=3,
                align="center",
            )
        for node in plan.nodes:
            if not node.caption or node.caption_x is None or node.caption_y is None:
                continue
            center_x = node.caption_x * scale
            center_y = (page_height - node.caption_y) * scale
            font = self._font(
                scale,
                node.font_family,
                node.caption_font_size_pt,
                node.caption_font_weight,
                node.caption_font_style,
            )
            text_box = draw.multiline_textbbox(
                (0, 0),
                node.caption,
                font=font,
                spacing=3,
                align="center",
            )
            text_width = text_box[2] - text_box[0]
            text_height = text_box[3] - text_box[1]
            draw.multiline_text(
                (center_x - text_width / 2, center_y - text_height / 2),
                node.caption,
                fill=node.text_color,
                font=font,
                spacing=3,
                align="center",
            )
        for edge in plan.edges:
            if edge.type not in ("directed", "dependency") or not edge.route_points:
                continue
            previous, tip = edge.route_points[-2], edge.route_points[-1]
            delta_x = (tip[0] - previous[0]) * scale
            delta_y = -(tip[1] - previous[1]) * scale
            length = hypot(delta_x, delta_y)
            if length <= 1e-6:
                continue
            unit_x, unit_y = delta_x / length, delta_y / length
            normal_x, normal_y = -unit_y, unit_x
            tip_x, tip_y = tip[0] * scale, (page_height - tip[1]) * scale
            arrow_length = min(18.0, length)
            base_x = tip_x - unit_x * arrow_length
            base_y = tip_y - unit_y * arrow_length
            half_width = 7.0
            draw.polygon(
                (
                    (tip_x, tip_y),
                    (base_x + normal_x * half_width, base_y + normal_y * half_width),
                    (base_x - normal_x * half_width, base_y - normal_y * half_width),
                ),
                fill=edge.line_color,
            )
        for edge in plan.edges:
            if not edge.label or edge.label_x is None or edge.label_y is None:
                continue
            center_x = edge.label_x * scale
            center_y = (page_height - edge.label_y) * scale
            width = edge.label_width * scale
            height = edge.label_height * scale
            draw.rectangle(
                (
                    center_x - width / 2,
                    center_y - height / 2,
                    center_x + width / 2,
                    center_y + height / 2,
                ),
                fill=appearance["pageBackgroundColor"],
            )
            font = self._font(
                scale,
                edge.font_family,
                edge.font_size_pt,
                edge.font_weight,
                edge.font_style,
            )
            text_box = draw.multiline_textbbox(
                (0, 0),
                edge.label,
                font=font,
                spacing=3,
            )
            text_width = text_box[2] - text_box[0]
            text_height = text_box[3] - text_box[1]
            draw.multiline_text(
                (center_x - text_width / 2, center_y - text_height / 2),
                edge.label,
                fill=edge.text_color,
                font=font,
                spacing=3,
                align="center",
            )
        output.parent.mkdir(parents=True, exist_ok=True)
        image.save(output)
        return {"sessionId": session_id, "output": str(output), "width": image.width, "height": image.height}

    def export(self, session_id: str, formats: list[str], output_dir: Path) -> dict:
        outputs = {}
        for format_name in formats:
            target = output_dir / f"diagram.{format_name}"
            if format_name == "png":
                outputs[format_name] = self.snapshot(session_id, target)["output"]
            else:
                target.write_text(f"mock {format_name}", encoding="utf-8")
                outputs[format_name] = str(target)
        return {"sessionId": session_id, "outputs": outputs}

    def inspect(self, session_id: str) -> dict:
        plan = self.sessions[session_id]["plan"]
        nodes = [
            {
                "id": node.id,
                "text": node.text,
                "x": node.x,
                "y": node.y,
                "width": node.width,
                "height": node.height,
                "oneD": False,
                "horizontalAlign": 1,
                "verticalAlign": 1,
                "textBlockExists": True,
                "textPinX": node.x,
                "textPinY": node.y,
                "textWidth": node.width * node.text_block_width_ratio,
                "textHeight": node.height * node.text_block_height_ratio,
                "textWidthRatio": node.text_block_width_ratio,
                "textHeightRatio": node.text_block_height_ratio,
                "fontFormula": f'FONT("{node.font_family}")',
                "fontId": None,
                "fontName": node.font_family,
                "asianFontFormula": f'FONT("{node.asian_font_family}")',
                "asianFontId": None,
                "asianFontName": node.asian_font_family,
                "fontSizePt": node.font_size_pt,
                "charStyle": char_style_code(
                    node.font_weight,
                    node.font_style,
                ),
                "textColorFormula": node.text_color,
                "textColor": node.text_color,
                "lineColorFormula": node.line_color,
                "lineColor": node.line_color,
                "fillColorFormula": node.fill_color,
                "fillColor": node.fill_color,
                "lineWeight": node.line_weight_pt / 72,
                "lineWeightPt": node.line_weight_pt,
                "cornerRadiusIn": node.corner_radius_in,
            }
            for node in plan.nodes
        ]
        captions = [
            {
                "id": node.id,
                "ownerNodeId": node.id,
                "name": f"VMCP_CAPTION_{node.id}",
                "text": node.caption,
                "x": node.caption_x,
                "y": node.caption_y,
                "width": node.caption_width,
                "height": node.caption_height,
                "horizontalAlign": 1,
                "verticalAlign": 1,
                "fontFormula": f'FONT("{node.font_family}")',
                "fontId": None,
                "fontName": node.font_family,
                "asianFontFormula": f'FONT("{node.asian_font_family}")',
                "asianFontId": None,
                "asianFontName": node.asian_font_family,
                "fontSizePt": node.caption_font_size_pt,
                "charStyle": char_style_code(
                    node.caption_font_weight,
                    node.caption_font_style,
                ),
                "textColorFormula": node.text_color,
                "textColor": node.text_color,
            }
            for node in plan.nodes
            if node.caption and node.caption_x is not None and node.caption_y is not None
        ]
        by_id = {node.id: node for node in plan.nodes}
        edges = []
        for edge in plan.edges:
            points = edge.route_points or [
                port_point(by_id[edge.source], edge.source_port, edge.source_port_position),
                port_point(by_id[edge.target], edge.target_port, edge.target_port_position),
            ]
            item = {
                "id": edge.id,
                "text": edge.label,
                "source": edge.source,
                "target": edge.target,
                "sourcePort": edge.source_port,
                "targetPort": edge.target_port,
                "sourcePortPosition": edge.source_port_position,
                "targetPortPosition": edge.target_port_position,
                "routing": edge.routing,
                "beginX": points[0][0],
                "beginY": points[0][1],
                "endX": points[-1][0],
                "endY": points[-1][1],
                "shapeRouteStyle": 1 if edge.routing == "orthogonal" else 2,
                "pathPoints": [list(point) for point in points],
                "glueCount": 2,
                "beginArrow": 0,
                "endArrow": 13 if edge.type in ("directed", "dependency") else 0,
                "endArrowSize": 2,
                "lineWeight": edge.line_weight_pt / 72,
                "lineWeightPt": edge.line_weight_pt,
                "lineColorFormula": edge.line_color,
                "lineColor": edge.line_color,
                "oneD": True,
            }
            if edge.label and edge.label_x is not None and edge.label_y is not None:
                item.update(
                    {
                        "labelX": edge.label_x,
                        "labelY": edge.label_y,
                        "labelWidth": edge.label_width,
                        "labelHeight": edge.label_height,
                        "fontFormula": f'FONT("{edge.font_family}")',
                        "fontId": None,
                        "fontName": edge.font_family,
                        "asianFontFormula": f'FONT("{edge.asian_font_family}")',
                        "asianFontId": None,
                        "asianFontName": edge.asian_font_family,
                        "fontSizePt": edge.font_size_pt,
                        "charStyle": char_style_code(
                            edge.font_weight,
                            edge.font_style,
                        ),
                        "textColorFormula": edge.text_color,
                        "textColor": edge.text_color,
                    }
                )
            edges.append(item)
        return {"nodes": nodes, "edges": edges, "captions": captions, "visible": False}

    def edit(self, session_id: str, operations: list[dict]) -> dict:
        plan = self.sessions[session_id]["plan"]
        by_id = {item.id: item for item in plan.nodes}
        applied = []
        for operation in operations:
            node = by_id[operation["nodeId"]]
            if operation["op"] == "move":
                node.x = float(operation["x"])
                node.y = float(operation["y"])
            elif operation["op"] == "resize":
                node.width = float(operation["width"])
                node.height = float(operation["height"])
            elif operation["op"] == "set_text":
                node.text = str(operation["text"])
            applied.append(operation)
        return {"sessionId": session_id, "applied": applied}

    def close(self, session_id: str, save: bool = True) -> dict:
        self.sessions.pop(session_id, None)
        return {"sessionId": session_id, "closed": True, "saved": save}
