# ADR-002: Windows 用户范围 DPAPI 凭证文件

## Status

Accepted

## Context

账号可能使用短 API Key、MiMo Token Plan 专属 API Key，或较大的订阅登录凭证包。`accounts.credential_ref` 已预留，但此前没有实际存储。ADR-001 要求操作系统级保护和数据库中只保留引用，其“OS credential store (DPAPI on Windows)”表述没有确定密文的落盘方式。

Windows Credential Manager 的 generic 凭证单条 `CredentialBlob` 上限为 2560 字节；订阅登录凭证包的长度尚不确定。Windows DPAPI 可以保护字节数据，应用自行保存返回的密文。参见 [CREDENTIAL 结构](https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw)和 [CryptProtectData](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)。

## Decision

- `internal/infrastructure/credential` 保存不透明字节，不识别厂商、认证方式或订阅方案。
- Windows 使用当前用户范围的 DPAPI 加解密，不启用机器范围保护；密文保存在 `<DataDir>/credentials/<profile>/`，文件名由随机 128 位引用生成。
- 单条明文上限为 1 MiB；引用只允许版本前缀和固定长度的小写十六进制，不接受路径。
- 数据库只保存 `credential_ref`。写入返回新引用；读取返回调用方负责管理的明文字节；删除只移除密文文件。
- API Key、Token Plan、订阅方案等信息属于后续账号/提供商元数据。MiMo Token Plan 的凭证仍是其专属 API Key，专属 Base URL 属于提供商配置。订阅登录凭证的具体获取、刷新及可用协议为 `TBD`；订阅名称不自动赋予 API 平台调用权限。

## Alternatives Considered

- Windows Credential Manager：提供原生凭证存储，但单条大小限制会约束未来较大的凭证包。
- 将 DPAPI 密文放入 SQLite：仍可保持明文不入库，但会让现有 `credential_ref` 边界失去意义，也不利于单独管理凭证生命周期。
- 应用自管加密密钥：增加密钥存放和轮换问题，当前无必要。

## Reasoning

DPAPI 用户范围保护满足本地单用户目标；密文文件没有 Credential Manager 的单条大小限制，并能随现有数据目录迁移。固定格式随机引用和配置隔离降低误读其他文件或其他环境的风险。存储层保持通用，后续账号服务可以按厂商协议定义内容和更新策略。

## Consequences

- 本决策细化 ADR-001 的凭证保护要求：实际存储是应用管理的 DPAPI 密文文件，不能称为 Windows Credential Manager 条目。
- 数据目录的密文备份一般不能在另一个 Windows 用户或机器上直接解密；迁移数据目录只保证同用户环境内可继续读取。
- 写入凭证后、保存数据库引用前若失败，账号服务必须删除新文件；未来真实账号流程需要处理孤儿密文清理。
- macOS/Linux 将来需要各自实现，当前返回明确的不支持错误。
