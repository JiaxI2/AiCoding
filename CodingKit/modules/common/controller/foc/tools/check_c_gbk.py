from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
C_PATTERNS = ["src/*.c", "src/*.h", "examples/*.c"]


def decode_c_source(path):
    data = path.read_bytes()
    for encoding in ("utf-8", "gbk"):
        try:
            return data.decode(encoding), encoding
        except UnicodeDecodeError:
            pass
    raise UnicodeDecodeError("utf-8/gbk", data, 0, 1, "not valid UTF-8 or GBK")


def main():
    failures = []
    for pattern in C_PATTERNS:
        for path in ROOT.glob(pattern):
            try:
                text, _ = decode_c_source(path)
            except UnicodeDecodeError as exc:
                failures.append(f"{path}: decode failed: {exc}")
                continue
            if "HU JIAXUAN" not in text:
                failures.append(f"{path}: missing author")
    if failures:
        for item in failures:
            print(item)
        return 1
    print("C source encoding and author check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
