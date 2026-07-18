from __future__ import annotations

import argparse
import importlib
import json
import sys
from typing import Any
import winreg


def check_import(module: str) -> dict[str, Any]:
    try:
        importlib.import_module(module)
        return {"name": f"import:{module}", "ok": True}
    except Exception as exc:
        return {"name": f"import:{module}", "ok": False, "error": str(exc)}


def check_powerpoint_registration() -> dict[str, Any]:
    try:
        with winreg.OpenKey(
            winreg.HKEY_CLASSES_ROOT,
            r"PowerPoint.Application\CLSID",
        ) as key:
            clsid, _ = winreg.QueryValueEx(key, None)
        return {
            "name": "powerpoint-com-registration",
            "ok": bool(clsid),
            "clsid": clsid,
        }
    except OSError as exc:
        return {
            "name": "powerpoint-com-registration",
            "ok": False,
            "error": str(exc),
        }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--json", action="store_true")
    parser.parse_args()

    checks = [
        {"name": "platform:windows", "ok": sys.platform == "win32"},
        check_import("mcp"),
        check_import("pydantic"),
        check_import("pythoncom"),
        check_import("win32com.client"),
        check_import("src.server"),
        check_powerpoint_registration(),
    ]
    result = {"ok": all(check["ok"] for check in checks), "checks": checks}
    print(json.dumps(result, ensure_ascii=False))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
