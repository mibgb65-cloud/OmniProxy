package proxy

// AccountCandidate 仅由受信路由填充，正文无法声明备用账号。
type AccountCandidate struct{ AccountID, UpdatedAt string }
