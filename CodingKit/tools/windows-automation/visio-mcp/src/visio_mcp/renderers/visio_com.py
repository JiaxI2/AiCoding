from __future__ import annotations

import gc
from pathlib import Path
import platform
import re
import time
from uuid import uuid4

from PIL import Image

from ..errors import PlatformError, SessionError
from ..geometry import PORT_FACTORS, place_label, rectangle, segments
from ..model import DiagramPlan
from ..styling import char_style_code, color_formula, resolve_document_style
from .base import Renderer


class VisioComRenderer(Renderer):
    _RPC_E_CALL_REJECTED = -2147418111
    _RPC_E_DISCONNECTED = -2147417848

    def __init__(self) -> None:
        self.sessions: dict[str, dict] = {}

    def _win32(self):
        if platform.system() != "Windows":
            raise PlatformError("Visio COM renderer requires Windows")
        try:
            import pythoncom
            import win32com.client
        except ImportError as exc:
            raise PlatformError("pywin32 is not installed") from exc
        return win32com.client, pythoncom

    def doctor(self) -> dict:
        app = None
        pythoncom = None
        try:
            win32, pythoncom = self._win32()
            pythoncom.CoInitialize()
            app = win32.DispatchEx("Visio.Application")
            return {
                "available": True,
                "renderer": "visio-com",
                "version": str(app.Version),
                "visibleEditing": True,
            }
        except Exception as exc:
            return {"available": False, "renderer": "visio-com", "error": str(exc), "visibleEditing": False}
        finally:
            if app is not None:
                app.Quit()
            if pythoncom is not None:
                pythoncom.CoUninitialize()

    def _shape_geometry(self, page, node):
        x1, y1 = node.x - node.width / 2, node.y - node.height / 2
        x2, y2 = node.x + node.width / 2, node.y + node.height / 2
        if node.type == "decision":
            shape = page.DrawPolyline(
                [
                    node.x,
                    y2,
                    x2,
                    node.y,
                    node.x,
                    y1,
                    x1,
                    node.y,
                    node.x,
                    y2,
                ],
                0,
            )
        elif node.type == "junction":
            shape = page.DrawOval(x1, y1, x2, y2)
        else:
            shape = page.DrawRectangle(x1, y1, x2, y2)
        shape.Text = node.text
        shape.NameU = f"VMCP_{node.id}"
        self._set_formula(
            shape,
            "Rounding",
            f"{node.corner_radius_in:.4f} in",
        )
        self._set_formula(shape, "LineColor", color_formula(node.line_color))
        self._set_formula(shape, "LineWeight", f"{node.line_weight_pt:.4f} pt")
        self._set_formula(shape, "FillPattern", "1")
        self._set_formula(shape, "FillForegnd", color_formula(node.fill_color))
        self._configure_node_text(shape, node)
        if node.type == "note":
            self._set_formula(shape, "LinePattern", "0")
            self._set_formula(shape, "FillPattern", "0")
        if node.type == "group":
            try:
                shape.SendToBack()
            except Exception:
                pass
        return shape

    @staticmethod
    def _set_formula(shape, cell_name: str, formula: str) -> None:
        try:
            shape.CellsU(cell_name).FormulaU = formula
        except Exception:
            pass

    def _set_text_style(
        self,
        shape,
        font_family: str,
        asian_font_family: str,
        font_size_pt: float,
        font_weight: str,
        font_style: str,
        text_color: str,
    ) -> None:
        self._set_formula(shape, "Char.Font", f'FONT("{font_family}")')
        self._set_formula(
            shape,
            "Char.AsianFont",
            f'FONT("{asian_font_family}")',
        )
        self._set_formula(shape, "Char.Size", f"{font_size_pt:.4f} pt")
        self._set_formula(
            shape,
            "Char.Style",
            str(char_style_code(font_weight, font_style)),
        )
        self._set_formula(shape, "Char.Color", color_formula(text_color))

    def _configure_node_text(self, shape, node) -> None:
        try:
            if not shape.CellExistsU("TxtPinX", 0):
                shape.AddRow(1, 12, 0)
        except Exception:
            pass
        width_factor = f"{node.text_block_width_ratio:.6f}"
        height_factor = f"{node.text_block_height_ratio:.6f}"
        formulas = {
            "Para.HorzAlign": "1",
            "VerticalAlign": "1",
            "LeftMargin": "0.01 in",
            "RightMargin": "0.01 in",
            "TopMargin": "0.01 in",
            "BottomMargin": "0.01 in",
            "TxtPinX": "Width*0.5",
            "TxtPinY": "Height*0.5",
            "TxtWidth": f"Width*{width_factor}",
            "TxtHeight": f"Height*{height_factor}",
            "TxtLocPinX": "TxtWidth*0.5",
            "TxtLocPinY": "TxtHeight*0.5",
            "TxtAngle": "0 deg",
        }
        for cell_name, formula in formulas.items():
            self._set_formula(shape, cell_name, formula)
        self._set_text_style(
            shape,
            node.font_family,
            node.asian_font_family,
            node.font_size_pt,
            node.font_weight,
            node.font_style,
            node.text_color,
        )

    def _caption_geometry(self, page, node, owner):
        x1 = node.caption_x - node.caption_width / 2
        y1 = node.caption_y - node.caption_height / 2
        x2 = node.caption_x + node.caption_width / 2
        y2 = node.caption_y + node.caption_height / 2
        shape = page.DrawRectangle(x1, y1, x2, y2)
        shape.Text = node.caption
        shape.NameU = f"VMCP_CAPTION_{node.id}"
        self._set_formula(shape, "LinePattern", "0")
        self._set_formula(shape, "FillPattern", "0")
        self._set_formula(shape, "Para.HorzAlign", "1")
        self._set_formula(shape, "VerticalAlign", "1")
        self._set_formula(shape, "TxtPinX", "Width*0.5")
        self._set_formula(shape, "TxtPinY", "Height*0.5")
        self._set_formula(shape, "TxtWidth", "Width")
        self._set_formula(shape, "TxtHeight", "Height")
        self._set_formula(shape, "TxtLocPinX", "TxtWidth*0.5")
        self._set_formula(shape, "TxtLocPinY", "TxtHeight*0.5")
        self._set_text_style(
            shape,
            node.font_family,
            node.asian_font_family,
            node.caption_font_size_pt,
            node.caption_font_weight,
            node.caption_font_style,
            node.text_color,
        )

        owner_ref = f"Sheet.{int(owner.ID)}"
        position = node.caption_position - 0.5
        offset = node.caption_offset
        side = node.caption_resolved_side or node.caption_side
        if side in ("top", "bottom"):
            self._set_formula(
                shape,
                "PinX",
                f"GUARD({owner_ref}!PinX+({position:.6f})*{owner_ref}!Width)",
            )
            sign = "+" if side == "top" else "-"
            self._set_formula(
                shape,
                "PinY",
                f"GUARD({owner_ref}!PinY{sign}{owner_ref}!Height/2{sign}{offset:.6f} in{sign}Height/2)",
            )
        else:
            sign = "+" if side == "right" else "-"
            self._set_formula(
                shape,
                "PinX",
                f"GUARD({owner_ref}!PinX{sign}{owner_ref}!Width/2{sign}{offset:.6f} in{sign}Width/2)",
            )
            self._set_formula(
                shape,
                "PinY",
                f"GUARD({owner_ref}!PinY+({position:.6f})*{owner_ref}!Height)",
            )
        return shape

    def _label_connector(self, connector, edge) -> None:
        if not edge.label:
            return
        connector.Text = edge.label
        try:
            connector.CellsU("TxtAngle").FormulaU = "-Angle"
            connector.CellsU("TextBkgnd").FormulaU = "RGB(255,255,255)+1"
            connector.CellsU("TextBkgndTrans").FormulaU = "0%"
            connector.CellsU("LeftMargin").FormulaU = "0.03 in"
            connector.CellsU("RightMargin").FormulaU = "0.03 in"
            connector.CellsU("TopMargin").FormulaU = "0.02 in"
            connector.CellsU("BottomMargin").FormulaU = "0.02 in"
            connector.CellsU("Para.HorzAlign").FormulaU = "1"
            connector.CellsU("VerticalAlign").FormulaU = "1"
            connector.CellsU("TxtWidth").ResultIU = float(edge.label_width)
            connector.CellsU("TxtHeight").ResultIU = float(edge.label_height)
            self._set_text_style(
                connector,
                edge.font_family,
                edge.asian_font_family,
                edge.font_size_pt,
                edge.font_weight,
                edge.font_style,
                edge.text_color,
            )
            if edge.label_x is not None and edge.label_y is not None:
                local_x, local_y = connector.XYFromPage(edge.label_x, edge.label_y)
                connector.CellsU("TxtPinX").ResultIU = float(local_x)
                connector.CellsU("TxtPinY").ResultIU = float(local_y)
        except Exception:
            pass

    @staticmethod
    def _font_details(shape, cell_name: str) -> dict:
        try:
            cell = shape.CellsU(cell_name)
            formula = str(cell.FormulaU)
            font_id = int(round(float(cell.ResultIU)))
        except Exception:
            return {
                "formula": "",
                "id": None,
                "name": "",
            }
        font_name = ""
        for owner in (getattr(shape, "Document", None), getattr(shape, "Application", None)):
            fonts = getattr(owner, "Fonts", None)
            if fonts is None:
                continue
            for accessor in ("ItemFromID", "Item"):
                try:
                    font_name = str(getattr(fonts, accessor)(font_id).Name)
                    break
                except Exception:
                    continue
            if font_name:
                break
        if not font_name:
            match = re.search(r'FONT\("([^"]+)"\)', formula, re.IGNORECASE)
            if match:
                font_name = match.group(1)
        return {
            "formula": formula,
            "id": font_id,
            "name": font_name,
        }

    @staticmethod
    def _color_details(shape, cell_name: str) -> dict:
        try:
            formula = str(shape.CellsU(cell_name).FormulaU)
        except Exception:
            return {"formula": "", "hex": ""}
        match = re.search(
            r"RGB\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)",
            formula,
            re.IGNORECASE,
        )
        if not match:
            return {"formula": formula, "hex": ""}
        red, green, blue = (int(value) for value in match.groups())
        return {
            "formula": formula,
            "hex": f"#{red:02X}{green:02X}{blue:02X}",
        }

    @staticmethod
    def _font_size_pt(shape) -> float:
        try:
            return float(shape.CellsU("Char.Size").ResultIU) * 72.0
        except Exception:
            return 0.0

    @staticmethod
    def _char_style(shape) -> int:
        try:
            return int(round(float(shape.CellsU("Char.Style").ResultIU)))
        except Exception:
            return 0

    @staticmethod
    def _path_points(shape) -> list[list[float]]:
        output: list[list[float]] = []
        try:
            for index in range(1, int(shape.Paths.Count) + 1):
                values = shape.Paths.Item(index).Points(0.01)
                points = [
                    [float(values[offset]), float(values[offset + 1])]
                    for offset in range(0, len(values), 2)
                ]
                if output and points and output[-1] == points[0]:
                    output.extend(points[1:])
                else:
                    output.extend(points)
        except Exception:
            return []
        return output

    def render(self, plan: DiagramPlan, output: Path, visible: bool = False) -> dict:
        win32, pythoncom = self._win32()
        pythoncom.CoInitialize()
        app = win32.DispatchEx("Visio.Application")
        app.Visible = bool(visible)
        app.AlertResponse = 7
        document = app.Documents.Add("")
        page = document.Pages.Item(1)
        page.Name = plan.document.get("title", "Diagram")[:100]
        settings = plan.document.get("page", {})
        try:
            page.PageSheet.CellsU("PageWidth").ResultIU = float(settings.get("width", 16))
            page.PageSheet.CellsU("PageHeight").ResultIU = float(settings.get("height", 9))
            appearance = resolve_document_style(plan.document)["appearance"]
            page.PageSheet.CellsU("PageColor").FormulaU = color_formula(
                appearance["pageBackgroundColor"]
            )
        except Exception:
            pass

        shapes = {node.id: self._shape_geometry(page, node) for node in plan.nodes}
        for node in plan.nodes:
            if node.caption and node.caption_x is not None and node.caption_y is not None:
                self._caption_geometry(page, node, shapes[node.id])
        connectors = {}
        for edge in plan.edges:
            connector = page.Drop(app.ConnectorToolDataObject, 0, 0)
            connector.NameU = f"VMCP_{edge.id}"
            source_x, source_y = PORT_FACTORS[edge.source_port]
            target_x, target_y = PORT_FACTORS[edge.target_port]
            if edge.source_port in ("left", "right"):
                source_y = edge.source_port_position
            else:
                source_x = edge.source_port_position
            if edge.target_port in ("left", "right"):
                target_y = edge.target_port_position
            else:
                target_x = edge.target_port_position
            connector.CellsU("BeginX").GlueToPos(shapes[edge.source], source_x, source_y)
            connector.CellsU("EndX").GlueToPos(shapes[edge.target], target_x, target_y)
            connector.CellsU("ShapeRouteStyle").FormulaU = "1" if edge.routing == "orthogonal" else "2"
            connector.CellsU("ConFixedCode").FormulaU = "0"
            connector.CellsU("BeginArrow").FormulaU = "0"
            connector.CellsU("LineColor").FormulaU = color_formula(edge.line_color)
            connector.CellsU("LineWeight").FormulaU = f"{edge.line_weight_pt:.4f} pt"
            if edge.type in ("directed", "dependency"):
                connector.CellsU("EndArrow").FormulaU = "13"
                connector.CellsU("EndArrowSize").FormulaU = "2"
            else:
                connector.CellsU("EndArrow").FormulaU = "0"
            connectors[edge.id] = connector

        actual_points = {}
        for edge in plan.edges:
            connector = connectors[edge.id]
            begin = (
                float(connector.CellsU("BeginX").ResultIU),
                float(connector.CellsU("BeginY").ResultIU),
            )
            end = (
                float(connector.CellsU("EndX").ResultIU),
                float(connector.CellsU("EndY").ResultIU),
            )
            points = self._path_points(connector)
            if points:
                points[0] = [begin[0], begin[1]]
                points[-1] = [end[0], end[1]]
                actual_points[edge.id] = [
                    (float(point[0]), float(point[1]))
                    for point in points
                ]
            else:
                actual_points[edge.id] = [begin, end]
        actual_segments = [
            segment
            for edge in plan.edges
            for segment in segments(actual_points[edge.id])
        ]
        occupied_text = [
            rectangle(
                node.caption_x,
                node.caption_y,
                node.caption_width,
                node.caption_height,
            )
            for node in plan.nodes
            if node.caption and node.caption_x is not None and node.caption_y is not None
        ]
        margin = float(settings.get("margin", 0.5))
        page_width = float(settings.get("width", 16))
        page_height = float(settings.get("height", 9))
        for edge in plan.edges:
            if edge.label:
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
                    actual_points[edge.id],
                    edge.label_width,
                    edge.label_height,
                    edge.label_side,
                    edge.label_offset,
                    edge.label_position,
                    plan.nodes,
                    actual_segments,
                    occupied_text,
                    page_width,
                    page_height,
                    margin,
                )
                occupied_text.append(
                    rectangle(
                        edge.label_x,
                        edge.label_y,
                        edge.label_width,
                        edge.label_height,
                    )
                )
            self._label_connector(connectors[edge.id], edge)
        output.parent.mkdir(parents=True, exist_ok=True)
        document.SaveAs(str(output.resolve()))
        session_id = str(uuid4())
        self.sessions[session_id] = {
            "app": app,
            "doc": document,
            "page": page,
            "output": str(output.resolve()),
            "visible": visible,
            "pythoncom": pythoncom,
            "plan": plan,
        }
        return {
            "sessionId": session_id,
            "output": str(output.resolve()),
            "renderer": "visio-com",
            "visible": visible,
        }

    def _session(self, session_id: str) -> dict:
        if session_id not in self.sessions:
            raise SessionError(f"Unknown session: {session_id}")
        return self.sessions[session_id]

    @staticmethod
    def _raster_margin_ratio(session: dict) -> float:
        plan = session.get("plan")
        if plan is None:
            return 0.05
        settings = plan.document.get("page", {})
        width = float(settings.get("width", 16))
        height = float(settings.get("height", 9))
        margin = float(settings.get("margin", 0.5))
        return max(0.04, min(0.08, margin / max(0.01, min(width, height))))

    @staticmethod
    def _pad_png(path: Path, margin_ratio: float) -> None:
        if not path.exists():
            return
        with Image.open(path) as source:
            image = source.convert("RGBA")
        padding_x = max(8, round(image.width * margin_ratio))
        padding_y = max(8, round(image.height * margin_ratio))
        output = Image.new(
            "RGB",
            (image.width + 2 * padding_x, image.height + 2 * padding_y),
            "white",
        )
        output.paste(image, (padding_x, padding_y), image)
        output.save(path)

    def _com_call(self, session: dict, operation, attempts: int = 10):
        for attempt in range(attempts):
            try:
                return operation()
            except Exception as exc:
                if getattr(exc, "hresult", None) != self._RPC_E_CALL_REJECTED or attempt == attempts - 1:
                    raise
                session["pythoncom"].PumpWaitingMessages()
                time.sleep(0.2)

    @classmethod
    def _is_disconnected(cls, error: Exception) -> bool:
        return (
            getattr(error, "hresult", None) == cls._RPC_E_DISCONNECTED
            or isinstance(error, AttributeError)
        )

    def snapshot(self, session_id: str, output: Path) -> dict:
        session = self._session(session_id)
        output.parent.mkdir(parents=True, exist_ok=True)
        session["page"].Export(str(output.resolve()))
        if output.suffix.lower() == ".png":
            self._pad_png(output, self._raster_margin_ratio(session))
        return {"sessionId": session_id, "output": str(output.resolve())}

    def export(self, session_id: str, formats: list[str], output_dir: Path) -> dict:
        session = self._session(session_id)
        output_dir.mkdir(parents=True, exist_ok=True)
        outputs = {}
        base = Path(session["output"]).stem
        ordered_formats = sorted(
            formats,
            key=lambda value: 0 if value.lower() == "vsdx" else 2 if value.lower() == "pdf" else 1,
        )
        for format_name in ordered_formats:
            target = (output_dir / f"{base}.{format_name}").resolve()
            normalized_format = format_name.lower()
            if normalized_format == "vsdx":
                self._com_call(session, lambda: session["doc"].SaveAs(str(target)))
            elif normalized_format == "pdf":
                self._com_call(session, lambda: session["doc"].ExportAsFixedFormat(1, str(target), 1, 0))
            else:
                self._com_call(session, lambda: session["page"].Export(str(target)))
                if normalized_format == "png":
                    self._pad_png(target, self._raster_margin_ratio(session))
            outputs[format_name] = str(target)
        return {"sessionId": session_id, "outputs": outputs}

    def inspect(self, session_id: str) -> dict:
        session = self._session(session_id)
        nodes, edges, captions = [], [], []
        planned_edges = {edge.id: edge for edge in session["plan"].edges}
        shapes = session["page"].Shapes
        shape_count = int(self._com_call(session, lambda: shapes.Count))
        for index in range(1, shape_count + 1):
            shape = self._com_call(
                session,
                lambda index=index: shapes.Item(index),
            )
            name = str(shape.NameU)
            if not name.startswith("VMCP_"):
                continue
            if name.startswith("VMCP_CAPTION_"):
                font = self._font_details(shape, "Char.Font")
                asian_font = self._font_details(shape, "Char.AsianFont")
                text_color = self._color_details(shape, "Char.Color")
                captions.append(
                    {
                        "id": name[len("VMCP_CAPTION_") :],
                        "ownerNodeId": name[len("VMCP_CAPTION_") :],
                        "name": name,
                        "text": str(shape.Text or ""),
                        "x": float(shape.CellsU("PinX").ResultIU),
                        "y": float(shape.CellsU("PinY").ResultIU),
                        "width": float(shape.CellsU("Width").ResultIU),
                        "height": float(shape.CellsU("Height").ResultIU),
                        "horizontalAlign": float(shape.CellsU("Para.HorzAlign").ResultIU),
                        "verticalAlign": float(shape.CellsU("VerticalAlign").ResultIU),
                        "fontFormula": font["formula"],
                        "fontId": font["id"],
                        "fontName": font["name"],
                        "asianFontFormula": asian_font["formula"],
                        "asianFontId": asian_font["id"],
                        "asianFontName": asian_font["name"],
                        "fontSizePt": self._font_size_pt(shape),
                        "charStyle": self._char_style(shape),
                        "textColorFormula": text_color["formula"],
                        "textColor": text_color["hex"],
                    }
                )
                continue
            item = {
                "id": name[5:],
                "name": name,
                "text": str(shape.Text or ""),
                "x": float(shape.CellsU("PinX").ResultIU),
                "y": float(shape.CellsU("PinY").ResultIU),
                "width": float(shape.CellsU("Width").ResultIU),
                "height": float(shape.CellsU("Height").ResultIU),
                "oneD": bool(shape.OneD),
            }
            if item["oneD"]:
                edge = planned_edges.get(item["id"])
                begin_x = float(shape.CellsU("BeginX").ResultIU)
                begin_y = float(shape.CellsU("BeginY").ResultIU)
                end_x = float(shape.CellsU("EndX").ResultIU)
                end_y = float(shape.CellsU("EndY").ResultIU)
                path_points = self._path_points(shape)
                if path_points:
                    path_points[0] = [begin_x, begin_y]
                    path_points[-1] = [end_x, end_y]
                else:
                    path_points = [[begin_x, begin_y], [end_x, end_y]]
                item.update(
                    {
                        "beginX": begin_x,
                        "beginY": begin_y,
                        "endX": end_x,
                        "endY": end_y,
                        "shapeRouteStyle": float(shape.CellsU("ShapeRouteStyle").ResultIU),
                        "pathPoints": path_points,
                        "glueCount": int(shape.Connects.Count),
                        "beginArrow": float(shape.CellsU("BeginArrow").ResultIU),
                        "endArrow": float(shape.CellsU("EndArrow").ResultIU),
                        "endArrowSize": float(shape.CellsU("EndArrowSize").ResultIU),
                        "lineWeight": float(shape.CellsU("LineWeight").ResultIU),
                        "lineWeightPt": float(
                            shape.CellsU("LineWeight").ResultIU
                        )
                        * 72.0,
                    }
                )
                line_color = self._color_details(shape, "LineColor")
                item.update(
                    {
                        "lineColorFormula": line_color["formula"],
                        "lineColor": line_color["hex"],
                    }
                )
                if edge is not None:
                    item.update(
                        {
                            "source": edge.source,
                            "target": edge.target,
                            "sourcePort": edge.source_port,
                            "targetPort": edge.target_port,
                            "sourcePortPosition": edge.source_port_position,
                            "targetPortPosition": edge.target_port_position,
                            "routing": edge.routing,
                        }
                    )
                if item["text"]:
                    label_x, label_y = shape.XYToPage(
                        float(shape.CellsU("TxtPinX").ResultIU),
                        float(shape.CellsU("TxtPinY").ResultIU),
                    )
                    item.update(
                        {
                            "labelX": float(label_x),
                            "labelY": float(label_y),
                            "labelWidth": float(shape.CellsU("TxtWidth").ResultIU),
                            "labelHeight": float(shape.CellsU("TxtHeight").ResultIU),
                        }
                    )
                    font = self._font_details(shape, "Char.Font")
                    asian_font = self._font_details(shape, "Char.AsianFont")
                    text_color = self._color_details(shape, "Char.Color")
                    item.update(
                        {
                            "fontFormula": font["formula"],
                            "fontId": font["id"],
                            "fontName": font["name"],
                            "asianFontFormula": asian_font["formula"],
                            "asianFontId": asian_font["id"],
                            "asianFontName": asian_font["name"],
                            "fontSizePt": self._font_size_pt(shape),
                            "charStyle": self._char_style(shape),
                            "textColorFormula": text_color["formula"],
                            "textColor": text_color["hex"],
                        }
                    )
                edges.append(item)
            else:
                text_block_exists = bool(shape.CellExistsU("TxtPinX", 0))
                item.update(
                    {
                        "horizontalAlign": float(shape.CellsU("Para.HorzAlign").ResultIU),
                        "verticalAlign": float(shape.CellsU("VerticalAlign").ResultIU),
                        "textBlockExists": text_block_exists,
                    }
                )
                if text_block_exists:
                    text_x, text_y = shape.XYToPage(
                        float(shape.CellsU("TxtPinX").ResultIU),
                        float(shape.CellsU("TxtPinY").ResultIU),
                    )
                    item.update(
                        {
                            "textPinX": float(text_x),
                            "textPinY": float(text_y),
                            "textWidth": float(shape.CellsU("TxtWidth").ResultIU),
                            "textHeight": float(shape.CellsU("TxtHeight").ResultIU),
                        }
                    )
                font = self._font_details(shape, "Char.Font")
                asian_font = self._font_details(shape, "Char.AsianFont")
                text_color = self._color_details(shape, "Char.Color")
                line_color = self._color_details(shape, "LineColor")
                fill_color = self._color_details(shape, "FillForegnd")
                item.update(
                    {
                        "textWidthRatio": item.get("textWidth", 0.0) / max(0.01, item["width"]),
                        "textHeightRatio": item.get("textHeight", 0.0) / max(0.01, item["height"]),
                        "fontFormula": font["formula"],
                        "fontId": font["id"],
                        "fontName": font["name"],
                        "asianFontFormula": asian_font["formula"],
                        "asianFontId": asian_font["id"],
                        "asianFontName": asian_font["name"],
                        "fontSizePt": self._font_size_pt(shape),
                        "charStyle": self._char_style(shape),
                        "textColorFormula": text_color["formula"],
                        "textColor": text_color["hex"],
                        "lineColorFormula": line_color["formula"],
                        "lineColor": line_color["hex"],
                        "fillColorFormula": fill_color["formula"],
                        "fillColor": fill_color["hex"],
                        "lineWeight": float(
                            shape.CellsU("LineWeight").ResultIU
                        ),
                        "lineWeightPt": float(
                            shape.CellsU("LineWeight").ResultIU
                        )
                        * 72.0,
                        "cornerRadiusIn": float(
                            shape.CellsU("Rounding").ResultIU
                        ),
                    }
                )
                nodes.append(item)
        return {
            "sessionId": session_id,
            "nodes": nodes,
            "edges": edges,
            "captions": captions,
            "visible": session["visible"],
        }

    def edit(self, session_id: str, operations: list[dict]) -> dict:
        session = self._session(session_id)
        applied = []
        for operation in operations:
            shape = session["page"].Shapes.ItemU(f"VMCP_{operation['nodeId']}")
            kind = operation["op"]
            if kind == "move":
                shape.CellsU("PinX").ResultIU = float(operation["x"])
                shape.CellsU("PinY").ResultIU = float(operation["y"])
            elif kind == "resize":
                shape.CellsU("Width").ResultIU = float(operation["width"])
                shape.CellsU("Height").ResultIU = float(operation["height"])
            elif kind == "set_text":
                shape.Text = str(operation["text"])
            elif kind == "select":
                session["app"].ActiveWindow.Select(shape, 2)
            else:
                raise ValueError(f"Unsupported edit op: {kind}")
            applied.append(operation)
        session["doc"].Save()
        return {"sessionId": session_id, "applied": applied}

    def close(self, session_id: str, save: bool = True) -> dict:
        session = self._session(session_id)
        pythoncom = session["pythoncom"]
        error = None
        try:
            if save:
                try:
                    self._com_call(session, lambda: session["doc"].Save())
                except Exception as exc:
                    if not self._is_disconnected(exc):
                        error = exc
            try:
                self._com_call(session, lambda: session["doc"].Close())
            except Exception as exc:
                if not self._is_disconnected(exc):
                    error = error or exc
            try:
                self._com_call(session, lambda: session["app"].Quit())
            except Exception as exc:
                if not self._is_disconnected(exc):
                    error = error or exc
        finally:
            self.sessions.pop(session_id, None)
            session["page"] = None
            session["doc"] = None
            session["app"] = None
            gc.collect()
            pythoncom.CoUninitialize()
        if error is not None:
            raise error
        return {"sessionId": session_id, "closed": True, "saved": save}
