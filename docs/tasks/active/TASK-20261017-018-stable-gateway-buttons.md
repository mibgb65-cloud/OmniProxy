# TASK-20261017-018: 网关按钮状态切换保持尺寸稳定

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

点击启动网关按钮时会闪动，希望内部动画、文案变化不影响按钮自身大小。

## Analysis

GatewayHeader 按钮宽度由当前文案决定，启停过程和刷新过程更换不同长度的文本，触发尺寸变化；通用 Button 的 transition-all 还会对几何变化产生过渡。

## Implementation Plan

1. 在网关页局部为全部状态文案预留同一尺寸，保留中英文与字号适配，图标继续使用固定尺寸。
2. 将这两个按钮的过渡限定为颜色、透明度等视觉属性，不修改全局 Button 或启停逻辑。
3. 验证延迟启停／刷新、原生按钮与相邻按钮位置，运行完整质量门并提交归档。

## Implementation Result

- 启停按钮为启动、停止、正在启动／停止四种文案共同预留最大宽度，刷新按钮为普通／进行中两种文案共同预留宽度；使用同一网格单元叠放，不硬编码像素宽度，语言和字号仍自适应。
- 隐藏占位文案设置 aria-hidden，屏幕阅读器只读取当前状态；增加 aria-busy，保留原禁用和重复提交保护。
- 仅对网关页这两个按钮覆盖过渡属性，限制为颜色、边框、阴影和透明度，排除尺寸；固定图标尺寸及减少动态效果行为继续沿用现有实现。没有修改全局按钮、网关控制逻辑或接口。

## Verification

- PASS：中英文延迟启停新用例覆盖同一按钮节点、当前可访问名称、aria-busy、禁用和防止重复提交；网关运行／侧栏共 2 个文件／20 项定向测试及 TypeScript 通过。
- PASS：正常项目开发版原生窗口连续采样 312 次，实际捕获启动网关／正在启动／停止网关／正在停止和普通／进行中刷新状态。启停按钮始终 116×36，刷新按钮始终 134×36，水平位置始终分别为 2218／2342，全部变化为 0px。验收后恢复网关初始停止状态，未发送模型请求。
- PASS：`VITEST_MAX_WORKERS=4 task --force verify BIN_DIR=tmp/stable-gateway-buttons-20261017/verify-bin`，退出 0。格式、Go vet／staticcheck／ESLint、TypeScript、Go 测试、93 个文件／522 项前端测试及 Windows 生产构建通过。仅有既有 scrollTo 测试环境及 Vite chunk 体积提示。
- PASS：`git diff --check`；只修改本任务文件，保留开始前五份无关未跟踪记录。日志与原生采样结果位于忽略目录 tmp/stable-gateway-buttons-20261017/。

## Documentation

仅同步任务记录与 CURRENT；产品流程、API 和技术栈未改变，无需更新其他规范或 ADR。

## Remaining Risks

语言或全局字号切换时，按钮仍按新的排版尺寸自适应；同一语言和字号下的运行状态切换保持尺寸稳定。

## Git

待提交。

## Result

网关启停／刷新按钮状态变化不再改变尺寸，开发版已更新，原生几何采样与完整质量门通过。
