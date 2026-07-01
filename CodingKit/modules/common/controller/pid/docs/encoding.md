# 编码说明

## C 源码编码

`src/*.c`、`src/*.h` 和 `examples/*.c` 使用 GBK 编码。

原因：当前模板源码包含中文注释，并面向 Windows / CCS / TI 嵌入式工程兼容使用。

## 其他文件编码

README、docs、tests、tools、simulink 脚本使用 UTF-8。

## Agent 修改要求

后续 Agent 修改 C 文件时必须：

1. 按 GBK 读取；
2. 按 GBK 写回；
3. 不自动转换为 UTF-8；
4. 不在 C 源码中引入第三方项目名或上层业务语义。

可运行：

```bash
python tools/check_c_gbk.py
```
