from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from visio_mcp.quality import image_quality


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixtures", required=True)
    args = parser.parse_args()
    results = [{"file": str(path), **image_quality(path)} for path in sorted(Path(args.fixtures).glob("*.png"))]
    report = {
        "count": len(results),
        "minimumScore": min((item["score"] for item in results), default=0),
        "results": results,
    }
    print(json.dumps(report, indent=2))
    raise SystemExit(1 if results and report["minimumScore"] < 70 else 0)


if __name__ == "__main__":
    main()
