from __future__ import annotations

from copy import deepcopy
import json
import os
from pathlib import Path
import re
from typing import Any

from jsonschema import Draft202012Validator


PROJECT_ROOT = Path(__file__).resolve().parents[2]
DEFAULT_STYLE_PROFILES_PATH = PROJECT_ROOT / "styles" / "style-profiles.json"
STYLE_PROFILE_SCHEMA_PATH = PROJECT_ROOT / "schemas" / "style-profile.schema.json"
COLOR_PATTERN = re.compile(r"^#[0-9A-Fa-f]{6}$")

BASE_STYLE_PROFILE = {
    "typography": {
        "latinFontFamily": "SimSun",
        "asianFontFamily": "SimSun",
        "mathFontFamily": "Times New Roman",
        "minimumFontSizePt": 10,
        "nodeTextBlockWidthRatio": 0.8,
        "nodeTextBlockHeightRatio": 0.8,
        "roles": {
            "module": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "operator": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "math": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "math",
            },
            "signal": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "body": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "caption": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "groupTitle": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "bold",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "note": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "dense": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
            "edgeLabel": {
                "fontSizePt": 10,
                "minimumFontSizePt": 10,
                "fontWeight": "regular",
                "fontStyle": "normal",
                "fontFamilyRole": "default",
            },
        },
    },
    "appearance": {
        "pageBackgroundColor": "#FFFFFF",
        "textColor": "#000000",
        "nodeLineColor": "#000000",
        "nodeFillColor": "#FFFFFF",
        "nodeLineWeightPt": 0.75,
        "connectorLineColor": "#000000",
        "connectorLineWeightPt": 0.75,
        "nodeCornerRadiusIn": 0.12,
    },
    "nodeStyles": {
        "default": {},
        "square": {"cornerRadiusIn": 0.0},
        "secondary": {
            "textColor": "#262626",
            "lineColor": "#595959",
            "fillColor": "#F2F2F2",
        },
        "feedback": {
            "textColor": "#17365D",
            "lineColor": "#1F4E79",
            "fillColor": "#D9EAF7",
        },
        "warning": {
            "textColor": "#9C0006",
            "lineColor": "#C00000",
            "fillColor": "#FCE4D6",
        },
        "success": {
            "textColor": "#375623",
            "lineColor": "#548235",
            "fillColor": "#E2F0D9",
        },
    },
    "edgeStyles": {
        "default": {},
        "secondary": {
            "textColor": "#404040",
            "lineColor": "#595959",
        },
        "feedback": {
            "textColor": "#17365D",
            "lineColor": "#1F4E79",
        },
        "warning": {
            "textColor": "#9C0006",
            "lineColor": "#C00000",
        },
        "success": {
            "textColor": "#375623",
            "lineColor": "#548235",
        },
    },
}


def style_profiles_path() -> Path:
    configured = os.environ.get("VISIO_MCP_STYLE_PROFILES")
    return Path(configured).expanduser().resolve() if configured else DEFAULT_STYLE_PROFILES_PATH


def load_style_profiles() -> dict[str, Any]:
    path = style_profiles_path()
    registry = json.loads(path.read_text(encoding="utf-8"))
    schema = json.loads(STYLE_PROFILE_SCHEMA_PATH.read_text(encoding="utf-8"))
    errors = sorted(
        Draft202012Validator(schema).iter_errors(registry),
        key=lambda error: list(error.path),
    )
    if errors:
        detail = [
            {
                "path": "/".join(map(str, error.path)),
                "message": error.message,
            }
            for error in errors
        ]
        raise ValueError(
            f"Invalid Visio style profile registry {path}: "
            f"{json.dumps(detail, ensure_ascii=False)}"
        )
    default_profile = registry["defaultProfile"]
    if default_profile not in registry["profiles"]:
        raise ValueError(
            f"Default Visio style profile is not defined: {default_profile}"
        )
    return registry


def read_style_profiles_text() -> str:
    return style_profiles_path().read_text(encoding="utf-8")


def _deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    result = deepcopy(base)
    for key, value in override.items():
        if (
            isinstance(value, dict)
            and isinstance(result.get(key), dict)
        ):
            result[key] = _deep_merge(result[key], value)
        else:
            result[key] = deepcopy(value)
    return result


def _expand_registry_profile(raw_profile: dict[str, Any]) -> dict[str, Any]:
    profile = deepcopy(BASE_STYLE_PROFILE)
    font = raw_profile["font"]
    text = raw_profile["text"]
    line = raw_profile["line"]
    profile["typography"].update(
        {
            "latinFontFamily": font["ordinaryFamily"],
            "asianFontFamily": font.get(
                "asianFamily",
                font["ordinaryFamily"],
            ),
            "mathFontFamily": font.get(
                "mathFamily",
                font["ordinaryFamily"],
            ),
            "nodeTextBlockWidthRatio": text["safeRatio"],
            "nodeTextBlockHeightRatio": text["safeRatio"],
        }
    )
    default_font_size = float(text["fontSizePt"])
    for role in profile["typography"]["roles"].values():
        role["fontSizePt"] = default_font_size
        role["minimumFontSizePt"] = default_font_size
    profile["appearance"].update(
        {
            "nodeLineWeightPt": line["weightPt"],
            "connectorLineWeightPt": line["weightPt"],
            "nodeCornerRadiusIn": line["cornerRadiusIn"],
        }
    )
    return profile


def resolve_document_style(document: dict[str, Any]) -> dict[str, Any]:
    registry = load_style_profiles()
    profile_name = str(document.get("styleProfile", registry["defaultProfile"]))
    if profile_name not in registry["profiles"]:
        raise ValueError(f"Unknown Visio style profile: {profile_name}")
    profile = _expand_registry_profile(registry["profiles"][profile_name])
    document_typography = document.get("typography", {})
    font_size_override = document_typography.get("fontSizePt")
    math_font_size_override = document_typography.get("mathFontSizePt")
    profile["typography"] = _deep_merge(
        profile["typography"],
        document_typography,
    )
    if font_size_override is not None:
        for role_name, role in profile["typography"]["roles"].items():
            if role_name != "math":
                role["fontSizePt"] = float(font_size_override)
                role["minimumFontSizePt"] = float(font_size_override)
    if math_font_size_override is not None:
        profile["typography"]["roles"]["math"]["fontSizePt"] = float(
            math_font_size_override
        )
        profile["typography"]["roles"]["math"][
            "minimumFontSizePt"
        ] = float(math_font_size_override)
    legacy_font_family = document_typography.get("fontFamily")
    if legacy_font_family:
        if "latinFontFamily" not in document_typography:
            profile["typography"]["latinFontFamily"] = legacy_font_family
        if "asianFontFamily" not in document_typography:
            profile["typography"]["asianFontFamily"] = legacy_font_family
    document_appearance = document.get("appearance", {})
    shared_line_weight = document_appearance.get("lineWeightPt")
    if shared_line_weight is not None:
        profile["appearance"]["nodeLineWeightPt"] = float(shared_line_weight)
        profile["appearance"]["connectorLineWeightPt"] = float(
            shared_line_weight
        )
    profile["appearance"] = _deep_merge(
        profile["appearance"],
        document_appearance,
    )
    for role_name, role in profile["typography"]["roles"].items():
        missing = {
            "fontWeight",
            "fontStyle",
            "fontFamilyRole",
            "fontSizePt",
        } - set(role)
        if missing:
            raise ValueError(
                f"Incomplete typography role {role_name!r}: missing {sorted(missing)}"
            )
    profile["name"] = profile_name
    return profile


def resolve_style_token(
    profile: dict[str, Any],
    collection: str,
    token: str | None,
) -> dict[str, Any]:
    styles = profile[collection]
    resolved_token = token or "default"
    if resolved_token not in styles:
        raise ValueError(
            f"Unknown {collection} token {resolved_token!r} "
            f"for style profile {profile['name']!r}"
        )
    return _deep_merge(styles["default"], styles[resolved_token])


def resolve_text_role(
    profile: dict[str, Any],
    role: str,
) -> dict[str, Any]:
    roles = profile["typography"]["roles"]
    if role not in roles:
        raise ValueError(
            f"Unknown typography role {role!r} "
            f"for style profile {profile['name']!r}"
        )
    return deepcopy(roles[role])


def resolve_role_font_size(
    profile: dict[str, Any],
    role: str,
    page_width: float,
    page_height: float,
) -> float:
    del page_width, page_height
    role_style = resolve_text_role(profile, role)
    return float(role_style["fontSizePt"])


def resolve_line_weight(
    profile: dict[str, Any],
    base_weight_pt: float,
    page_width: float,
    page_height: float,
) -> float:
    del profile, page_width, page_height
    return float(base_weight_pt)


def normalize_color(value: str) -> str:
    if not COLOR_PATTERN.fullmatch(value):
        raise ValueError(f"Expected #RRGGBB color, got: {value}")
    return value.upper()


def color_formula(value: str) -> str:
    normalized = normalize_color(value)
    red = int(normalized[1:3], 16)
    green = int(normalized[3:5], 16)
    blue = int(normalized[5:7], 16)
    return f"RGB({red},{green},{blue})"


def char_style_code(font_weight: str, font_style: str) -> int:
    return (1 if font_weight == "bold" else 0) + (
        2 if font_style == "italic" else 0
    )
