from __future__ import annotations

from collections import defaultdict
from copy import deepcopy

from .model import DiagramPlan
from .quality import structural_quality


def auto_repair(plan: DiagramPlan, max_passes: int = 4) -> tuple[DiagramPlan, dict]:
    repaired = deepcopy(plan)
    before = structural_quality(repaired)
    actions = []
    page = repaired.document.get("page", {})
    page_width = float(page.get("width", 16))
    page_height = float(page.get("height", 9))
    margin = float(page.get("margin", 0.5))
    layout = repaired.document.get("layout", {})
    direction = layout.get("direction", "LR")
    uniform_node_size = bool(layout.get("uniformNodeSize", True))

    for pass_number in range(max_passes):
        report = structural_quality(repaired)
        if not report["findings"]:
            break
        changed = False
        for finding in report["findings"]:
            if finding["code"] == "OUT_OF_PAGE":
                node = next(item for item in repaired.nodes if item.id == finding["objects"][0])
                old = (node.x, node.y)
                node.x = min(max(node.x, margin + node.width / 2), page_width - margin - node.width / 2)
                node.y = min(max(node.y, margin + node.height / 2), page_height - margin - node.height / 2)
                actions.append(
                    {"pass": pass_number + 1, "action": "clamp_to_page", "node": node.id, "from": old, "to": (node.x, node.y)}
                )
                changed = True
            elif finding["code"] == "TEXT_OVERFLOW_RISK":
                node = next(item for item in repaired.nodes if item.id == finding["objects"][0])
                if uniform_node_size:
                    old = {item.id: item.height for item in repaired.nodes}
                    target = min(3.0, max(item.height for item in repaired.nodes) + 0.35)
                    for item in repaired.nodes:
                        item.height = target
                    actions.append(
                        {
                            "pass": pass_number + 1,
                            "action": "grow_uniform_height",
                            "nodes": [item.id for item in repaired.nodes],
                            "from": old,
                            "to": target,
                        }
                    )
                else:
                    old = node.height
                    node.height = min(3.0, node.height + 0.35)
                    actions.append(
                        {"pass": pass_number + 1, "action": "grow_height", "node": node.id, "from": old, "to": node.height}
                    )
                changed = True
            elif finding["code"] == "NODE_OVERLAP":
                first = next(item for item in repaired.nodes if item.id == finding["objects"][0])
                second = next(item for item in repaired.nodes if item.id == finding["objects"][1])
                old = (second.x, second.y)
                second.y = max(margin + second.height / 2, second.y - (second.height + 0.35))
                if abs(second.y - old[1]) < 0.01:
                    second.x = min(page_width - margin - second.width / 2, second.x + second.width + 0.35)
                actions.append(
                    {
                        "pass": pass_number + 1,
                        "action": "separate_nodes",
                        "nodes": [first.id, second.id],
                        "from": old,
                        "to": (second.x, second.y),
                    }
                )
                changed = True
            elif finding["code"] == "INCONSISTENT_NODE_SIZE":
                nodes = [next(item for item in repaired.nodes if item.id == node_id) for node_id in finding["objects"]]
                target_width = sorted(item.width for item in nodes)[len(nodes) // 2]
                target_height = sorted(item.height for item in nodes)[len(nodes) // 2]
                old = {item.id: (item.width, item.height) for item in nodes}
                for item in nodes:
                    item.width = target_width
                    item.height = target_height
                actions.append(
                    {
                        "pass": pass_number + 1,
                        "action": "normalize_node_size",
                        "nodes": [item.id for item in nodes],
                        "from": old,
                        "to": (target_width, target_height),
                    }
                )
                changed = True
            elif finding["code"] in ("LAYER_MISALIGNED", "ORDER_MISALIGNED"):
                nodes = [next(item for item in repaired.nodes if item.id == node_id) for node_id in finding["objects"]]
                if finding["code"] == "LAYER_MISALIGNED":
                    axis = "x" if direction in ("LR", "RL") else "y"
                else:
                    axis = "y" if direction in ("LR", "RL") else "x"
                old = {item.id: getattr(item, axis) for item in nodes}
                target = sum(old.values()) / len(old)
                for item in nodes:
                    setattr(item, axis, target)
                actions.append(
                    {
                        "pass": pass_number + 1,
                        "action": "align_centers",
                        "axis": axis,
                        "nodes": [item.id for item in nodes],
                        "from": old,
                        "to": target,
                    }
                )
                changed = True
            elif finding["code"] == "INCONSISTENT_LAYER_SPACING":
                groups = defaultdict(list)
                for item in repaired.nodes:
                    groups[item.layer].append(item)
                layers = sorted(groups)
                if len(layers) > 2:
                    axis = "x" if direction in ("LR", "RL") else "y"
                    centers = [sum(getattr(item, axis) for item in groups[layer]) / len(groups[layer]) for layer in layers]
                    step = (centers[-1] - centers[0]) / (len(centers) - 1)
                    old = {}
                    for index, layer in enumerate(layers):
                        target = centers[0] + index * step
                        for item in groups[layer]:
                            old[item.id] = getattr(item, axis)
                            setattr(item, axis, target)
                    actions.append(
                        {
                            "pass": pass_number + 1,
                            "action": "equalize_layer_spacing",
                            "axis": axis,
                            "nodes": [item.id for item in repaired.nodes],
                            "from": old,
                            "toStep": step,
                        }
                    )
                    changed = True
        if not changed:
            break

    after = structural_quality(repaired)
    return repaired, {
        "before": before,
        "after": after,
        "actions": actions,
        "improved": after["score"] > before["score"],
    }
