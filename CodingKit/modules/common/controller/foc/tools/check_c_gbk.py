from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
C_PATTERNS = ["src/*.c", "src/*.h", "examples/*.c"]


def main():
    failures = []
    for pattern in C_PATTERNS:
        for path in ROOT.glob(pattern):
            data = path.read_bytes()
            try:
                text = data.decode("gbk")
            except UnicodeDecodeError as exc:
                failures.append(f"{path}: GBK decode failed: {exc}")
                continue
            if "HU JIAXUAN" not in text:
                failures.append(f"{path}: missing author")
            try:
                data.decode("utf-8")
                failures.append(f"{path}: should not be valid UTF-8")
            except UnicodeDecodeError:
                pass
    if failures:
        for item in failures:
            print(item)
        return 1
    print("C source GBK check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
