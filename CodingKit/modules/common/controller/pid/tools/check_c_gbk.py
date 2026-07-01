from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TARGETS = list((ROOT / "src").glob("*.c")) + list((ROOT / "src").glob("*.h")) + list((ROOT / "examples").glob("*.c"))


def main() -> int:
    failed = []
    for path in TARGETS:
        try:
            path.read_text(encoding="gbk")
        except UnicodeDecodeError as exc:
            failed.append((path, str(exc)))

    if failed:
        for path, error in failed:
            print(f"GBK check failed: {path}: {error}")
        return 1

    print("C 源码 GBK 编码检查通过。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
