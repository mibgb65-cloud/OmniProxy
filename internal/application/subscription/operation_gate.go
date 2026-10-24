package subscription

import "omniproxy/internal/core/syncgate"

// 登录提交使用全局排他，已保存账号的读取／续期／注销使用账号级互斥。
type operationGate = syncgate.Gate
