# TASK-20261014-017: 接入 Kimi 开放平台模型与 API

## Status

Completed

## Created

2026-10-14（Asia/Singapore）

## Request

用户要求接入 Kimi 模型，并参考 https://platform.kimi.com/docs/get-api-key 官方文档。

## Goal

将既有 Kimi 展示预设接为可调用的普通 API Key 提供商，完成官方模型、目录、路由、原生协议、思考／工具／流式及用量的最小闭环。

## Scope

核对官方认证与端点、当前模型与限制、协议与续接语义；复用现有账号／DPAPI／网关／目录及用量，补充分支和必要验证、文档及提交。

## Out of Scope

套餐购买、Kimi Code 订阅账号、多模态／文件／Batch 独立能力；真实 Key 尚未提供，不发送付费推理或改用户客户端配置。

# Analysis

main 基线 `9451f11`。已读产品、架构、技术栈、编码及已有开发／验证规范。已有 `builtin_kimi` 和图标，但只有展示预设；普通 API 的国内与海外 Key 不互通。增加账号级 cn／global 与配置版本，默认国内；旧空地址按原 `.ai` 语义迁移为海外，未知地址保留并提示确认，不按 Key 外观猜平台。

官方当前四款模型为 `kimi-k3`、`kimi-k2.7-code`、`kimi-k2.7-code-highspeed`、`kimi-k2.6`；K2.5 与 Moonshot V1 已下线。Chat 均可用；Responses 当前明确仅 K3，Messages 参考声明 K3，Claude 集成文档另明确 K2.7 Code／K2.6，按已确认能力开放。原生协议保留思考和工具上下文，K3 原生 Responses 支持 namespace 与 apply_patch custom 工具，不能套用只允许 function 的其他适配器。K3 effort 是 low／high／max；K2 系列思考配置有独立约束。管理目录提供 context_length，但成功目录仍不代表真实长窗口验证。

方案：复用 Step 的账号级平台隔离思路，新增独立 Kimi 选项表和固定官方域名解析；原生文本调用和有界目录发现接入现有 DPAPI／缓存／路由，自动生成国内／海外通道别名并拒绝裸名歧义。余额虽有官方接口，本任务仅接模型，保持未接入状态；不将 Code 模型当作订阅认证。

# Implementation Plan

1. 阅读官方模型、API、思考／工具及目录文档，核对地区选择与协议能力。
2. 基于现有标准库原生适配器，实施必要账号／目录／路由／网关接入。
3. 更新 UI 预设、模型同步与接入检查；校验兼容、边界和版本失效。
4. 运行针对验证与 `task verify`，检查实际开发窗口；记录真实 Key 限制。
5. 同步文档、检查差异、提交并归档。

# Implementation Result

- 增加 Kimi 标准库原生适配器、四款现役模型／标称上下文／推理元数据和有界模型发现。Chat 四款，Messages K3／K2.7 Code／K2.6，Responses K3；高速版 Messages 保持未确认。原生保留推理历史、工具关联、namespace、apply_patch custom grammar 和 SSE。
- 迁移 023 保存账号平台与版本、兼容旧海外默认，扩展目录版本视图；复用 DPAPI、事务、凭据轮换及并发版本检查。切平台失效目录和已解析请求，不跨平台重试。
- 接入路由、平台别名、真实模型目录／客户端能力过滤、接入检查及当次地区／缓存用量。账号新增／批量／编辑增加平台选择和标识；Kimi Code 订阅与余额保持未接入。
- 为遵循文件大小与业务逻辑边界，将账号扫描移至 metadata.go、目录同步移至 sync.go、编辑表单业务状态移至 use-account-edit-form.ts；保留既有外部行为并回归 Step 及其他账号路径。
- 同步 PRD、OVERVIEW、TECH_STACK、CURRENT，新增 `product/KIMI.md`，重新生成 TypeScript 绑定；无新增依赖或用户客户端配置变更。

# Verification

- PASS：provider 契约覆盖两地区三协议、推理／签名／工具历史、K3 custom／namespace、错误不跨区重试／重定向、参数与模态／状态边界、目录校验和真正增量 SSE／取消。
- PASS：账号平台并发、非法输入／Key、批量回滚与凭据清理；兼容旧默认／显式国内／未知地址迁移；目录同步期间切平台不写旧结果。
- PASS：两地区网关三协议集成、别名改写、固定受保护 Key、混合平台歧义、在途请求拒绝切区、缓存输入口径、K2 不进入 Responses 目录。
- PASS：4 项新前端测试覆盖新账号默认国内、海外整批提交、编辑保留空 Key／版本、未知平台提示；已有账号、Step、模型目录等 19 项针对回归通过。
- PASS：首轮 `task verify BIN_DIR=tmp/kimi-20261014/verify-bin`，所有 Go 测试、73 个文件／401 项前端测试、格式／Lint／类型检查及生产构建通过，无新增 Lint 警告。既有缺 uname／tail、jsdom scrollTo 和构建大包提示不影响结果。
- PASS：原生项目开发窗口 PID 21976，实际路径 `E:/go/new/wails3-v/OmniProxy/bin/omniproxy.exe`；国内默认、海外选择、单个／批量、深浅主题核对并逐张查看截图，恢复原暗色主题，未输入或保存 Key。证据位于忽略目录 `tmp/kimi-20261014/`。
- PASS：最后核对后将高速版未确认的 Messages 从能力及客户端目录中排除并增加回归断言；最终 `task verify BIN_DIR=tmp/kimi-20261014/final-verify-bin` 所有 Go 测试、73 个文件／401 项前端测试、格式／Lint／类型检查及生产构建全部通过。无新增 Lint 警告。

# Documentation

官方资料来自用户指定页、同一官方站点的 Markdown／OpenAPI，以及国际站模型／API 概述；关键依据列于 `product/KIMI.md`。仅文本接入的范围、未确认协议与真实账号验证边界均明确。

# Remaining Risks

本轮无真实 Kimi Key、真实 CLI 工具循环或长输入验证；成功预设／契约测试不代表当前账号权限。高速版 Messages、K2 Responses、多模态／文件／Batch／Code 订阅及余额不声明已接入。

# Deviations

从原全球单一预设扩展为账号级平台选择，依据官方 Key 不互通要求且保留原账号语义；最后只开放高速版已明确确认的 Chat。沿用现有账号级隔离模式，不增加新的架构层或依赖。

# Git

`a20b0ab` — `feat(kimi): 接入两平台模型与原生 API`。

三份原有无关未跟踪任务记录保持原样，未纳入提交；没有推送远程。

# Result

Kimi 普通 API Key 模型接入完成，当前四款模型与两平台配置可用，原生调用按已确认协议开放；最终质量门与原生界面验证通过，真实 Key／长窗口保持待验收。

# Completed

2026-10-14（Asia/Singapore）。
