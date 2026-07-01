# 编码说明

本 kit 为兼容部分 Windows/CCS 嵌入式工程，C 源码使用 GBK：

- `src/*.c`：GBK
- `src/*.h`：GBK
- `examples/*.c`：GBK

非 C 文件使用 UTF-8：

- `README.md`
- `docs/*.md`
- `tests/*.py`
- `tools/*.py`
- `simulink/*.m`

检查命令：

```bash
python tools/check_c_gbk.py
```
