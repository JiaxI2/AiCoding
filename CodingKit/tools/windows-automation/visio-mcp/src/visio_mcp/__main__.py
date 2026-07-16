from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from .model import load_json
from .protocol import run_stdio
from .service import VisioService


def main() -> None:
    parser = argparse.ArgumentParser(prog="visio-mcp")
    parser.add_argument(
        "--renderer",
        choices=["auto", "mock", "visio"],
        default=os.environ.get("VISIO_MCP_RENDERER", "auto"),
    )
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("server")
    doctor = sub.add_parser("doctor")
    doctor.add_argument("--json", action="store_true")
    validate = sub.add_parser("validate")
    validate.add_argument("--input", required=True)
    validate.add_argument("--json", action="store_true")
    plan = sub.add_parser("plan")
    plan.add_argument("--input", required=True)
    plan.add_argument("--json", action="store_true")
    render = sub.add_parser("render")
    render.add_argument("--input", required=True)
    render.add_argument("--output", required=True)
    render.add_argument("--visible", action="store_true")
    render.add_argument("--export", default="")
    render.add_argument("--json", action="store_true")
    quality = sub.add_parser("quality")
    quality.add_argument("--image")
    quality.add_argument("--json", action="store_true")
    args = parser.parse_args()

    service = VisioService(renderer=args.renderer)
    if args.cmd == "server":
        run_stdio(service)
        return
    if args.cmd == "doctor":
        result = service.doctor()
    elif args.cmd == "validate":
        result = service.validate(load_json(args.input))
    elif args.cmd == "plan":
        result = service.plan(load_json(args.input))
    elif args.cmd == "render":
        result = service.render(load_json(args.input), args.output, args.visible, True)
        if args.export:
            formats = [item.strip() for item in args.export.split(",") if item.strip()]
            result["export"] = service.export(result["sessionId"], formats, str(Path(args.output).parent))
    elif args.cmd == "quality":
        result = service.quality_check(image=args.image)
    else:
        raise SystemExit(2)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
