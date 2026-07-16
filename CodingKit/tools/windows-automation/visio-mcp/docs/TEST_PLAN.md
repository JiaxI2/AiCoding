# Visio MCP 测试方案

## Smoke

- Diagram schema、style-profile schema、活动 JSON profile 和引用完整性。
- 确定性布局。
- mock render、snapshot、inspect 和 close。
- 官方 MCP client initialize、12 tools/4 resources discovery，并验证不暴露 workflow prompt。
- prompt、logging、tool execution error 和非法 JSON 恢复。

## Full

- 全部单元测试。
- 30 次 mock benchmark。
- 输出目录安全和修复收敛。

## Release

- Visio COM doctor。
- 架构图与紧凑工程控制图两组真实 VSDX 生成。
- PNG、SVG、PDF、quality.json 和 inspection.json 导出。
- visible open、inspect、snapshot 和 close。
- 真实文字块水平/垂直居中。
- 真实 Latin/Asian 字体、显式字号、字重/斜体、文字颜色、框线/填充颜色、线宽和圆角。
- 真实 connector 双端 glue、端口绝对坐标和首末段外向性。
- 固定箭头样式/尺寸/线宽，箭头终端净空和节点碰撞为零。
- PNG 保留标准外边距，`CONTENT_TOUCHES_BORDER` 为零。
- 结束后不新增孤立 `VISIO.EXE`。

## 质量目标

- 结构准确率 100%。
- 节点重叠面积阈值为 0。
- 出界节点为 0。
- 同一 `sizeClass` 宽高差为 0；同轴可统一尺寸差为 0。
- 同一 `sizeClass` 的连续同轴框绝对边界间距差不超过 `0.03 in`。
- 文字/端口所需尺寸不足和非容器无依据过大均为 0。
- 普通节点文字块宽高为 `0.80 +/- 0.02`，菱形约 `0.70`；无依据 override 为 0。
- 节点、caption 和连接线标签字号误差不超过 `0.25 pt`；同一
  `fontRole + sizeClass` 的 live 字号跨度不超过 `0.25 pt`；低于角色可读下限为 0。
- 字体、字形样式、文字/框线/填充颜色、线宽和圆角 mismatch 均为 0；
  线宽误差不超过 `0.10 pt`，圆角误差不超过 `0.005 in`。
- caption 与连接线标签锚点误差分别不超过 `0.02 in` 和 `0.03 in`。
- 紧凑模式页面利用率横向至少 75%、纵向至少 65%，同轴间距不超过 `max(0.8 in, 1.5 * nodeGap)`。
- 端点向内、箭头终端不足、箭头覆盖节点和未校准箭头几何均为 0。
- text/line、text/shape、text/text collision 和 connector crossing 均为 0。
- mock validate/plan p95 小于 100 ms。
- mock render p95 小于 1 s。
